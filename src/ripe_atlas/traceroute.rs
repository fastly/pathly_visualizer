use crate::asn::AutonomousSystem;
use crate::ip::{IPv4Address, IPv6Address};
use crate::ripe_atlas::serde_utils::skip_empty_in_vec;
use crate::ripe_atlas::{AddressFamily, Protocol, UnixTimestamp};
use crate::{ASNTable, GeneralMeasurement};
use log::error;
use serde::{Deserialize, Serialize};
use smallvec::{smallvec, SmallVec};
use std::borrow::Cow;
use std::str::FromStr;

/// https://atlas.ripe.net/docs/apis/result-format/#version-4570
#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct Traceroute<'a> {
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
    /// variation for the Paris mode of traceroute (int)
    ///
    /// > Note: For some reason this value is not always present. The specification says it should
    /// > so I don't know why it gets excluded sometimes. Noted from response with fw 4790.
    pub paris_id: Option<i64>,
    /// "UDP" or "ICMP" (or "TCP", fw >= 4600) (string)
    pub proto: Protocol,
    /// list of hop elements (array)
    pub result: Vec<TraceHop<'a>>,
    /// packet size (int)
    pub size: u64,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum TraceHop<'a> {
    Error {
        error: Cow<'a, str>,
    },
    Result {
        hop: u32,
        #[serde(deserialize_with = "skip_empty_in_vec")]
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
        /// The specification does not include this, but it sometimes comes up if a connectivity
        /// type error comes up mid-run.
        error: Cow<'a, str>,
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

impl<'a> GeneralMeasurement<'a, Traceroute<'a>> {
    pub fn iter_route_asns<'b, 'c>(&'b self, asn_table: &'c ASNTable) -> TracerouteASNIter<'b, 'c> {
        TracerouteASNIter {
            trace: self,
            asn_table,
            prev: None,
            index: 0,
        }
    }

    pub fn iter_route(&self) -> TracerouteIPIter {
        TracerouteIPIter {
            trace: self,
            index: 0,
        }
    }
}

pub struct TracerouteASNIter<'a, 'b> {
    trace: &'a GeneralMeasurement<'a, Traceroute<'a>>,
    asn_table: &'b ASNTable,
    prev: Option<SmallVec<[&'b AutonomousSystem; 3]>>,
    index: usize,
}

impl<'a, 'b> TracerouteASNIter<'a, 'b> {
    fn find_asn(&self, ip: &str) -> Option<&'b AutonomousSystem> {
        let result = match self.trace.af {
            AddressFamily::IPv4 => {
                IPv4Address::from_str(ip).map(|x| self.asn_table.asn_for_ipv4(x))
            }
            AddressFamily::IPv6 => {
                IPv6Address::from_str(ip).map(|x| self.asn_table.asn_for_ipv6(x))
            }
        };

        match result {
            Ok(v) => v,
            Err(e) => {
                let ip_version = self.trace.af as u8;
                error!("Failed to parse IPv{} {:?}: {:?}", ip_version, ip, e);
                None
            }
        }
    }

    fn next_hop(&mut self) -> Option<SmallVec<[&'b AutonomousSystem; 3]>> {
        if self.index > self.trace.result.len() + 1 {
            return None;
        }

        self.index += 1;
        if self.index == 1 {
            return Some(
                self.find_asn(self.trace.from.as_ref())
                    .into_iter()
                    .collect(),
            );
        }

        if self.index == self.trace.result.len() + 2 {
            return Some(
                self.find_asn(self.trace.dst_name.as_ref())
                    .into_iter()
                    .collect(),
            );
        }

        match &self.trace.result[self.index - 2] {
            TraceHop::Error { .. } => Some(SmallVec::new()),
            TraceHop::Result { result, .. } => {
                // Collect ip strings from a given hop
                let mut unique_ips: SmallVec<[_; 3]> = result
                    .iter()
                    .filter_map(|x| match x {
                        TraceReply::Reply { from, .. } => Some(from.as_ref()),
                        _ => None,
                    })
                    .collect();
                // Sort and remove duplicate ips from this hop
                unique_ips.sort_unstable();
                unique_ips.dedup();

                Some(
                    unique_ips
                        .into_iter()
                        .filter_map(|x| self.find_asn(x))
                        .collect(),
                )
            }
        }
    }
}

impl<'a, 'b> Iterator for TracerouteASNIter<'a, 'b> {
    type Item = SmallVec<[&'b AutonomousSystem; 3]>;

    fn next(&mut self) -> Option<Self::Item> {
        loop {
            let mut next = self.next_hop()?;

            if next.is_empty() {
                continue;
            }

            next.sort_unstable();
            next.dedup();

            match &mut self.prev {
                None => {
                    self.prev = Some(next.clone());
                    return Some(next);
                }
                Some(previous) if &next != previous => {
                    *previous = next.clone();
                    return Some(next);
                }
                _ => {}
            }
        }
    }
}

pub struct TracerouteIPIter<'a> {
    trace: &'a GeneralMeasurement<'a, Traceroute<'a>>,
    index: usize,
}

// impl<'a> TracerouteIPIter<'a> {
//     fn next_hop(&mut self) -> Option<SmallVec<[&'a str; 3]>> {
//         if self.index > self.trace.result.len() + 1 {
//             return None;
//         }
//
//         self.index += 1;
//         if self.index == 1 {
//             return Some(smallvec![self.trace.from.as_ref()]);
//         }
//
//         if self.index == self.trace.result.len() + 2 {
//             return Some(smallvec![self.trace.dst_name.as_ref()]);
//         }
//
//         match &self.trace.result[self.index - 2] {
//             TraceHop::Error { .. } => Some(SmallVec::new()),
//             TraceHop::Result { result, .. } => {
//                 // Collect ip strings from a given hop
//                 let mut unique_ips: SmallVec<[&str; 3]> = result.iter()
//                     .filter_map(|x| {
//                         match x {
//                             TraceReply::Reply { from, .. } => Some(from.as_ref()),
//                             _ => None,
//                         }
//                     })
//                     .collect();
//
//                 // Sort and remove duplicate ips from this hop
//                 unique_ips.sort_unstable();
//                 unique_ips.dedup();
//                 Some(unique_ips)
//             }
//         }
//     }
// }

impl<'a> Iterator for TracerouteIPIter<'a> {
    type Item = SmallVec<[&'a str; 3]>;

    fn next(&mut self) -> Option<Self::Item> {
        if self.index > self.trace.result.len() + 1 {
            return None;
        }

        self.index += 1;
        if self.index == 1 {
            return Some(smallvec![self.trace.from.as_ref()]);
        }

        if self.index == self.trace.result.len() + 2 {
            return Some(smallvec![self.trace.dst_name.as_ref()]);
        }

        match &self.trace.result[self.index - 2] {
            TraceHop::Error { .. } => Some(SmallVec::new()),
            TraceHop::Result { result, .. } => {
                // Collect ip strings from a given hop
                let mut unique_ips: SmallVec<[&str; 3]> = result
                    .iter()
                    .filter_map(|x| match x {
                        TraceReply::Reply { from, .. } => Some(from.as_ref()),
                        _ => None,
                    })
                    .collect();

                // Sort and remove duplicate ips from this hop
                unique_ips.sort_unstable();
                unique_ips.dedup();
                Some(unique_ips)
            }
        }
        // let mut timeouts = 0;
        // while let Some(next) = self.next_hop() {
        //     if next.is_empty() {
        //         timeouts += 1;
        //         continue
        //     }
        //
        //     if timeouts > 0 {
        //         self.index -= 1;
        //         return Some(Err(Timeout(timeouts)))
        //     }
        //
        //     return Some(Ok(next))
        // }
        //
        // (timeouts > 0).then(|| Err(Timeout(timeouts)))

        // loop {
        //     let next = self.next_hop()?;
        //     if next.is_empty() {
        //         timeouts += 1;
        //         continue;
        //     }
        //
        //     // match self.prev.replace(next.clone()) {
        //     //     None | Some(previous) if &next != previous => return Some(next),
        //     //     _ => {},
        //     // }
        //
        //     // match &mut self.prev {
        //     //     None => {
        //     //         self.prev = Some(next.clone());
        //     //         return Some(next);
        //     //     }
        //     //     Some(previous) if &next != previous => {
        //     //         *previous = next.clone();
        //     //         return Some(next);
        //     //     }
        //     //     _ => {}
        //     // }
        // }
    }
}

pub struct Timeout(pub u32);
