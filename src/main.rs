#![feature(bench_black_box)]

use std::fs::File;
use std::hint::black_box;
use std::io::{self, Error, Read};
use std::io::{BufRead, BufReader};
use std::io::ErrorKind::{InvalidInput, UnexpectedEof};
use std::time::Instant;
use byteorder::{BigEndian, ByteOrder};
use bytes::BytesMut;
use flate2::bufread::GzDecoder;
// use gzip::GzipReader;

fn main() -> io::Result<()> {

    for i in 0..10 {
        test_for(1024 << i)?;
    }

    let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
    let mut file = BufReader::new(File::open(input_file)?);

    let mut file = GzDecoder::new(file);


    // let mut buffer = vec![0; 8192];

    let start_time = Instant::now();


    // let segment_length = 128 * 1024 * 1024;
    let mut num_msgs = 0;
    let mut total_length = 0;
    let mut max_msg_len = 0;

    for msg in BGPChunks::new(file) {
        let msg = msg?;
        num_msgs += 1;
        total_length += msg.len();
        max_msg_len = max_msg_len.max(msg.len());
    }

    // let mut has_more = true;
    // while has_more {
    //     let mut buffer = BytesMut::zeroed(segment_length);
    //
    //     let mut fill = 0;
    //     while fill < segment_length {
    //         let bytes_read = file.read(&mut buffer[fill..])?;
    //         if bytes_read == 0 {
    //             has_more = false;
    //             break
    //         }
    //
    //         fill += bytes_read;
    //     }
    //
    // }


    // let mut byte_count = 0;

    // loop {
    //     match file.read(&mut buffer[..]) {
    //         Ok(0) => break,
    //         Ok(x) => {
    //             byte_count += x;
    //             black_box(&buffer[..]);
    //         }
    //         Err(e) => return Err(e),
    //     }
    // }

    // loop {
    //     let buffer = file.fill_buf()?;
    //     match buffer.len() {
    //         0 => break,
    //         x => {
    //             byte_count += x;
    //             file.consume(x);
    //         }
    //     }
    // }

    println!("Number of messages: {}", num_msgs);
    println!("Average message length: {}", total_length  / num_msgs);
    println!("Largest message length: {}", max_msg_len);
    println!("Byte Count: {}GB", (total_length as f64) / 1024.0 / 1024.0 / 1024.0);

    println!("Total elapsed time: {:?}", start_time.elapsed());
    Ok(())
}

fn test_for(buff_len: usize) -> io::Result<()> {
    let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
    let mut file = BufReader::new(File::open(input_file)?);
    let mut file = GzDecoder::new(file);

    let start_time = Instant::now();

    let mut num_msgs = 0;
    let mut total_length = 0;
    let mut max_msg_len = 0;

    for msg in BGPChunks::new(file) {
        let msg = msg?;
        num_msgs += 1;
        total_length += msg.len();
        max_msg_len = max_msg_len.max(msg.len());
        black_box(msg);
    }

    println!("Took {:?} for buffer of length {}KB", start_time.elapsed(), buff_len / 1024);
    Ok(())
}



pub struct BGPChunks<R> {
    next_chunk: BytesMut,
    // chunk_usage: usize,
    chunk_size: usize,
    reader: R,
    finished: bool,
}

impl<R> BGPChunks<R> {
    fn new(reader: R) -> Self {
        Self::with_capacity(64 * 1024, reader)
    }

    fn with_capacity(capacity: usize, reader: R) -> Self {
        BGPChunks {
            chunk_size: capacity,
            next_chunk: BytesMut::new(),
            reader,
            finished: false,
            // chunk_usage: 0
        }
    }
}

impl<R: Read> BGPChunks<R> {
    fn read_at_least(&mut self, length: usize) -> io::Result<usize> {
        let previous_length = self.next_chunk.len();
        let target_length = length.max(self.chunk_size);
        self.next_chunk.resize(target_length, 0);

        let mut fill = previous_length;
        while fill < target_length {
            match self.reader.read(&mut self.next_chunk[fill..])? {
                0 => break,
                x => fill += x,
            }
        }

        self.next_chunk.truncate(fill);

        if fill < length {
            // Unable to fill buffer to required length
            return Err(Error::from(UnexpectedEof))
        }

        Ok(fill - previous_length)
    }
}

impl<R: Read> Iterator for BGPChunks<R> {
    type Item = io::Result<BytesMut>;

    fn next(&mut self) -> Option<Self::Item> {
        if self.finished {
            return None
        }

        let header = match MRTCommonHeader::try_from(&*self.next_chunk) {
            Ok(header) => header,
            Err(_) => {
                match self.read_at_least(MRTCommonHeader::LENGTH) {
                    Ok(_) => {}
                    Err(e) if e.kind() == UnexpectedEof => {
                        self.finished = true;
                        return None;
                    }
                    Err(e) => {
                        self.finished = true;
                        return Some(Err(e))
                    }
                }

                MRTCommonHeader::try_from(&*self.next_chunk)
                    .expect("should be infallible due to previous read_at_least")
            }
        };

        let length_required = MRTCommonHeader::LENGTH + header.length as usize;
        if self.next_chunk.len() < length_required {
            if let Err(e) = self.read_at_least(length_required) {
                self.finished = true;
                return Some(Err(e))
            }
        }

        Some(Ok(self.next_chunk.split_to(length_required)))
    }
}


pub struct MRTCommonHeader {
    timestamp: u32,
    msg_type: u16,
    sub_type: u16,
    length: u32,
}

impl MRTCommonHeader {
    /// The header occupies 12 bytes at the start of each message
    const LENGTH: usize = 12;


}

impl TryFrom<&[u8]> for MRTCommonHeader {
    type Error = io::Error;

    #[inline(always)]
    fn try_from(value: &[u8]) -> Result<Self, Self::Error> {
        // Enforce that the length is checked at most once.
        let buffer: &[u8; 12] = if value.len() >= 12 {
            value[..12].try_into().expect("Infallible due to previous check")
        } else {
            return Err(Error::from(InvalidInput))
        };

        Ok(MRTCommonHeader {
            timestamp: BigEndian::read_u32(&buffer[0..4]),
            msg_type: BigEndian::read_u16(&buffer[4..6]),
            sub_type: BigEndian::read_u16(&buffer[6..8]),
            length: BigEndian::read_u32(&buffer[8..12]),
        })
    }
}







