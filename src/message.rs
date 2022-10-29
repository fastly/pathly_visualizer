use bgp_models::mrt::EntryType::*;
use bgp_models::mrt::{CommonHeader, EntryType};
use byteorder::{NetworkEndian, ReadBytesExt};
use bytes::BytesMut;
use num_traits::FromPrimitive;
use std::io;
use std::io::ErrorKind::{InvalidData, UnexpectedEof};
use std::io::{Error, Read};

pub struct BGPChunks<R> {
    next_chunk: BytesMut,
    chunk_size: usize,
    reader: R,
    finished: bool,
}

impl<R> BGPChunks<R> {
    pub fn new(reader: R) -> Self {
        Self::with_capacity(32 * 1024, reader)
    }

    pub fn with_capacity(capacity: usize, reader: R) -> Self {
        BGPChunks {
            chunk_size: capacity,
            next_chunk: BytesMut::new(),
            reader,
            finished: false,
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
            return Err(Error::from(UnexpectedEof));
        }

        Ok(fill - previous_length)
    }
}

impl<R: Read> Iterator for BGPChunks<R> {
    type Item = io::Result<BytesMut>;

    fn next(&mut self) -> Option<Self::Item> {
        if self.finished {
            return None;
        }

        let header = match parse_common_header(&mut &self.next_chunk[..]) {
            Ok(header) => header,
            Err(_) => {
                match self.read_at_least(COMMON_HEADER_MIN_LENGTH) {
                    Ok(_) => {}
                    Err(e) if e.kind() == UnexpectedEof => {
                        self.finished = true;
                        return None;
                    }
                    Err(e) => {
                        self.finished = true;
                        return Some(Err(e));
                    }
                }

                parse_common_header(&mut &self.next_chunk[..])
                    .expect("should be infallible due to previous read_at_least")
            }
        };

        let length_required = header_length(&header) + header.length as usize;
        if self.next_chunk.len() < length_required {
            if let Err(e) = self.read_at_least(length_required) {
                self.finished = true;
                return Some(Err(e));
            }
        }

        Some(Ok(self.next_chunk.split_to(length_required)))
    }
}

const COMMON_HEADER_MIN_LENGTH: usize = 12;

#[inline]
pub fn header_length(header: &CommonHeader) -> usize {
    match header.microsecond_timestamp {
        Some(_) => COMMON_HEADER_MIN_LENGTH + 4,
        None => COMMON_HEADER_MIN_LENGTH,
    }
}

pub fn parse_common_header<R: Read>(buffer: &mut R) -> io::Result<CommonHeader> {
    let timestamp = buffer.read_u32::<NetworkEndian>()?;

    let entry_type = buffer.read_u16::<NetworkEndian>()?;
    let entry_type = EntryType::from_u16(entry_type)
        .ok_or_else(|| Error::new(InvalidData, format!("Got entry type: {}", entry_type)))?;
    let entry_subtype = buffer.read_u16::<NetworkEndian>()?;

    let length = buffer.read_u32::<NetworkEndian>()?;

    let microsecond_timestamp = match entry_type {
        BGP4MP_ET | ISIS_ET | OSPFv3_ET => Some(buffer.read_u32::<NetworkEndian>()?),
        _ => None,
    };

    Ok(CommonHeader {
        timestamp,
        microsecond_timestamp,
        entry_type,
        entry_subtype,
        length,
    })
}
