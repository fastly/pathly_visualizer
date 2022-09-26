mod message;

use crate::message::{BGPChunks, MRTCommonHeader};
use byteorder::{BigEndian, ByteOrder};
use bytes::BytesMut;
use flate2::bufread::GzDecoder;
use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::io::ErrorKind::{InvalidInput, UnexpectedEof};
use std::io::{self, Error, Read};
use std::io::{BufRead, BufReader};
use std::time::Instant;

fn main() -> io::Result<()> {
    let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
    let mut file = BufReader::new(File::open(input_file)?);

    let mut file = GzDecoder::new(file);

    let start_time = Instant::now();

    let mut num_msgs = 0;
    let mut total_length = 0;
    let mut max_msg_len = 0;

    let mut msg_counts: HashMap<_, u64> = HashMap::new();

    for msg in BGPChunks::new(file) {
        let msg = msg?;
        num_msgs += 1;
        total_length += msg.len();
        max_msg_len = max_msg_len.max(msg.len());

        if let Ok(header) = MRTCommonHeader::try_from(&*msg) {
            let msg_type = (
                header.msg_type().expect("Valid message type"),
                header.sub_type(),
            );
            *msg_counts.entry(msg_type).or_default() += 1;
        }
    }

    let mut msg_counts_ordered = msg_counts
        .into_iter()
        .map(|(k, v)| (v, k))
        .collect::<Vec<_>>();
    msg_counts_ordered.sort_unstable();

    for (count, id) in msg_counts_ordered {
        println!("\t{:?}: {}", id, count);
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
