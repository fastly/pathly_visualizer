mod attribute;
mod cache;
mod dump;
mod env;
mod message;

use crate::cache::PersistentCache;
use crate::dump::index_table::PeerIndexTable;
use crate::dump::rib_entry::parse_rib_subtypes;
use crate::env::setup_dotenv;
use crate::message::{parse_common_header, BGPChunks};
use bgp_models::bgp::AttrType;
use bgp_models::mrt::EntryType::TABLE_DUMP_V2;
use bgp_models::mrt::TableDumpV2Type;
use bytes::BytesMut;
use flate2::bufread::GzDecoder;
use log::{error, LevelFilter};
use num_traits::FromPrimitive;
use std::collections::HashMap;
use std::fs::File;
use std::io;
use std::io::ErrorKind::{InvalidData, Unsupported};
use std::io::{BufReader, Error};
use std::process::exit;
use std::time::Instant;

// fn main() {
//     setup_logging();
//     setup_dotenv();
//
//     let mut cache = match PersistentCache::from_env() {
//         Ok(v) => v,
//         Err(e) => {
//             error!("Failed to init http cache: {:?}", e);
//             exit(1);
//         }
//     };
//
//     cache.clear_bak_files();
//     // cache.update_cache("https://data.ris.ripe.net/rrc00/2022.01/").unwrap();
// }

fn setup_logging() {
    pretty_env_logger::formatted_builder()
        .format_timestamp(None)
        .filter_level(LevelFilter::Debug)
        .filter_module("rustls", LevelFilter::Warn)
        .filter_module("ureq", LevelFilter::Warn)
        // .filter_module("reqwest", LevelFilter::Warn)
        // .filter_module("cookie_store", LevelFilter::Warn)
        .init();
}

fn main() -> io::Result<()> {
    let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
    let file = BufReader::new(File::open(input_file)?);

    let file = GzDecoder::new(file);

    let start_time = Instant::now();

    let mut num_msgs = 0;
    let mut total_length = 0;
    let mut max_msg_len = 0;
    let mut attr_counts: HashMap<AttrType, u64> = HashMap::new();

    for msg in BGPChunks::new(file) {
        let msg = msg?;
        num_msgs += 1;
        total_length += msg.len();
        max_msg_len = max_msg_len.max(msg.len());

        if let Err(e) = handle_bgp_message(msg, &mut attr_counts) {
            eprintln!("{:?}", e);
        }
    }

    // let mut attr_ordered = attr_counts.into_iter().map(|(k, v)| (v, k)).collect::<Vec<_>>();
    // attr_ordered.sort_unstable();
    for (k, v) in attr_counts {
        println!("\t{:?}: {}", k, v);
    }

    println!("Number of messages: {}", num_msgs);
    println!("Average message length: {}", total_length / num_msgs);
    println!("Largest message length: {}", max_msg_len);
    println!(
        "Byte Count: {}GB",
        (total_length as f64) / 1024.0 / 1024.0 / 1024.0
    );

    println!("Total elapsed time: {:?}", start_time.elapsed());
    Ok(())
}

/// I'm still not sure what data I want to collect from this
fn handle_bgp_message(data: BytesMut, attr_counts: &mut HashMap<AttrType, u64>) -> io::Result<()> {
    let mut buffer = &*data;
    let header = parse_common_header(&mut buffer)?;

    if header.entry_type == TABLE_DUMP_V2 {
        let subtype = TableDumpV2Type::from_u16(header.entry_subtype)
            .ok_or_else(|| Error::from(InvalidData))?;

        match subtype {
            TableDumpV2Type::PeerIndexTable => {
                let index_table = PeerIndexTable::try_from(buffer)?;
                for peer in index_table.peer_table {
                    println!("{:?}", peer);
                }

                println!("View name: {:?}", index_table.view_name);
                println!("Succeeded in building index table!");
            }
            TableDumpV2Type::RibIpv4Unicast | TableDumpV2Type::RibIpv6Unicast => {
                let rib = parse_rib_subtypes(&mut buffer)?;
                // println!("Unicast: {:?}", rib.entries.len());

                for entry in &rib.entries {
                    for attribute in &entry.attribute {
                        *attr_counts.entry(attribute.attr_type).or_default() += 1;
                    }
                }

                assert!(
                    buffer.is_empty(),
                    "Parse did not consume all bytes. {} remaining",
                    buffer.len()
                );
            }
            x => {
                return Err(Error::new(
                    Unsupported,
                    format!("Unexpected BGP Table dump subtype: {:?}", x),
                ))
            }
        }
    }

    Ok(())
}
