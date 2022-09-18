use std::collections::HashMap;
use std::fmt::{Display, Formatter};
use std::fs::File;
use std::hint::black_box;
// use std::hint::black_box;
use std::io::{BufReader, Read, BufRead, BufWriter, Write};
use bzip2::bufread::BzDecoder;
use format_serde_error::SerdeError;
use tokio::time::Instant;
use serde_json::{Number, Value};
use crate::bench::ProgressCounter;
use crate::ripe_atlas::traceroute::Traceroute;
use crate::simple_bench;


simple_bench!{READ_LINE}
simple_bench!{PARSE_JSON}



// 0 were missing firmware versions
// Longest measurement required 30341 bytes
// Took a total of 736.7206657s
// READ_LINE: 342.4180802s over 8891677 usages (38.509µs average)
// PARSE_JSON: 387.2345553s over 8891676 usages (43.549µs average)
pub fn deserialize_test() -> anyhow::Result<()> {
    let path = "C:\\Users\\Jasper\\Downloads\\traceroute-2022-09-01T2300.bz2";
    let file = BufReader::new(File::open(path)?);

    // let mut decoder = BzDecoder::new(file);
    let mut decoder = BufReader::new(BzDecoder::new(file));

    let mut errors = BufWriter::new(File::create("parse_errors.txt")?);

    let progress_interval = 10000;
    let mut progress = 0;

    let mut error_count = 0;
    let mut version_count: HashMap<u32, u64> = HashMap::new();
    let mut probe_count: HashMap<i64, u64> = HashMap::new();
    let mut missing_fw = 0;
    let mut longest_line = 0;

    // let mut buffer = vec![0; 64 * 1024];
    let mut total_bytes = 0;
    let start_time = Instant::now();
    let mut buffer = String::new();
    // loop {
    //     let bytes_read = decoder.read(&mut buffer)?;
    //
    //     if bytes_read == 0 {
    //         break
    //     }
    //
    //     total_bytes += bytes_read;
    //     black_box(&buffer[..]);
    // }

    while simple_bench!(READ_LINE: decoder.read_line(&mut buffer)? != 0) {
        longest_line = longest_line.max(buffer.len());
        let json = simple_bench!(PARSE_JSON: serde_json::from_str::<Traceroute>(&buffer));
        // let json = simple_bench!(PARSE_JSON: serde_json::from_str::<TracerouteMeasurement>(&buffer));

        match json.map_err(|err| SerdeError::new(buffer.to_owned(), err)) {
            Ok(v) => {
                let count = version_count.entry(v.fw).or_default();
                *count += 1;
                let count = probe_count.entry(v.prb_id).or_default();
                *count += 1;
            },
            Err(e) => {
                error_count += 1;
                writeln!(&mut errors, "{}\nItem #{}:\n{}", prettyify_json(&buffer)?, progress, &e)?;
                // println!("{}", prettyify_json(&buffer)?);
                println!("Item #{}:\n{}", progress, e);
                // if error_count >= 1 {
                //     panic!();
                // }
            }
        }

        // Display how many items it has processed to give a sense of speed
        progress += 1;
        if progress % progress_interval == 0 {
            println!("Working... {} ({} errors)", progress, error_count);
        }

        buffer.clear();
    }

    println!("Found a total of {} entries with {} errors", progress, error_count);

    // println!("Firmware versions:");
    // let mut versions = version_count.into_iter().collect::<Vec<_>>();
    // versions.sort();
    // for (k, v) in versions {
    //     println!("\t{}: {}", k, v);
    // }
    //
    // println!("{} were missing firmware versions", missing_fw);

    println!("Probe counts:");
    let mut versions = probe_count.into_iter().map(|(k, v)| (v, k)).collect::<Vec<_>>();
    versions.sort();
    for (k, v) in versions {
        println!("\t{}: {}", v, k);
    }

    println!("Longest measurement required {} bytes", longest_line);

    println!("Took a total of {:?}", start_time.elapsed());
    println!("READ_LINE: {}", &READ_LINE);
    println!("PARSE_JSON: {}", &PARSE_JSON);

    // println!("Read a total of {} bytes", total_bytes);

    Ok(())
}


pub fn try_convert<R: BufRead>(reader: &mut R) -> anyhow::Result<u64> {
    let mut bytes_in: u64 = 0;
    let mut mock_out = ByteCounter::default();
    let mut progress = ProgressCounter::new(10000);

    let mut buffer = String::new();
    loop  {
        match reader.read_line(&mut buffer)? {
            0 => break,
            x => bytes_in += x as u64,
        };

        let data = serde_json::from_str::<Traceroute>(&buffer)?;
        serde_cbor::to_writer(&mut mock_out, &data)?;
        buffer.clear();

        // Add imaginary linebreak (Optimizes to incrementing mock_out count)
        assert_eq!(mock_out.write(&[b'\n']).expect("Infallible"), 1);

        progress.periodic(|count| {
            let ratio = mock_out.count as f64 / bytes_in as f64;
            println!("Completed {}: {} input / {} output ({:.5} Compression)", count, HumanReadableBytes(bytes_in), HumanReadableBytes(mock_out.count), ratio);
        });
    }

    Ok(mock_out.count)
}

pub fn count_dns_lookups<R: BufRead>(reader: &mut R) -> anyhow::Result<u64> {
    let mut progress = ProgressCounter::new(10000);
    let mut dns_lookups = 0;

    let mut buffer = String::new();
    while reader.read_line(&mut buffer)? > 0 {
        let data = serde_json::from_str::<Traceroute>(&buffer)?;
        buffer.clear();

        if let Some(x) = &data.dst_addr {
            if x != &data.dst_name {
                dns_lookups += 1;
            }
        }

        progress.periodic(|count| {
            let ratio = dns_lookups as f64 / count as f64;
            println!("Completed {}: {} lookups ({:.5}%)", count, dns_lookups, ratio * 100.0);
        });
    }

    Ok(dns_lookups)
}



#[derive(Default, Debug)]
struct ByteCounter {
    count: u64,
}

impl Write for ByteCounter {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        self.count += buf.len() as u64;
        Ok(buf.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

fn prettyify_json(x: &str) -> anyhow::Result<String> {
    let value: Value = serde_json::from_str(x)?;
    serde_json::to_string_pretty(&value).map_err(Into::into)
}



pub struct HumanReadableBytes(pub u64);

impl Display for HumanReadableBytes {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        const BYTE_SUFFIX: &'static [&'static str] = &["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
        let mut bytes_float = self.0 as f64;
        let mut suffix = 0;

        loop {
            if bytes_float < 1024.0 {
                return write!(f, "{:.3}{}", bytes_float, BYTE_SUFFIX[suffix])
            }

            bytes_float /= 1024.0;
            suffix += 1;
        }
    }
}
