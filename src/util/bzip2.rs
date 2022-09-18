use bzip2::bufread::BzDecoder;
use std::ffi::OsString;
use std::fs::File;
use std::io::{BufReader, Error, ErrorKind, IoSliceMut, Read};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::{env, io};

fn has_executable<P: AsRef<Path>>(name: P) -> Option<PathBuf> {
    let path = name.as_ref();

    // If path is absolute, no need to check it on the path
    if path.is_absolute() {
        return path.is_file().then(|| path.to_path_buf());
    }

    platform_specific_resolution(path)
}

#[cfg(target_os = "unix")]
fn platform_specific_resolution(name: &Path) -> Option<PathBuf> {
    let path = env::var_os("PATH")?;

    env::split_paths(&path)
        .filter_map(|dir| {
            let full_path = dir.join(name);
            full_path.is_file().then(|| full_path)
        })
        .next()
}

// idk if it is even possible to install lbzip2 or pbzip2 on Windows, but we may as well try
// searching for one.
#[cfg(target_os = "windows")]
fn platform_specific_resolution(name: &Path) -> Option<PathBuf> {
    let path = env::var_os("PATH")?;
    let path_ext = env::var_os("PATHEXT").map_or(Vec::new(), |x| env::split_paths(&x).collect());

    // Windows searches the current directory first
    if let Some(location) = win_search_location(name, &path_ext) {
        return Some(location);
    }

    for directory in env::split_paths(&path) {
        if let Some(location) = win_search_location(&directory.join(name), &path_ext) {
            return Some(location);
        }
    }

    None
}

#[cfg(target_os = "windows")]
fn win_search_location(path: &Path, path_ext: &[PathBuf]) -> Option<PathBuf> {
    // Case where file already has an extension
    if path.is_file() {
        return Some(path.to_path_buf());
    }

    // Try each extension
    let mut path_buffer = OsString::with_capacity(path.as_os_str().len() + 8);
    for extension in path_ext {
        path_buffer.clear();
        path_buffer.push(path.as_os_str());
        path_buffer.push(extension.as_os_str());

        let path_ref: &Path = path_buffer.as_ref();
        if path_ref.is_file() {
            return Some(PathBuf::from(path_buffer));
        }
    }

    None
}

#[cfg(all(unix, feature = "wide_pipe"))]
fn max_pipe_size() -> Option<usize> {
    let mut file = File::open("/proc/sys/fs/pipe-max-size").ok()?;
    let mut buffer = String::new();
    file.read_to_string(&mut buffer).ok()?;
    buffer.parse().ok()
}

/// Creates a pair of files corresponding to a unix pipe. Data written to the second file can then
/// be read from the first. The pipe buffer is also expanded to a desired size of 1MB or greater if
/// possible to reduce the chance of IO blocking. This is important since the default bzip2 block
/// size is 900kB and if bzip2 is blocked by IO it loses CPU time that could be spent decoding the
/// next block. Unfortunately, bzip2 is very slow compared to competitors when decoding so this can
/// have an impact on performance in some cases.
#[cfg(all(unix, feature = "wide_pipe"))]
fn wide_pipe() -> io::Result<(File, File)> {
    use std::os::unix::io::FromRawFd;

    unsafe {
        let mut pipes: [c_int; 2] = [0; 2];
        if libc::pipe(pipes.as_mut_ptr()) != 0 {
            return Err(Error::last_os_error());
        }

        let pipe_size = libc::fcntl(pipes[1], libc::F_GETPIPE_SZ);

        // Read max pipe size or assume it is 1MB like stated on linux stack exchange
        let mut desired_size = max_pipe_size().unwrap_or(1024 * 1024).min(1024 * 1024);

        // Increase pipe buffer size
        if desired_size > pipe_size as usize {
            libc::fcntl(pipes[1], libc::F_SETPIPE_SZ, desired_size as c_int);
        }

        Some((File::from_raw_fd(pipes[0]), File::from_raw_fd(pipes[1])))
    }
}

/// The bzip2 decompression algorithm is quite slow to decompress files. While we can use the
/// static linked bzip2 implementation included in this executable, we would ideally like to have
/// a parallel bzip2 implementation on this system. Since decompression is usually the slowest
/// factor in processing data, we can improve our performance roughly linearly with the number of
/// cores available on this system.
fn find_bzip_installation() -> Option<PathBuf> {
    ["lbzip2", "pbzip2", "bzip2"]
        .into_iter()
        .filter_map(has_executable)
        .next()
}

pub struct BzipDecoderStream {
    stream: Box<dyn 'static + Read>,
    _child: Option<Child>,
}

impl BzipDecoderStream {
    pub fn new<P: AsRef<Path>>(path: P) -> io::Result<Self> {
        match find_bzip_installation() {
            Some(location) => Self::piped_stream(path, location),
            None => Self::direct_stream(path),
        }
    }

    pub fn direct_stream<P: AsRef<Path>>(path: P) -> io::Result<Self> {
        let decoder = BzDecoder::new(BufReader::new(File::open(path)?));
        Ok(BzipDecoderStream {
            stream: Box::new(decoder),
            _child: None,
        })
    }

    pub fn piped_stream<P: AsRef<Path>>(path: P, executable: PathBuf) -> io::Result<Self> {
        let mut cmd = Command::new(executable);
        cmd.args(["-d", "-c", "-k"])
            .arg(path.as_ref())
            .stdin(Stdio::null())
            .stderr(Stdio::null());

        #[cfg(all(unix, feature = "wide_pipe"))]
        if let Ok((read, write)) = wide_pipe() {
            let child = cmd.stdout(write).spawn()?;
            return Ok(BzipDecoderStream {
                stream: Some(read),
                _child: Some(child),
            });
        }

        let mut child = cmd.stdout(Stdio::piped()).spawn()?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| Error::new(ErrorKind::BrokenPipe, "Stdout pipe not found"))?;

        Ok(BzipDecoderStream {
            stream: Box::new(stdout),
            _child: Some(child),
        })
    }
}

impl Read for BzipDecoderStream {
    fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
        self.stream.read(buf)
    }

    fn read_vectored(&mut self, bufs: &mut [IoSliceMut<'_>]) -> io::Result<usize> {
        self.stream.read_vectored(bufs)
    }
}
