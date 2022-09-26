extern crate core;

mod attribute;
mod dump;
mod message;

use crate::dump::index_table::PeerIndexTable;
use crate::message::{parse_common_header, BGPChunks};
use bgp_models::mrt::EntryType::TABLE_DUMP_V2;
use bgp_models::mrt::TableDumpV2Type;
use bytes::BytesMut;
use flate2::bufread::GzDecoder;
use num_traits::FromPrimitive;
use std::fs::File;
use std::io;
use std::io::BufReader;
use std::time::Instant;

fn main() -> io::Result<()> {
    let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
    let file = BufReader::new(File::open(input_file)?);

    let file = GzDecoder::new(file);

    let start_time = Instant::now();

    let mut num_msgs = 0;
    let mut total_length = 0;
    let mut max_msg_len = 0;

    for msg in BGPChunks::new(file) {
        let msg = msg?;
        num_msgs += 1;
        total_length += msg.len();
        max_msg_len = max_msg_len.max(msg.len());

        if let Err(e) = handle_bgp_message(msg) {
            eprintln!("{:?}", e);
        }
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
fn handle_bgp_message(data: BytesMut) -> io::Result<()> {
    let mut buffer = &*data;
    let header = parse_common_header(&mut buffer)?;

    if header.entry_type == TABLE_DUMP_V2
        && TableDumpV2Type::from_u16(header.entry_subtype) == Some(TableDumpV2Type::PeerIndexTable)
    {
        let index_table = PeerIndexTable::try_from(buffer)?;
        for peer in index_table.peer_table {
            println!("{:?}", peer);
        }

        println!("View name: {:?}", index_table.view_name);
        println!("Succeeded in building index table!");
    }

    Ok(())
}
