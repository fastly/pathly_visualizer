use bitflags::bitflags;
use byteorder::{BigEndian, ReadBytesExt};
use std::io;
use std::io::ErrorKind::InvalidData;
use std::io::{BufRead, Error};
use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

#[derive(Debug)]
pub struct PeerIndexTable {
    pub collector_bgp_id: u32,
    pub view_name: Option<String>,
    pub peer_table: Vec<BgpPeer>,
}

impl TryFrom<&[u8]> for PeerIndexTable {
    type Error = Error;

    fn try_from(mut value: &[u8]) -> Result<Self, Self::Error> {
        let collector_bgp_id = value.read_u32::<BigEndian>()?;

        let view_name = match value.read_u16::<BigEndian>()? {
            0 => None,
            x => {
                let bytes = value[..x as usize].to_vec();
                value.consume(x as usize);
                Some(String::from_utf8(bytes).map_err(|_| Error::from(InvalidData))?)
            }
        };

        let peer_count = value.read_u16::<BigEndian>()?;
        let mut peer_table = Vec::with_capacity(peer_count as usize);

        for _ in 0..peer_count {
            peer_table.push(BgpPeer::read_peer(&mut value)?)
        }

        assert!(
            value.is_empty(),
            "Index table buffer finished with {} bytes remaining",
            value.len()
        );

        Ok(PeerIndexTable {
            collector_bgp_id,
            view_name,
            peer_table,
        })
    }
}

bitflags! {
    struct BgpPeerType: u8 {
        const WIDE_ASN = 0b0000_0010;
        const IPV6 = 0b0000_0001;
    }
}

#[derive(Debug)]
pub struct BgpPeer {
    pub peer_id: u32,
    pub peer_address: IpAddr,
    pub peer_asn: u32,
}

impl BgpPeer {
    fn read_peer(buffer: &mut &[u8]) -> io::Result<Self> {
        let peer_type =
            BgpPeerType::from_bits(buffer.read_u8()?).ok_or_else(|| Error::from(InvalidData))?;

        let peer_id = buffer.read_u32::<BigEndian>()?;

        let peer_address = if peer_type.contains(BgpPeerType::IPV6) {
            let address = buffer.read_u128::<BigEndian>()?;
            Ipv6Addr::from(address.to_ne_bytes()).into()
        } else {
            let address = buffer.read_u32::<BigEndian>()?;
            Ipv4Addr::from(address.to_ne_bytes()).into()
        };

        let peer_asn = if peer_type.contains(BgpPeerType::WIDE_ASN) {
            buffer.read_u32::<BigEndian>()?
        } else {
            buffer.read_u16::<BigEndian>()? as u32
        };

        Ok(BgpPeer {
            peer_id,
            peer_address,
            peer_asn,
        })
    }
}
