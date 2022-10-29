use bgp_models::bgp::{AttrType, Attribute, AttributeValue, Nlri, Origin};
use bgp_models::network::{Afi, NetworkPrefix, NextHopAddress, Safi};
use bitflags::bitflags;
use byteorder::{NetworkEndian, ReadBytesExt};
use ipnetwork::{IpNetwork, Ipv4Network, Ipv6Network};
use num_traits::FromPrimitive;
use std::io;
use std::io::ErrorKind::InvalidData;
use std::io::{Error, Read, Take};
use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};
use std::slice::from_ref;

#[derive(Debug)]
pub struct TestRibSubtypes {
    pub sequence_number: u32,
    // pub prefix_len: u8,
    // pub prefix: [u8; 32],
    pub prefix: NetworkPrefix,
    pub entries: Vec<RIBEntry>,
}

pub fn parse_rib_subtypes(buffer: &mut &[u8], afi: Afi, safi: Safi) -> io::Result<TestRibSubtypes> {
    let sequence_number = buffer.read_u32::<NetworkEndian>()?;
    let prefix = parse_prefix(buffer, afi)?;

    // let prefix_len = buffer.read_u8()?;
    // // if prefix_len > 32 {
    // //     println!("Got large prefix length: {}", prefix_len);
    // // }
    //
    // let prefix_byte_len = (prefix_len as usize + 7) / 8;
    // let mut prefix = [0; 32];
    // buffer.read_exact(&mut prefix[..prefix_byte_len])?;

    Ok(TestRibSubtypes {
        sequence_number: buffer.read_u32::<NetworkEndian>()?,
        // prefix_len,
        // prefix,
        entries: {
            let num_entries = buffer.read_u16::<NetworkEndian>()?;
            let mut entries = Vec::with_capacity(num_entries as usize);
            for _ in 0..num_entries {
                entries.push(parse_rib_entry(
                    buffer,
                    Some((afi, safi)),
                    from_ref(&prefix),
                )?);
            }
            entries
        },
        prefix,
    })
}

#[derive(Debug)]
pub struct RIBEntry {
    pub peer_index: u16,
    pub originated_time: u32,
    pub attribute: Vec<Attribute>,
}

pub fn parse_rib_entry(
    buffer: &mut &[u8],
    rib_type: Option<(Afi, Safi)>,
    prefixes: &[NetworkPrefix],
) -> io::Result<RIBEntry> {
    Ok(RIBEntry {
        peer_index: buffer.read_u16::<NetworkEndian>()?,
        originated_time: buffer.read_u32::<NetworkEndian>()?,
        attribute: {
            let length = buffer.read_u16::<NetworkEndian>()? as usize;
            let mut attr_buffer = &buffer[..length];
            *buffer = &buffer[length..];
            parse_attributes(&mut attr_buffer, rib_type, prefixes)?

            // let mut data = vec![0; length as usize];
            // buffer.read_exact(&mut data)?;
            // // data
            //
            // let mut buffer = &data[..];
            //
            //
            // if buffer[1] == 1 {
            //     println!("Origin bytes: {}", buffer.len());
            //     panic!("{:?}\n", buffer);
            // }
            // // MP_UNREACHABLE_NLRI: 3838
            // // ORIGIN: 53997579
            // // MP_REACHABLE_NLRI: 2928024
            //
            // let attr = Attribute {
            //     flags: AttributeFlags::from_bits(buffer.read_u8()?)
            //         .ok_or_else(|| Error::from(InvalidData))?,
            //     attr_type: AttrType::from_u8(buffer.read_u8()?)
            //         .ok_or_else(|| Error::from(InvalidData))?,
            // };
            //
            //
            // // while attr.attr_type == AttrType::ORIGIN && !buffer.is_empty() {
            // //     let length = buffer.read_u8()?;
            // //     let mut prefix = [0; 32];
            // //     buffer.read_exact(&mut prefix[..(length as usize + 7) / 8])?;
            // // }
            //
            // attr
            // todo!()
        },
    })
}

bitflags! {
    pub struct AttributeFlags: u8 {
        const OPTIONAL = 0b1000_0000;
        const TRANSITIVE = 0b0100_0000;
        const PARTIAL = 0b0010_0000;
        const EXTEND_LENGTH = 0b0001_0000;
    }
}

pub fn parse_attributes(
    buffer: &mut &[u8],
    rib_type: Option<(Afi, Safi)>,
    prefixes: &[NetworkPrefix],
) -> io::Result<Vec<Attribute>> {
    let mut attributes = Vec::new();
    while buffer.len() > 0 {
        let flags =
            AttributeFlags::from_bits(buffer.read_u8()?).ok_or_else(|| Error::from(InvalidData))?;
        let attr_type =
            AttrType::from_u8(buffer.read_u8()?).ok_or_else(|| Error::from(InvalidData))?;

        // You would think they wouldn't force the length of fixed sized variants be recorded if
        // they want to save space.
        let _attr_length = if flags.contains(AttributeFlags::EXTEND_LENGTH) {
            buffer.read_u8()? as usize
        } else {
            buffer.read_u16::<NetworkEndian>()? as usize
        };

        let next_attribute = match attr_type {
            AttrType::ORIGIN => {
                let origin =
                    Origin::from_u8(buffer.read_u8()?).ok_or_else(|| Error::from(InvalidData))?;

                AttributeValue::Origin(origin)
            }
            AttrType::MP_REACHABLE_NLRI => {
                AttributeValue::MpReachNlri(parse_reachable_nlri(buffer, rib_type, prefixes)?)
            }
            x => unimplemented!("Parse attribute: {:?}", x),
        };

        attributes.push(Attribute {
            attr_type,
            value: next_attribute,
            flag: flags.bits,
        });
    }

    Ok(attributes)
}

pub fn parse_reachable_nlri(
    buffer: &mut &[u8],
    rib_type: Option<(Afi, Safi)>,
    prefixes: &[NetworkPrefix],
) -> io::Result<Nlri> {
    let read_full_info = buffer.get(0) == Some(&0);

    let (afi, safi) = match rib_type {
        Some(x) if !read_full_info => x,
        _ => {
            let afi = Afi::from_u16(buffer.read_u16::<NetworkEndian>()?)
                .ok_or_else(|| Error::from(InvalidData))?;

            let safi = Safi::from_u8(buffer.read_u8()?).ok_or_else(|| Error::from(InvalidData))?;

            (afi, safi)
        }
    };

    let next_hop = match buffer.read_u8()? {
        0 => None,
        4 => Some(NextHopAddress::Ipv4(Ipv4Addr::from(
            buffer.read_u32::<NetworkEndian>()?,
        ))),
        16 => Some(NextHopAddress::Ipv6(Ipv6Addr::from(
            buffer.read_u128::<NetworkEndian>()?,
        ))),
        32 => Some(NextHopAddress::Ipv6LinkLocal(
            Ipv6Addr::from(buffer.read_u128::<NetworkEndian>()?),
            Ipv6Addr::from(buffer.read_u128::<NetworkEndian>()?),
        )),
        _ => return Err(Error::from(InvalidData)),
    };

    // let nlri_prefixes = if read_full_info {
    //     // Read 2 placeholder bytes
    //     buffer.read_exact(&mut [0u8; 2])?;
    //
    //
    // } else {
    //     prefixes.to_vec()
    // };

    todo!()
}

fn parse_prefix(buffer: &mut &[u8], afi: Afi) -> io::Result<NetworkPrefix> {
    let len = buffer.read_u8()?;

    let prefix = match afi {
        Afi::Ipv4 if len <= 32 => {
            let mut ip_buffer = [0u8; 4];
            buffer.read_exact(&mut ip_buffer[..(len as usize + 7) / 8])?;

            IpNetwork::V4(
                Ipv4Network::new(Ipv4Addr::from(ip_buffer), len)
                    .map_err(|_| Error::from(InvalidData))?,
            )
        }
        Afi::Ipv6 if len <= 128 => {
            let mut ip_buffer = [0u8; 16];
            buffer.read_exact(&mut ip_buffer[..(len as usize + 7) / 8])?;

            IpNetwork::V6(
                Ipv6Network::new(Ipv6Addr::from(ip_buffer), len)
                    .map_err(|_| Error::from(InvalidData))?,
            )
        }
        _ => return Err(Error::from(InvalidData)),
    };

    Ok(NetworkPrefix { prefix, path_id: 0 })
}
