#![cfg_attr(all(feature = "bench", feature = "nightly"), feature(bench_black_box))]
// TODO: Remove this once I begin making an actual usable program instead of random tests
#![allow(dead_code)]

use crate::asn::ASNTable;
use crate::rate_limit::UsageLimiter;
use crate::ripe_atlas::MeasurementResponse;
use crate::util::HumanBytes;
use bzip2::bufread::BzDecoder;
use std::fs::File;
use std::io;
use std::io::{BufRead, BufReader, Read};
use std::path::Path;
use std::time::{Duration, Instant};
use tokio::runtime::Builder;

mod asn;
mod ip;
mod rate_limit;
mod ripe_atlas;
mod util;

#[cfg(feature = "bench")]
mod bz2;

fn main() {
    let builder = Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("Failed to build Tokio async runtime");
    // .enter();
    //
    // builder.block_on(start());

    let start = async {
        println!("Fetching ASN table...");
        let _ = ASNTable::fetch_and_load().await.unwrap();
    };

    builder.block_on(start);

    // perform_file_read_test();

    // let path = "C:\\Users\\Jasper\\Downloads\\traceroute-2022-09-01T2300.bz2";
    // {
    //     let path = "C:\\Users\\Jasper\\Downloads\\dns-2022-08-08T2300.bz2";
    //     let file = BufReader::new(File::open(path).unwrap());
    //     let mut decoder = BufReader::new(BzDecoder::new(file));
    //
    //     // let output = try_convert(&mut decoder)?;
    //     // let output = count_dns_lookups(&mut decoder)?;
    //     // println!("Total output bytes: {}", HumanReadableBytes(output));
    //     debug_read::<GeneralMeasurement<DNSLookup>, _>(&mut decoder).unwrap();
    // }

    #[cfg(unix)]
    unsafe {
        let path = "/mnt/c/Users/Jasper/Downloads/dns-2022-08-08T2300.bz2";
        // let file = BufReader::new(File::open(path).unwrap());
        // let mut decoder = BufReader::new(BzDecoder::new(file));

        let mut pipes: [c_int; 2] = [0; 2];
        assert_eq!(libc::pipe(pipes.as_mut_ptr()), 0);

        let read_size = libc::fcntl(pipes[0], libc::F_GETPIPE_SZ);
        let write_size = libc::fcntl(pipes[1], libc::F_GETPIPE_SZ);
        println!("Pipe sizes: {} and {}", read_size, write_size);
        // println!("Max pipe size: {:?}", max_pipe_size());

        let mut max_pipe_size = match max_pipe_size() {
            Some(v) => v,
            None => 1024 * 1024, // Probably 1MB like stated on linux stack exchange
        };

        // 1MB should be enough for our use case
        let desired_size = max_pipe_size.min(1024 * 1024);
        println!("Using desired pipe size of {}", desired_size);

        // Increase pipe buffer size
        if (write_size as usize) < desired_size {
            libc::fcntl(pipes[1], libc::F_SETPIPE_SZ, desired_size as c_int);

            let read_size = libc::fcntl(pipes[0], libc::F_GETPIPE_SZ);
            let write_size = libc::fcntl(pipes[1], libc::F_GETPIPE_SZ);
            println!("Pipe sizes: {} and {}", read_size, write_size);
        }

        use std::os::unix::io::FromRawFd;

        let command = Command::new("lbzcat")
            .arg("-n")
            .arg(format!("{}", (num_cpus::get() - 1).max(1)))
            .arg(path)
            .stdin(Stdio::null())
            .stdout(Stdio::from_raw_fd(pipes[1]))
            .spawn()
            .expect("Spawned lbzcat successfully");

        let mut decode_stream = BufReader::new(File::from_raw_fd(pipes[0]));
        // let mut decode_stream = BufReader::with_capacity(128 * 1024, File::from_raw_fd(pipes[0]));
        // let mut decode_stream = BufReader::with_capacity(1024, command.stdout.unwrap());

        let (bytes, duration) = file_read_time(&mut decode_stream).unwrap();
        println!("Read {} bytes in {:?}", HumanReadableBytes(bytes), duration);
    }

    // With bzip2 crate
    // Successfully parsed all values in 141.881831s
    // Line Read Time:  110.1027722s over 8147079 usages (13.513µs average)
    // JSON Parse Time: 29.2561112s over 8147078 usages (3.59µs average)

    // With call to lbzcat
    // Successfully parsed all values in 43.0157711s
    // Line Read Time:  6.2918785s over 8147079 usages (771ns average)
    // JSON Parse Time: 27.8783114s over 8147078 usages (3.421µs average)

    // With lbzcat and rayon
    // Successfully parsed all values in 32.3510148s
    // JSON Parse Time: 85.4278365s over 8147078 usages (10.485µs average)

    // Ok(())
}

fn max_pipe_size() -> Option<usize> {
    let mut file = File::open("/proc/sys/fs/pipe-max-size").ok()?;
    let mut buffer = String::new();
    file.read_to_string(&mut buffer).ok()?;
    buffer.parse().ok()
}

pub fn perform_file_read_test() {
    let large_file = "C:\\Users\\Jasper\\Downloads\\dns-2022-08-08T2300.bz2";

    // for size in 0..8 {
    let size = 4;
    let buffer_size = 4096 * (1 << size);
    let (bytes, duration) = file_read_test(&large_file, buffer_size, 900000).unwrap();
    println!(
        "Using buffer size of {}: Read {} bytes in {:?}",
        HumanBytes(buffer_size as u64),
        HumanBytes(bytes),
        duration
    );
    // }
}

pub fn file_read_time<B: BufRead>(buffer: &mut B) -> io::Result<(u64, Duration)> {
    let mut total_bytes = 0;
    let start_time = Instant::now();

    loop {
        let read_len = buffer.fill_buf()?.len();
        if read_len == 0 {
            return Ok((total_bytes, start_time.elapsed()));
        }

        buffer.consume(read_len);
        total_bytes += read_len as u64;
    }
}

pub fn file_read_test<P: AsRef<Path>>(
    path: P,
    in_buff_size: usize,
    out_buff_size: usize,
) -> io::Result<(u64, Duration)> {
    let file = BufReader::with_capacity(in_buff_size, File::open(path)?);
    let mut decoded = BufReader::with_capacity(out_buff_size, BzDecoder::new(file));
    let mut total_bytes = 0;
    let start_time = Instant::now();

    loop {
        let read_len = decoded.fill_buf()?.len();
        if read_len == 0 {
            return Ok((total_bytes, start_time.elapsed()));
        }

        decoded.consume(read_len);
        total_bytes += read_len as u64;
    }
}
