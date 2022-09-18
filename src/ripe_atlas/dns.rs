use serde::{Serialize, Deserialize, Deserializer};
use std::borrow::Cow;
use std::collections::HashMap;
use crate::ripe_atlas::{AddressFamily, Protocol};

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct DNSLookup<'a> {
    /// [optional] IP version: "4" or "6" (int)
    af: Option<AddressFamily>,
    /// [optional] instance ID for a collection of related measurement results (int)
    bundle: Option<i64>,
    dst_addr: Option<Cow<'a, str>>,
    dst_name: Option<Cow<'a, str>>,
    error: Option<DNSLookupError<'a>>,
    proto: Option<Protocol>,
    qbuf: Option<Cow<'a, str>>,
    result: Option<DNSResponse<'a>>,
    retry: Option<u32>,

}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct DNSResponse<'a> {
    /// answer count, RFC 1035 4.1.1 (int)
    #[serde(rename = "ANCOUNT")]
    answer_count: u32,
    /// additional record count, RFC 1035, 4.1.1 (int)
    #[serde(rename = "ARCOUNT")]
    additional_record_count: u32,
    /// query ID, RFC 1035 4.1.1 (int)
    #[serde(rename = "ID")]
    id: i64,
    /// name server count (int)
    #[serde(rename = "NSCOUNT")]
    name_server_count: u32,
    /// number of queries (int)
    #[serde(rename = "QDCOUNT")]
    number_of_queries: u32,
    /// answer payload buffer from the server, base64 encoded (string) See example code for decoding
    /// the value
    abuf: Cow<'a, str>,
    /// first two records from the response decoded by the probe, if they are TXT or SOA; other RR
    /// can be decoded from "abuf" (array of objects)
    answers: Option<Vec<DNSRecord<'a>>>,
    /// [optional] response time in milli seconds (float)
    rt: Option<f32>,
    /// [optional] response size (int)
    size: Option<u64>,
    /// [optional] TTL (hop limit for IPv6) field from UDP reply packet (from 5010) (int)
    ttl: Option<u32>,
}

// #[derive(Clone, Serialize, Deserialize, Debug)]
// #[serde(rename_all = "UPPERCASE")]
// pub struct DNSRecord<'a> {
//     mname: Cow<'a, str>,
//     name: Cow<'a, str>,
//     rdata: Vec<Cow<'a, str>>,
//     rname: Cow<'a, str>,
//     serial: i64,
//     ttl: i64,
//     r#type: Cow<'a, str>,
// }

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(tag = "TYPE", deny_unknown_fields)]
pub enum DNSRecord<'a> {
    #[serde(rename_all = "UPPERCASE")]
    TXT {
        // mname: Option<Cow<'a, str>>,
        name: Cow<'a, str>,
        #[serde(deserialize_with = "one_or_many")]
        rdata: Vec<Cow<'a, str>>,
    },
    #[serde(rename_all = "UPPERCASE")]
    SOA {
        mname: Cow<'a, str>,
        name: Cow<'a, str>,
        rname: Cow<'a, str>,
        serial: i64,
        ttl: i64,
    }
}


pub fn one_or_many<'de, D, T>(deserializer: D) -> Result<Vec<T>, D::Error>
    where D: Deserializer<'de>,
        T: Deserialize<'de>,
{
    #[derive(Deserialize)]
    #[serde(untagged)]
    pub enum OneOrMany<T> {
        One(T),
        Many(Vec<T>),
    }

    match OneOrMany::<T>::deserialize(deserializer)? {
        OneOrMany::One(x) => Ok(vec![x]),
        OneOrMany::Many(x) => Ok(x),
    }
}


// #[derive(Clone, Serialize, Deserialize, Debug)]
// pub struct DNSLookupError<'a> {
//     timeout: i64,
//     getaddrinfo: Cow<'a, str>,
// }
// #[derive(Clone, Serialize, Deserialize, Debug)]
// pub enum DNSLookupError<'a> {
//     #[serde(rename = "TUCONNECT")]
//     TLSNegotiation(Cow<'a, str>),
//     /// Failed DNS lookup after timeout in seconds
//     #[serde(rename = "timeout")]
//     Timeout(u64),
//     /// The caller did not provide adequate information to perform the DNS query. This data point is
//     /// useless.
//     #[serde(rename = "evdns_getaddrinfo")]
//     #[serde(alias = "reason")]
//     CallerAtFault(Cow<'a, str>),
//     /// Probably due to network being unreachable
//     #[serde(rename = "socket")]
//     SocketError(Cow<'a, str>),
//     /// I was unable to find any information at all about TU_READ_ERR. I can guess that there was an
//     /// error while reading from IO, but not much more than that as there were 0 hits from Google or
//     /// Bing.
//     #[serde(alias = "TU_READ_ERR")]
//     #[serde(alias = "senderror")]  // I can probably figure out what this is, but idc
//     Unknown(Cow<'a, str>),
// }

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum DNSLookupError<'a> {
    Timeout{
        timeout: u64,
    },
    Other {
        #[serde(flatten)]
        err_map: HashMap<Cow<'a, str>, Cow<'a, str>>,
    }
}

