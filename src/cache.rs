//! Part of the issue is that we need to fetch a ton of files. Some of them upwards of 1.3GB.
//! Attempt to cache as many of them between runs as possible.

use crate::env::var;
use log::{debug, error, info, trace, warn};
use std::collections::{BTreeMap, BinaryHeap};
use std::fs::{create_dir_all, read_dir, File};
use std::io::ErrorKind::{InvalidInput, NotFound, PermissionDenied};
use std::io::{BufReader, BufWriter, Error, ErrorKind, Read, Write};
use std::num::Wrapping;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::SystemTime;
use std::{fs, io};
use ureq::{Agent, AgentBuilder};

const USER_AGENT: &str = "Mozilla/5.0 (compatible; WPIFastlyMQPBot/1.0; +https://www.wpi.edu/academics/undergraduate/major-qualifying-project)";

pub struct PersistentCache {
    path: PathBuf,
    cache_usage: u64,
    size_limit: u64,
    entries: Vec<CacheEntry>,
    agent: Agent,
}

impl PersistentCache {
    pub fn from_env() -> io::Result<Self> {
        let path = PathBuf::from(var("cache_location"));
        let cache_size = var("cache_size");

        let size_limit = match parse_byte_size(&cache_size) {
            Some(v) => v,
            None => {
                error!("Got invalid cache size: {:?}", &cache_size);
                return Err(Error::from(InvalidInput));
            }
        };

        info!("Using folder for http cache: {}", path.display());
        info!("Using http cache size: {}B", size_limit);

        if !path.is_dir() {
            create_dir_all(&path)?;
        }

        let mut entries = Vec::new();
        let mut cache_usage = 0;

        for entry in read_dir(&path)? {
            let entry = entry?;
            let metadata = entry.metadata()?;

            if !metadata.is_file() {
                error!(
                    "Found non-file entry in cache folder: {}",
                    entry.path().display()
                );
                return Err(Error::from(InvalidInput));
            }

            cache_usage += metadata.len();
            entries.push(CacheEntry {
                path: entry.path(),
                size: metadata.len(),
                last_accessed: metadata.accessed()?,
                usage_arc: Arc::new(()),
            });
        }

        Ok(PersistentCache {
            path,
            size_limit,
            cache_usage,
            entries,
            agent: AgentBuilder::new().user_agent(USER_AGENT).build(),
        })
    }

    pub fn clear_bak_files(&mut self) {
        let mut files_to_clear = Vec::new();
        let mut bytes_to_clear = 0;

        let mut index = 0;
        while index < self.entries.len() {
            if self.entries[index].path.ends_with(".bak") {
                bytes_to_clear += self.entries[index].size;
                files_to_clear.push(self.entries.remove(index));
                continue;
            }
            index += 1;
        }

        if files_to_clear.is_empty() {
            return;
        }

        info!(
            "Found {} old bak files to clear ({} bytes)",
            files_to_clear.len(),
            bytes_to_clear
        );
        for entry in files_to_clear {
            match fs::remove_file(&entry.path) {
                Ok(_) => info!("Removed file: {}", entry.path.display()),
                Err(e) => {
                    error!("Unable to remove {} due to {:?}", entry.path.display(), &e);
                    self.entries.push(entry);
                    continue;
                }
            }
            self.cache_usage -= entry.size;
        }
    }

    /// Evict existing entries if needed to get the required amount of space. If the requires space
    /// could not be satisfied, false is returned instead
    pub fn evict_space_for(&mut self, required: u64) -> bool {
        if self.size_limit - self.cache_usage >= required {
            return true;
        } else if required > self.size_limit {
            return false;
        }

        self.entries.sort_by_key(|x| x.last_accessed);

        let mut index = 0;
        while index < self.entries.len() && self.size_limit - self.cache_usage < required {
            if Arc::strong_count(&self.entries[index].usage_arc) > 1 {
                index += 1;
                continue;
            }

            // Refresh entry timestamp
            let previous_access_time = self.entries[index].last_accessed;
            let new_access_time = match self.entries[index]
                .path
                .metadata()
                .and_then(|x| x.accessed())
            {
                Ok(x) => x,
                // File is still there, but we don't have access to it. Its probably fine so we can
                // try again next time.
                Err(e) if e.kind() == PermissionDenied => {
                    index += 1;
                    continue;
                }
                // File no longer exists or we no longer have control over it so we can remove it
                // from the entries
                Err(e) => {
                    warn!(
                        "Dropping cache entry {}: {:?}",
                        self.entries[index].path.display(),
                        e
                    );
                    self.cache_usage -= self.entries[index].size;
                    self.entries.remove(index);
                    continue;
                }
            };

            // If access time updated, add new time and resort the entries. This entry should only
            // move to an index the same as or later in the list assuming no one is messing with the
            // files. If they are then whatever, the cache doesn't need to be perfect.
            if new_access_time != previous_access_time {
                self.entries[index].last_accessed = new_access_time;
                self.entries.sort_by_key(|x| x.last_accessed);
                continue;
            }

            // We have now confirmed this is the oldest entry and it is not in use
            debug!(
                "Evicting cache entry: {}",
                self.entries[index].path.display()
            );
            if let Err(e) = fs::remove_file(&self.entries[index].path) {
                if e.kind() != NotFound {
                    warn!(
                        "Failed to evict cache entry {}: {:?}",
                        self.entries[index].path.display(),
                        e
                    );
                    index += 1;
                }
            }

            let entry = self.entries.remove(index);
            self.cache_usage -= entry.size;
        }

        self.size_limit - self.cache_usage > required
    }

    // pub fn update_cache(&self, url: &str) -> io::Result<PathBuf> {
    //     let file_name = self.path.join(cache_file_for(url));
    //     let mut bak_file = file_name.as_os_str().to_owned();
    //     bak_file.push(".bak");
    //
    //     let response = match self.agent.get(url).call() {
    //         Ok(x) => x,
    //         Err(e) => return Err(Error::new(ErrorKind::Other, e))
    //     };
    //
    //     let length = response.header("Content-Length").and_then(|x| x.parse::<u64>().ok());
    //
    //     let mut reader = BufReader::new(response.into_reader());
    //     let mut file = BufWriter::new(File::create(&bak_file)?);
    //
    //     loop {
    //         let buffer = reader.fill_buf()?;
    //         if buffer.is_empty() {
    //             break
    //         }
    //
    //         let bytes_read = file.write(buffer)?;
    //         reader.consume(bytes_read);
    //     }
    //
    //     file.flush()?;
    //     drop((file, reader));
    //
    //     fs::rename(&bak_file, &file_name)?;
    //     Ok(file_name)
    // }

    #[cold]
    pub fn get_cache_slow_path(&mut self, url: &str) -> io::Result<Box<dyn 'static + Read>> {
        let file_name = self.path.join(cache_file_for(url));
        let mut bak_file = file_name.as_os_str().to_owned();
        bak_file.push(".bak");

        let response = match self.agent.get(url).call() {
            Ok(x) => x,
            Err(e) => return Err(Error::new(ErrorKind::Other, e)),
        };

        let length = match response
            .header("Content-Length")
            .and_then(|x| x.parse::<u64>().ok())
        {
            Some(length) if self.evict_space_for(length) => length,
            // Space could not be cleared for the cache entry so bypass the cache and return the
            // http stream instead
            _ => return Ok(response.into_reader()),
        };

        let mut reader = response.into_reader();
        let mut file = File::create(&bak_file)?;

        // Copy data to bak file
        let result = io::copy(&mut reader, &mut file)
            .and_then(|_| file.flush())
            .and_then(|_| {
                // Close http response and bak file
                drop((file, reader));

                // Rename bak file to intended name
                fs::rename(&bak_file, &file_name)
            });

        if let Err(e) = result {
            warn!("Got error when fetching page {}: {:?}", url, &e);

            if let Err(e) = fs::remove_file(&bak_file) {
                if e.kind() != NotFound {
                    error!(
                        "Failed to remove bak file {} after failed read: {:?}",
                        bak_file.to_string_lossy(),
                        e
                    );
                }
            }

            return Err(e);
        }

        // Create final reader
        let ret = FileArcInstance {
            file: File::open(&file_name)?,
            _arc: Arc::new(()),
        };

        // Update cache state
        self.cache_usage += length;
        self.entries.push(CacheEntry {
            path: file_name,
            size: length,
            last_accessed: SystemTime::now(),
            usage_arc: ret._arc.clone(),
        });

        Ok(Box::new(ret))
    }

    fn has_cache_entry<P: AsRef<Path>>(&self, path: P) -> Option<Arc<()>> {
        self.entries
            .iter()
            .filter(|x| x.path == path.as_ref())
            .map(|x| x.usage_arc.clone())
            .next()
    }

    pub fn get(&mut self, url: &str) -> io::Result<Box<dyn 'static + Read>> {
        let file_name = self.path.join(cache_file_for(url));

        // The "hot" path of this cache.
        if let Some(usage_arc) = self.has_cache_entry(&file_name) {
            trace!("Got cache hit for {}", url);
            return Ok(Box::new(FileArcInstance {
                file: File::open(file_name)?,
                _arc: usage_arc,
            }));
        }

        trace!("Got cache miss for {}", url);
        self.get_cache_slow_path(url)
    }
}

struct CacheEntry {
    path: PathBuf,
    size: u64,
    last_accessed: SystemTime,
    /// Use an Arc as a an atomic reference count of readers for a given file
    usage_arc: Arc<()>,
}

fn fnv_hash(bytes: &[u8]) -> u64 {
    const FNV_64_PRIME: u64 = 0x100000001b3;

    let mut hash: Wrapping<u64> = Wrapping(0);
    for byte in bytes {
        hash *= FNV_64_PRIME;
        hash ^= *byte as u64;
    }

    hash.0
}

fn parse_byte_size(mut input: &str) -> Option<u64> {
    const BYTE_SUFFIX: [(u64, &str); 9] = [
        (1, "B"),
        (1 << 10, "KB"),
        (1 << 10, "KiB"),
        (1 << 20, "MB"),
        (1 << 20, "MiB"),
        (1 << 30, "GB"),
        (1 << 30, "GiB"),
        (1 << 40, "TB"),
        (1 << 40, "TiB"),
    ];

    let mut size = 1;
    for (multiplier, suffix) in BYTE_SUFFIX.into_iter().rev() {
        if input.len() > suffix.len()
            && input[input.len() - suffix.len()..].eq_ignore_ascii_case(suffix)
        {
            input = input[..input.len() - suffix.len()].trim();
            size = multiplier;
            break;
        }
    }

    if let Ok(x) = input.parse::<u64>() {
        return Some(size * x);
    }

    match input.parse::<f64>() {
        // If size is 1 do not allow fractional bytes
        Ok(x) if size == 1 => ((x as u64) as f64 == x).then(|| x as u64),
        Ok(x) => Some((size as f64 * x) as u64),
        Err(_) => None,
    }
}

fn cache_file_for(mut address: &str) -> String {
    address = address.trim().trim_matches('/');

    for prefix in ["http://", "https://", "www."] {
        if let Some(remaining) = address.strip_prefix(prefix) {
            address = remaining;
        }
    }

    let hash = fnv_hash(address.as_bytes());

    if let Some(path_sep) = address.rfind('/') {
        address = &address[path_sep + 1..];
    }

    while address.len() > 128 {
        let char_offset = address
            .char_indices()
            .map(|(x, _)| x)
            .skip(1)
            .next()
            // Its not like it is valid utf8 anyway, so take off a single byte
            .unwrap_or(1);
        address = &address[char_offset..];
    }

    format!("{:X}_{}", hash, address)
}

/// Since we open multiple read-only file descriptors we can't use a simple Arc<File>.
struct FileArcInstance {
    file: File,
    _arc: Arc<()>,
}

impl Read for FileArcInstance {
    fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
        self.file.read(buf)
    }
}
