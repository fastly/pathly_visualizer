use crate::ripe_atlas::{AddressFamily, Protocol, UnixTimestamp};
use serde::{Deserialize, Serialize};
use std::borrow::Cow;

/// https://atlas.ripe.net/docs/apis/result-format/#version-4570
#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct Traceroute<'a> {
    pub fw: u32,
    /// address family, 4 or 6 (integer)
    pub af: AddressFamily,
    /// IP address of the destination (string)
    ///
    /// > Note: Will not be present in cases where traceroute failed to resolve the host name.
    pub dst_addr: Option<Cow<'a, str>>,
    /// name of the destination (string)
    pub dst_name: Cow<'a, str>,
    /// Unix timestamp for end of measurement (int)
    pub endtime: UnixTimestamp,
    /// IP address of the probe as know by controller (string)
    pub from: Cow<'a, str>,
    /// measurement identifier (int)
    pub msm_id: i64,
    /// variation for the Paris mode of traceroute (int)
    ///
    /// > Note: For some reason this value is not always present. The specification says it should
    /// > so I don't know why it gets excluded sometimes. Noted from response with fw 4790.
    pub paris_id: Option<i64>,
    /// source probe ID (int)
    pub prb_id: i64,
    /// "UDP" or "ICMP" (or "TCP", fw >= 4600) (string)
    pub proto: Protocol,
    /// list of hop elements (array)
    pub result: Vec<TraceHop<'a>>,
    /// packet size (int)
    pub size: u64,
    /// source address used by probe (string)
    ///
    /// > Note: Will not be present in cases where traceroute failed to resolve the host name.
    /// > However, this should be the same as the `from` field unless I am misunderstanding
    /// > something.
    pub src_addr: Option<Cow<'a, str>>,
    /// Unix timestamp for start of measurement (int)
    pub timestamp: UnixTimestamp,
    /// "traceroute" (string)
    pub r#type: Cow<'a, str>,
}

// impl<'a> Traceroute<'a> {
//     pub fn memory_footprint(&self) -> usize {
//         let mut footprint = std::mem::size_of::<Self>();
//
//
//     }
//
//     // pub fn experimental_write<W: WriteBytesExt>(&self, writer: &mut W) -> io::Result<()> {
//     //     writer.write_u32::<LittleEndian>(self.fw)?;
//     //     writer.write_u8(self.af as u8)?;
//     //     Ok(())
//     // }
//     //
//     // pub fn experimental_write<W: WriteBytesExt>(&self, writer: &mut W) -> io::Result<()> {
//     //     writer.write_u32::<LittleEndian>(self.fw)?;
//     //     writer.write_u8(self.af as u8)?;
//     //     Ok(())
//     // }
// }

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum TraceHop<'a> {
    Error {
        error: Cow<'a, str>,
    },
    Result {
        hop: u32,
        result: Vec<TraceReply<'a>>,
    },
}

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum TraceReply<'a> {
    Timeout {
        /// Always "*"
        x: Cow<'a, str>,
    },
    Error {
        /// The specification does not include this option, but it sometimes comes up if a TraceHop
        /// type error comes up mid-run.
        error: Option<Cow<'a, str>>,
    },
    Reply {
        /// (optional) error ICMP: "N" (network unreachable,), "H" (destination unreachable),
        /// "A" (administratively prohibited), "P" (protocol unreachable), "p" (port unreachable)
        /// (string)
        err: Option<ErrorTypes>,
        /// IPv4 or IPv6 source address in reply (string)
        from: Cow<'a, str>,
        /// (optional) time-to-live in packet that triggered the error ICMP. Omitted if equal to 1 (int)
        #[serde(skip_serializing_if = "omit_icmp_ttl", default = "icmp_default_ttl")]
        ittl: i64,
        #[serde(flatten)]
        rtt: RoundTripTime,
        /// (optional) path MTU from a packet too big ICMP (int)
        mtu: Option<i64>,
        /// size of reply (int)
        size: u64,
        /// time-to-live in reply (int)
        ttl: i64,
        /// (optional) TCP flags in the reply packet, for TCP traceroute, concatenated, in the order
        /// 'F' (FIN), 'S' (SYN), 'R' (RST), 'P' (PSH), 'A' (ACK), 'U' (URG) (fw >= 4600) (string)
        flags: Option<Cow<'a, str>>,
        /// [optional] information when icmp header is found in reply (object)
        icmpext: Option<ICMPHeaderInfo>,
    },
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct ICMPHeaderInfo {
    /// RFC4884 version (int)
    pub version: i64,
    /// "1" if length indication is present, "0" otherwise (int)
    pub rfc4884: u8,
    /// elements of the object (array)
    pub obj: Vec<ICMPObj>,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct ICMPObj {
    /// RFC4884 class (int)
    pub class: i64,
    /// RFC4884 type (int)
    pub r#type: i64,
    /// [optional] MPLS data, RFC4950, shown when class is "1" and type is "1" (array)
    pub mpls: Option<Vec<MPLSData>>,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct MPLSData {
    /// for experimental use (int)
    pub exp: i64,
    /// mpls label (int)
    pub label: i64,
    /// bottom of stack (int)
    pub s: i64,
    /// time to live value (int)
    pub ttl: i64,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub enum RoundTripTime {
    /// round-trip-time of reply, not present when the response is late (float)
    #[serde(rename = "rtt")]
    OnTime(f32),
    /// (optional) number of packets a reply is late, in this case rtt is not present (int)
    #[serde(rename = "late")]
    Late(u32),
}

/// Utility functions which specify if the ttl should be excluded from traceroute hop reply details
fn omit_icmp_ttl(ttl: &i64) -> bool {
    *ttl == 1
}
fn icmp_default_ttl() -> i64 {
    1
}

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum ErrorTypes {
    Code(i32),
    Icmp(ICMPError),
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub enum ICMPError {
    #[serde(rename = "N")]
    NetworkUnreachable,
    #[serde(rename = "H", alias = "h")] // FIXME: Is "h" really a valid variant? I just guessed.
    DestinationUnreachable,
    #[serde(rename = "A")]
    AdministrativelyProhibited,
    #[serde(rename = "P")]
    ProtocolUnreachable,
    #[serde(rename = "p")]
    PortUnreachable,
}
