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

        let header = match parse_common_header(&mut &*self.next_chunk) {
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

                parse_common_header(&mut &*self.next_chunk)
                    .expect("should be infallible due to previous read_at_least")
            }
        };

        // let header_length = match header.microsecond_timestamp {
        //     Some(_) => COMMON_HEADER_MIN_LENGTH + 4,
        //     None => COMMON_HEADER_MIN_LENGTH,
        // };

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

// pub struct MRTCommonHeader<'a> {
//     buffer: &'a [u8; 12],
// }

// impl<'a> MRTCommonHeader<'a> {
//     /// The header occupies 12 bytes at the start of each message
//     pub const LENGTH: usize = 12;
//
//     pub fn timestamp(&self) -> u32 {
//         BigEndian::read_u32(&self.buffer[0..4])
//     }
//
//     pub fn msg_type(&self) -> io::Result<MrtType> {
//         match BigEndian::read_u16(&self.buffer[4..6]) {
//             11 => Ok(MrtType::OSPFv2),
//             12 => Ok(MrtType::TABLE_DUMP),
//             13 => Ok(MrtType::TABLE_DUMP_V2),
//             16 => Ok(MrtType::BGP4MP),
//             17 => Ok(MrtType::BGP4MP_ET),
//             32 => Ok(MrtType::ISIS),
//             33 => Ok(MrtType::ISIS_ET),
//             48 => Ok(MrtType::OSPFv3),
//             49 => Ok(MrtType::OSPFv3_ET),
//             _ => Err(Error::from(InvalidData)),
//         }
//     }
//     pub fn sub_type(&self) -> u16 {
//         BigEndian::read_u16(&self.buffer[6..8])
//     }
//     pub fn length(&self) -> u32 {
//         BigEndian::read_u32(&self.buffer[8..12])
//     }
// }

// impl<'a> TryFrom<&'a [u8]> for MRTCommonHeader<'a> {
//     type Error = io::Error;
//
//     #[inline(always)]
//     fn try_from(value: &'a [u8]) -> Result<Self, Self::Error> {
//         // Enforce that the length is checked at most once.
//         match value.get(..Self::LENGTH).map(|x| x.try_into()) {
//             Some(Ok(buffer)) => Ok(MRTCommonHeader { buffer }),
//             _ => Err(Error::from(InvalidInput)),
//         }
//     }
// }
//
// #[derive(Copy, Clone, Hash, Ord, PartialOrd, Eq, PartialEq, Debug)]
// #[repr(u16)]
// pub enum MrtType {
//     OSPFv2 = 11,
//     TABLE_DUMP = 12,
//     TABLE_DUMP_V2 = 13,
//     BGP4MP = 16,
//     BGP4MP_ET = 17,
//     ISIS = 32,
//     ISIS_ET = 33,
//     OSPFv3 = 48,
//     OSPFv3_ET = 49,
// }

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

    let entry_type = EntryType::from_u16(buffer.read_u16::<NetworkEndian>()?)
        .ok_or_else(|| Error::from(InvalidData))?;
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
