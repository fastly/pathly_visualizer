//! Part of the issue is that we need to fetch a ton of files. Some of them upwards of 1.3GB.
//! Attempt to cache as many of them between runs as possible.

use crate::env::var;
use crate::HumanBytes;
use log::{debug, error, info, trace, warn};
use std::collections::{BTreeMap, BinaryHeap};
use std::ffi::OsString;
use std::fs::{create_dir_all, read_dir, File};
use std::hint::spin_loop;
use std::io::ErrorKind::{InvalidInput, NotFound, PermissionDenied};
use std::io::{BufReader, BufWriter, Error, ErrorKind, Read, Write};
use std::num::Wrapping;
use std::path::{Path, PathBuf};
use std::sync::atomic::AtomicU64;
use std::sync::atomic::Ordering::SeqCst;
use std::sync::{Arc, Mutex};
use std::time::SystemTime;
use std::{fs, io};
use ureq::{Agent, AgentBuilder};

const USER_AGENT: &str = "Mozilla/5.0 (compatible; WPIFastlyMQPBot/1.0; +https://www.wpi.edu/academics/undergraduate/major-qualifying-project)";

pub struct PersistentCache {
    path: PathBuf,
    ready_count: AtomicU64,
    inner: Mutex<CacheInnerData>,
    agent: Agent,
}

struct CacheInnerData {
    cache_usage: u64,
    size_limit: u64,
    entries: Vec<CacheEntry>,
}

impl CacheInnerData {
    fn from_env(path: &PathBuf) -> io::Result<Self> {
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
                ready: true,
            });
        }

        info!(
            "Loaded cache with {} existing entries ({}/{})",
            entries.len(),
            HumanBytes(cache_usage),
            HumanBytes(size_limit)
        );

        Ok(CacheInnerData {
            size_limit,
            cache_usage,
            entries,
        })
    }

    fn cache_space(&self) -> (u64, u64) {
        (self.cache_usage, self.size_limit)
    }

    fn clear_bak_files(&mut self) {
        let mut files_to_clear = Vec::new();
        let mut bytes_to_clear = 0;

        let mut index = 0;
        while index < self.entries.len() {
            if self.entries[index]
                .path
                .as_os_str()
                .to_string_lossy()
                .ends_with(".bak")
            {
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
            "Found {} old bak files to clear ({})",
            files_to_clear.len(),
            HumanBytes(bytes_to_clear)
        );
        for entry in files_to_clear {
            match fs::remove_file(&entry.path) {
                Ok(_) => debug!("Removed file: {}", entry.path.display()),
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
    fn evict_space_for(&mut self, required: u64) -> bool {
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
            trace!(
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

    fn add_entry_stub(&mut self, path: PathBuf) -> Arc<()> {
        let usage_arc = Arc::new(());

        self.entries.push(CacheEntry {
            path,
            size: 0,
            last_accessed: SystemTime::now(),
            usage_arc: usage_arc.clone(),
            ready: false,
        });

        usage_arc
    }

    fn remove_entry_stub(&mut self, path: &PathBuf) {
        self.entries.retain(|x| {
            if &x.path == path {
                self.cache_usage -= x.size;
                return false;
            }
            true
        });
    }

    fn complete_entry_stub(&mut self, path: &PathBuf) {
        self.entries
            .iter_mut()
            .filter(|x| &x.path == path)
            .for_each(|x| x.ready = true);
    }

    fn set_stub_length(&mut self, path: &PathBuf, size: u64) -> bool {
        if !self.evict_space_for(size) {
            self.remove_entry_stub(path);
            return false;
        }

        self.cache_usage += size;
        self.entries
            .iter_mut()
            .filter(|x| &x.path == path)
            .for_each(|x| x.size = size);
        true
    }

    fn has_cache_entry<P: AsRef<Path>>(&self, path: P) -> Option<(Arc<()>, bool)> {
        self.entries
            .iter()
            .filter(|x| x.path == path.as_ref())
            .map(|x| (x.usage_arc.clone(), x.ready))
            .next()
    }

    fn is_entry_ready<P: AsRef<Path>>(&self, path: P) -> Option<bool> {
        self.entries
            .iter()
            .filter(|x| x.path == path.as_ref())
            .map(|x| x.ready)
            .next()
    }
}

impl PersistentCache {
    pub fn from_env() -> io::Result<Self> {
        let path = PathBuf::from(var("cache_location"));
        // let cache_size = var("cache_size");
        //
        // let size_limit = match parse_byte_size(&cache_size) {
        //     Some(v) => v,
        //     None => {
        //         error!("Got invalid cache size: {:?}", &cache_size);
        //         return Err(Error::from(InvalidInput));
        //     }
        // };
        //
        // info!("Using folder for http cache: {}", path.display());
        // info!("Using http cache size: {}B", size_limit);
        //
        // if !path.is_dir() {
        //     create_dir_all(&path)?;
        // }
        //
        // let mut entries = Vec::new();
        // let mut cache_usage = 0;
        //
        // for entry in read_dir(&path)? {
        //     let entry = entry?;
        //     let metadata = entry.metadata()?;
        //
        //     if !metadata.is_file() {
        //         error!(
        //             "Found non-file entry in cache folder: {}",
        //             entry.path().display()
        //         );
        //         return Err(Error::from(InvalidInput));
        //     }
        //
        //     cache_usage += metadata.len();
        //     entries.push(CacheEntry {
        //         path: entry.path(),
        //         size: metadata.len(),
        //         last_accessed: metadata.accessed()?,
        //         usage_arc: Arc::new(()),
        //         ready: true,
        //     });
        // }
        //
        // info!("Loaded cache with {} existing entries ({}/{})", entries.len(), HumanBytes(cache_usage), HumanBytes(size_limit));

        Ok(PersistentCache {
            ready_count: AtomicU64::new(0),
            inner: Mutex::new(CacheInnerData::from_env(&path)?),
            agent: AgentBuilder::new().user_agent(USER_AGENT).build(),
            path,
            // size_limit,
            // cache_usage,
            // entries,
        })
    }

    pub fn cache_space(&self) -> (u64, u64) {
        self.inner.lock().unwrap().cache_space()
        // (self.cache_usage, self.size_limit)
    }

    pub fn clear_bak_files(&mut self) {
        self.inner.lock().unwrap().clear_bak_files();
        // let mut files_to_clear = Vec::new();
        // let mut bytes_to_clear = 0;
        //
        // let mut index = 0;
        // while index < self.entries.len() {
        //     if self.entries[index].path.as_os_str().to_string_lossy().ends_with(".bak") {
        //         bytes_to_clear += self.entries[index].size;
        //         files_to_clear.push(self.entries.remove(index));
        //         continue;
        //     }
        //     index += 1;
        // }
        //
        // if files_to_clear.is_empty() {
        //     return;
        // }
        //
        // info!(
        //     "Found {} old bak files to clear ({})",
        //     files_to_clear.len(),
        //     HumanBytes(bytes_to_clear)
        // );
        // for entry in files_to_clear {
        //     match fs::remove_file(&entry.path) {
        //         Ok(_) => debug!("Removed file: {}", entry.path.display()),
        //         Err(e) => {
        //             error!("Unable to remove {} due to {:?}", entry.path.display(), &e);
        //             self.entries.push(entry);
        //             continue;
        //         }
        //     }
        //     self.cache_usage -= entry.size;
        // }
    }

    /// Evict existing entries if needed to get the required amount of space. If the requires space
    /// could not be satisfied, false is returned instead
    fn evict_space_for(&mut self, required: u64) -> bool {
        self.inner.lock().unwrap().evict_space_for(required)
        // if self.size_limit - self.cache_usage >= required {
        //     return true;
        // } else if required > self.size_limit {
        //     return false;
        // }
        //
        // self.entries.sort_by_key(|x| x.last_accessed);
        //
        // let mut index = 0;
        // while index < self.entries.len() && self.size_limit - self.cache_usage < required {
        //     if Arc::strong_count(&self.entries[index].usage_arc) > 1 {
        //         index += 1;
        //         continue;
        //     }
        //
        //     // Refresh entry timestamp
        //     let previous_access_time = self.entries[index].last_accessed;
        //     let new_access_time = match self.entries[index]
        //         .path
        //         .metadata()
        //         .and_then(|x| x.accessed())
        //     {
        //         Ok(x) => x,
        //         // File is still there, but we don't have access to it. Its probably fine so we can
        //         // try again next time.
        //         Err(e) if e.kind() == PermissionDenied => {
        //             index += 1;
        //             continue;
        //         }
        //         // File no longer exists or we no longer have control over it so we can remove it
        //         // from the entries
        //         Err(e) => {
        //             warn!(
        //                 "Dropping cache entry {}: {:?}",
        //                 self.entries[index].path.display(),
        //                 e
        //             );
        //             self.cache_usage -= self.entries[index].size;
        //             self.entries.remove(index);
        //             continue;
        //         }
        //     };
        //
        //     // If access time updated, add new time and resort the entries. This entry should only
        //     // move to an index the same as or later in the list assuming no one is messing with the
        //     // files. If they are then whatever, the cache doesn't need to be perfect.
        //     if new_access_time != previous_access_time {
        //         self.entries[index].last_accessed = new_access_time;
        //         self.entries.sort_by_key(|x| x.last_accessed);
        //         continue;
        //     }
        //
        //     // We have now confirmed this is the oldest entry and it is not in use
        //     debug!(
        //         "Evicting cache entry: {}",
        //         self.entries[index].path.display()
        //     );
        //     if let Err(e) = fs::remove_file(&self.entries[index].path) {
        //         if e.kind() != NotFound {
        //             warn!(
        //                 "Failed to evict cache entry {}: {:?}",
        //                 self.entries[index].path.display(),
        //                 e
        //             );
        //             index += 1;
        //         }
        //     }
        //
        //     let entry = self.entries.remove(index);
        //     self.cache_usage -= entry.size;
        // }
        //
        // self.size_limit - self.cache_usage > required
    }

    // #[cold]
    // pub fn get_cache_slow_path(&mut self, url: &str) -> io::Result<Box<dyn 'static + Read>> {
    //     let file_name = self.path.join(cache_file_for(url));
    //     let mut bak_file = file_name.as_os_str().to_owned();
    //     bak_file.push(".bak");
    //
    //     let response = match self.agent.get(url).call() {
    //         Ok(x) => x,
    //         Err(e) => return Err(Error::new(ErrorKind::Other, e)),
    //     };
    //
    //     let length = match response
    //         .header("Content-Length")
    //         .and_then(|x| x.parse::<u64>().ok())
    //     {
    //         Some(length) if self.evict_space_for(length) => length,
    //         // Space could not be cleared for the cache entry so bypass the cache and return the
    //         // http stream instead
    //         _ => return Ok(response.into_reader()),
    //     };
    //
    //
    //     let mut reader = response.into_reader();
    //     let mut file = File::create(&bak_file)?;
    //     let usage_arc = Arc::new(());
    //
    //     // Update cache state
    //     self.cache_usage += length;
    //     self.entries.push(CacheEntry {
    //         path: file_name.to_owned(),
    //         size: length,
    //         last_accessed: SystemTime::now(),
    //         usage_arc: usage_arc.clone(),
    //         ready: false,
    //     });
    //
    //
    //     // Copy data to bak file
    //     let result = io::copy(&mut reader, &mut file)
    //         .and_then(|_| file.flush())
    //         .and_then(|_| {
    //             // Close http response and bak file
    //             drop((file, reader));
    //
    //             // Rename bak file to intended name
    //             fs::rename(&bak_file, &file_name)
    //         });
    //
    //     if let Err(e) = result {
    //         warn!("Got error when fetching page {}: {:?}", url, &e);
    //
    //         if let Err(e) = fs::remove_file(&bak_file) {
    //             if e.kind() != NotFound {
    //                 error!(
    //                     "Failed to remove bak file {} after failed read: {:?}",
    //                     bak_file.to_string_lossy(),
    //                     e
    //                 );
    //             }
    //         }
    //
    //         return Err(e);
    //     }
    //
    //
    //     // Create final reader
    //     let ret = FileArcInstance {
    //         file: File::open(&file_name)?,
    //         _arc: usage_arc,
    //     };
    //
    //     self.entries.iter_mut()
    //         .filter(|x| &x.path == &file_name)
    //         .for_each(|x| x.ready = true);
    //     // self.entries.push(CacheEntry {
    //     //     path: file_name,
    //     //     size: length,
    //     //     last_accessed: SystemTime::now(),
    //     //     usage_arc: ret._arc.clone(),
    //     //     ready: true,
    //     // });
    //
    //     Ok(Box::new(ret))
    // }

    // fn has_cache_entry<P: AsRef<Path>>(&self, path: P) -> Option<Arc<()>> {
    //     self.entries
    //         .iter()
    //         .filter(|x| x.path == path.as_ref())
    //         .map(|x| x.usage_arc.clone())
    //         .next()
    // }

    fn remove_stub(&self, path: &PathBuf) {
        let mut guard = self.inner.lock().unwrap();
        guard.remove_entry_stub(path);
        self.ready_count.fetch_add(1, SeqCst);
    }

    #[cold]
    fn get_cache_slow_path(
        &self,
        url: &str,
        usage_arc: Arc<()>,
    ) -> io::Result<Box<dyn 'static + Read>> {
        let file_name = self.path.join(cache_file_for(url));
        let mut bak_file = file_name.as_os_str().to_owned();
        bak_file.push(".bak");

        let response = match self.agent.get(url).call() {
            Ok(x) => x,
            Err(e) => {
                self.remove_stub(&file_name);
                return Err(Error::new(ErrorKind::Other, e));
            }
        };

        let length = match response
            .header("Content-Length")
            .and_then(|x| x.parse::<u64>().ok())
        {
            // Some(length) if self.evict_space_for(length) => length,
            Some(length) => {
                let mut guard = self.inner.lock().unwrap();

                if !guard.set_stub_length(&file_name, length) {
                    self.ready_count.fetch_add(1, SeqCst);
                    return Ok(response.into_reader());
                }

                length
            }
            // Space could not be cleared for the cache entry so bypass the cache and return the
            // http stream instead
            _ => {
                self.remove_stub(&file_name);
                return Ok(response.into_reader());
            }
        };

        let mut reader = response.into_reader();
        let mut file = File::create(&bak_file)?;
        // let usage_arc = Arc::new(());

        // Update cache state
        // self.cache_usage += length;
        // self.entries.push(CacheEntry {
        //     path: file_name.to_owned(),
        //     size: length,
        //     last_accessed: SystemTime::now(),
        //     usage_arc: usage_arc.clone(),
        //     ready: false,
        // });

        // Copy data to bak file
        let result = io::copy(&mut reader, &mut file)
            .and_then(|_| file.flush())
            .and_then(|_| {
                // Close http response and bak file
                drop((file, reader));

                // Rename bak file to intended name
                fs::rename(&bak_file, &file_name)
            });
        // .and_then(|_| {
        //     let mut guard = self.inner.lock().unwrap();
        //     guard.complete_entry_stub(&file_name)
        //
        //     Ok(())
        // })
        // .and_then(|_| File::open(&file_name));

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

            self.remove_stub(&file_name);
            return Err(e);
        }

        let mut guard = self.inner.lock().unwrap();
        guard.complete_entry_stub(&file_name);
        self.ready_count.fetch_add(1, SeqCst);

        // Create final reader
        let ret = FileArcInstance {
            file: File::open(&file_name)?,
            _arc: usage_arc,
        };

        // self.entries.iter_mut()
        //     .filter(|x| &x.path == &file_name)
        //     .for_each(|x| x.ready = true);
        // self.entries.push(CacheEntry {
        //     path: file_name,
        //     size: length,
        //     last_accessed: SystemTime::now(),
        //     usage_arc: ret._arc.clone(),
        //     ready: true,
        // });

        Ok(Box::new(ret))
    }

    fn wait_on_entry_ready(&self, path: &PathBuf) -> bool {
        let mut ready_count = self.ready_count.load(SeqCst);

        loop {
            let guard = self.inner.lock().unwrap();
            match guard.is_entry_ready(path) {
                Some(true) => return true,
                None => return false,
                _ => {} // return true;
            }

            drop(guard);

            let mut new_count = ready_count;
            while new_count == ready_count {
                spin_loop();
                new_count = self.ready_count.load(SeqCst);
            }
            ready_count = new_count;
        }
    }

    pub fn get(&self, url: &str) -> io::Result<Box<dyn 'static + Read>> {
        let file_name = self.path.join(cache_file_for(url));

        loop {
            let present = {
                let mut guard = self.inner.lock().unwrap();
                guard
                    .has_cache_entry(&file_name)
                    .ok_or_else(|| guard.add_entry_stub(file_name.to_owned()))
            };

            match present {
                Ok((usage_arc, ready)) => {
                    if !ready && !self.wait_on_entry_ready(&file_name) {
                        continue;
                    }

                    trace!("Got cache hit for {}", url);
                    return Ok(Box::new(FileArcInstance {
                        file: File::open(file_name)?,
                        _arc: usage_arc,
                    }));
                }
                Err(usage_arc) => {
                    trace!("Got cache miss for {}", url);
                    return self.get_cache_slow_path(url, usage_arc);
                }
            }
        }
    }

    // pub fn get(&self, url: &str) -> io::Result<Box<dyn 'static + Read>> {
    //     let file_name = self.path.join(cache_file_for(url));
    //
    //     let present = {
    //         let guard = self.inner.lock().unwrap();
    //         let found_entry = guard.has_cache_entry(&file_name);
    //         found_entry
    //     };
    //
    //     // The "hot" path of this cache.
    //     if let Some((usage_arc, ready)) = present {
    //         trace!("Got cache hit for {}", url);
    //         if !ready {
    //             self.wait_on_entry_ready(&file_name);
    //         }
    //         return Ok(Box::new(FileArcInstance {
    //             file: File::open(file_name)?,
    //             _arc: usage_arc,
    //         }));
    //     }
    //     // if let Some(usage_arc) = self.has_cache_entry(&file_name) {
    //     //     trace!("Got cache hit for {}", url);
    //     //     return Ok(Box::new(FileArcInstance {
    //     //         file: File::open(file_name)?,
    //     //         _arc: usage_arc,
    //     //     }));
    //     // }
    //
    //     trace!("Got cache miss for {}", url);
    //     self.get_cache_slow_path(url)
    // }
}

struct CacheEntry {
    path: PathBuf,
    size: u64,
    last_accessed: SystemTime,
    /// Use an Arc as a an atomic reference count of readers for a given file
    usage_arc: Arc<()>,
    /// If the cache entry is ready to use yet
    ready: bool,
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
