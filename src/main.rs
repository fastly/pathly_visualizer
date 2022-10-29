mod asn;
mod attribute;
mod cache;
mod dump;
mod env;
mod message;
mod util;

use crate::asn::ASNTable;
use crate::cache::PersistentCache;
use crate::dump::index_table::PeerIndexTable;
use crate::dump::rib_entry::parse_rib_subtypes;
use crate::env::setup_dotenv;
use crate::message::{parse_common_header, BGPChunks};
use crate::util::bench::ProgressCounter;
use crate::util::graphviz::{DigraphDotFile, DirectedEdge, NodeCluster};
use crate::util::HumanBytes;
use bgp_models::bgp::{AsPath, AsPathSegment, AttrType, AttributeValue, BgpMessage};
use bgp_models::mrt::EntryType::TABLE_DUMP_V2;
use bgp_models::mrt::{Bgp4Mp, Bgp4MpMessage, MrtMessage, TableDumpV2Message, TableDumpV2Type};
use bgp_models::network::{Afi, Asn};
use bgp_models::prelude::{EntryType, NetworkPrefix};
use bgpkit_parser::parser::mrt::parse_mrt_record_bytes;
use bgpkit_parser::parser::utils::DataBytes;
use bgpkit_parser::ParserError;
use bytes::BytesMut;
use chrono::{Date, DateTime, Datelike, NaiveDate, Utc};
use flate2::bufread::GzDecoder;
use ipnetwork::{IpNetwork, Ipv4Network, Ipv6Network};
use itertools::Itertools;
use log::{debug, error, info, warn, LevelFilter};
use num_traits::{FromPrimitive, ToPrimitive};
use rayon::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use std::fmt::{Display, Formatter};
use std::fs::File;
use std::io::ErrorKind::{InvalidData, Unsupported};
use std::io::{BufReader, BufWriter, Error, Write};
use std::net::Ipv4Addr;
use std::process::exit;
use std::sync::atomic::AtomicU64;
use std::sync::atomic::Ordering::SeqCst;
use std::sync::mpsc::Sender;
use std::sync::{mpsc, Arc, Mutex};
use std::time::Instant;
use std::{io, thread};

const TARGET_ASN: u32 = 54113;

fn main() {
    let application_start_time = Instant::now();
    setup_logging();
    setup_dotenv();

    let mut cache = match PersistentCache::from_env() {
        Ok(v) => v,
        Err(e) => {
            error!("Failed to init http cache: {:?}", e);
            exit(1);
        }
    };

    cache.clear_bak_files();

    let asns = cache
        .get("https://iptoasn.com/data/ip2asn-combined.tsv.gz")
        .expect("Unable to get asn data");
    let asn_table =
        ASNTable::from_reader(&mut BufReader::new(asns)).expect("Unable to create asn table");

    for asn in asn_table.iter_asns() {
        if asn.name.to_ascii_lowercase().contains("fastly") {
            info!("{:?}", asn);
        }
    }

    let (send, recv) = mpsc::channel::<Response>();

    // info!("Guess: {:?}", asn_table.asn_for_ipv4(Ipv4Addr::from([151, 101, 0, 0])));
    // info!("Guess: {:?}", asn_table.asn_for_ipv4(Ipv4Addr::from([151, 101, 64, 0])));
    // info!("Guess: {:?}", asn_table.asn_for_ipv4(Ipv4Addr::from([151, 101, 128, 0])));
    // info!("Guess: {:?}", asn_table.asn_for_ip_str("2a04:4e42::0"));
    // info!("Guess: {:?}", asn_table.asn_for_ip_str("2a04:4e42:200::0"));
    // info!("Guess: {:?}", asn_table.asn_for_ip_str("2a04:4e42:400::0"));
    // info!("Guess: {:?}", asn_table.asn_for_ip_str("2a04:4e42:600::0"));

    // cache.update_cache("https://data.ris.ripe.net/rrc00/2022.01/").unwrap();
    let cache = Arc::new(cache);

    // https://data.ris.ripe.net/rrc00/2022.01/

    // Start 2 days back so we can begin at midnight
    // let mut today = Utc::today().pred().pred();

    // Utc

    // updates.20220128.1600.gz	2022-01-28 16:05	3.7M
    // updates.20220128.1555.gz	2022-01-28 16:00	2.9M
    // updates.20220131.2355.gz	2022-02-01 00:00	2.6M
    // updates.20220129.0000.gz	2022-01-29 00:05	6.0M

    let stats = Arc::new(Stats::default());

    let mut join_handles = Vec::new();

    let cache_handle = cache.clone();
    let stats_handle = stats.clone();

    // Increase the total number of rayon threads available since most will just be waiting on IO
    rayon::ThreadPoolBuilder::new()
        .num_threads(64)
        .build_global()
        .unwrap();

    let join_handle = thread::spawn(move || {
        let progress = Arc::new(ProgressCounter::default());

        // Multihop collectors: 0, 24, 25
        let all_collectors = [
            0, 1, 3, 4, 5, 6, 7, 10, 11, 12, 13, 14, 15, 16, 18, 19, 20, 21, 22, 23, 24, 25, 26,
        ];
        all_collectors
            .into_iter()
            .flat_map(|num| {
                DataIter::recent_for_probe(format!("rrc{:02}", num))
                    .map(move |x| (x, num))
                    .take(30 * 24 * 60 / 5)
            })
            // .map(|file_url| (file_url, ))
            .par_bridge()
            // .map(|(file_url,)| (file_url, &cache_handle, &stats_handle, send.clone(), &progress))
            .for_each_with(
                (cache_handle, stats_handle.clone(), send, progress),
                |(cache, stats, send, progress), (file_url, probe_num)| {
                    search_file(probe_num, &file_url, cache, stats, send, progress)
                },
            );

        info!("{}", stats_handle);
    });
    join_handles.push(join_handle);

    for join_handle in join_handles {
        join_handle.join().unwrap();
    }

    let mut response_file = BufWriter::new(File::create("data/bgp_paths.json.dat").unwrap());
    let mut graph = DigraphDotFile::default();

    let mut target_asn_prefixes = HashSet::new();

    let mut probe_counts: HashMap<u8, (u64, u64)> = HashMap::new();

    let mut collected = Vec::new();
    let mut prefixes = HashSet::new();
    while let Ok(response) = recv.recv() {
        // if response.announced_prefixes[0].is_ipv4() {
        //     continue
        // }

        for prefix in &response.announced_prefixes {
            prefixes.insert(*prefix);
        }

        collected.push(response);
    }

    let mut reduced_prefixes = prefixes.iter().copied().collect();
    let mappings = merge_prefixes(&mut reduced_prefixes);
    info!(
        "Reduced prefixes from {} to {}",
        prefixes.len(),
        reduced_prefixes.len()
    );
    prefixes.clear();
    prefixes.extend(reduced_prefixes);

    // for response in &mut collected {
    //     for prefix in &mut response.announced_prefixes {
    //         if let Some(entry) = mappings.get(prefix) {
    //             *prefix = *entry;
    //         }
    //     }
    //     response.announced_prefixes.sort_unstable();
    //     response.announced_prefixes.dedup();
    // }

    let mut dst_counts: HashMap<IpNetwork, u64> = HashMap::new();

    collected
        .iter()
        .flat_map(|x| x.announced_prefixes.iter())
        .filter(|x| x.is_ipv6())
        .for_each(|x| *dst_counts.entry(*x).or_default() += 1);

    let target_prefix = dst_counts
        .into_iter()
        .reduce(|(x, a), (y, b)| if a > b { (x, a) } else { (y, b) })
        .unwrap()
        .0;

    {
        let mut collector_counts: HashMap<u8, (u64, u64)> = HashMap::new();

        collected.iter().for_each(|x| {
            let (ipv4, ipv6) = collector_counts.entry(x.probe).or_default();
            if x.announced_prefixes.iter().any(|y| y.is_ipv4()) {
                *ipv4 += 1;
            }

            if x.announced_prefixes.iter().any(|y| y.is_ipv6()) {
                *ipv6 += 1;
            }
        });

        let mut ordered = collector_counts
            .into_iter()
            .map(|(k, v)| (v, k))
            .collect::<Vec<_>>();
        ordered.sort_unstable();

        println!("Collector: IPv4 IPv6");
        for ((ipv4, ipv6), probe) in ordered.into_iter().rev() {
            println!("rrc{:02}: {} {}", probe, ipv4, ipv6);
        }
    }

    let mut repeat_asns: HashMap<u32, (usize, usize)> = HashMap::new();
    // while let Ok(mut response) = recv.recv() {
    for mut response in collected {
        serde_json::to_writer(&mut response_file, &response).unwrap();

        if response.asn_path.is_empty() {
            // Shouldn't be possible... right?
            warn!("Found response with empty asn path");
        }

        response.announced_prefixes.retain(|x| {
            // x.is_ipv6()
            *x == target_prefix
            // match x {
            //     // IpNetwork::V4(prefix) => prefix.prefix() <= 20,
            //     IpNetwork::V4(prefix) => *prefix == Ipv4Network::new(Ipv4Addr::from([151, 101, 0, 0]), 16).unwrap(),
            //     // IpNetwork::V4(prefix) => prefix.contains(Ipv4Addr::from([151, 101, 0, 0])),
            //     IpNetwork::V6(_) => false,
            // }
        });

        if response.announced_prefixes.is_empty() {
            continue;
        }

        for prefix in &response.announced_prefixes {
            target_asn_prefixes.insert(prefix.to_string());
        }

        if response.asn_path.len() == 1 {
            for prefix in &response.announced_prefixes {
                graph.edge(format!("rrc{:02}", response.probe), prefix.to_string());
            }
        } else {
            graph.edge(
                format!("rrc{:02}", response.probe),
                format!("AS{}", response.asn_path[0]),
            );

            for pair in response.asn_path[..response.asn_path.len() - 1].windows(2) {
                graph.edge(format!("AS{}", pair[0]), format!("AS{}", pair[1]));
            }

            response
                .asn_path
                .iter()
                .copied()
                .dedup_with_count()
                .for_each(|(n, asn)| {
                    if n > 1 || repeat_asns.contains_key(&asn) {
                        let (min, max) = repeat_asns.entry(asn).or_insert_with(|| (n, n));
                        *min = (*min).min(n);
                        *max = (*max).max(n);
                    }
                });

            for prefix in &response.announced_prefixes {
                graph.edge(
                    format!("AS{}", response.asn_path[response.asn_path.len() - 2]),
                    prefix.to_string(),
                );
            }
        }
    }

    info!("{} repeat asns in paths", repeat_asns.len());
    for (asn, (min, max)) in repeat_asns {
        let asn_str = format!("AS{}", asn);
        let label = if min == max {
            format!("x{}", min)
        } else {
            format!("x{}-{}", min, max)
        };

        graph.insert_edge(DirectedEdge::new(asn_str.clone(), asn_str).label(label));
    }

    graph.cluster(NodeCluster::new(
        format!("AS{}", TARGET_ASN),
        target_asn_prefixes,
    ));
    graph.add_graph_properties("rankdir=\"LR\"");
    graph.add_graph_properties("newrank=true");
    graph.save("bgp_graph").unwrap();
    // graph.save_png("bgp_graph.png").unwrap();

    info!("Finished in {:?}", application_start_time.elapsed());
}

fn merge_prefixes(prefixes: &mut Vec<IpNetwork>) -> HashMap<IpNetwork, IpNetwork> {
    let mut mappings: HashMap<IpNetwork, IpNetwork> = prefixes.iter().map(|x| (*x, *x)).collect();
    let mut had_updates = true;

    while had_updates {
        prefixes.sort_unstable();
        had_updates = {
            let prev_len = prefixes.len();
            prefixes.dedup();
            prev_len != prefixes.len()
        };

        let mut a_idx = 0;
        for b_idx in 1..prefixes.len() {
            match (prefixes[a_idx], prefixes[b_idx]) {
                (a, b) if a == b => {
                    had_updates = true;
                    continue;
                }
                (IpNetwork::V4(a), IpNetwork::V4(b)) if a.is_supernet_of(b) => {
                    mappings.insert(IpNetwork::V4(b), IpNetwork::V4(a));
                    had_updates = true;
                    continue;
                }
                (IpNetwork::V6(a), IpNetwork::V6(b)) if a.is_supernet_of(b) => {
                    mappings.insert(IpNetwork::V6(b), IpNetwork::V6(a));
                    had_updates = true;
                    continue;
                }
                (IpNetwork::V4(a), IpNetwork::V4(b)) if a.prefix() > 0 => {
                    let parent = Ipv4Network::new(a.network(), a.prefix() - 1).unwrap();

                    if a.prefix() == b.prefix() && b.is_subnet_of(parent) {
                        mappings.insert(IpNetwork::V4(a), IpNetwork::V4(parent));
                        mappings.insert(IpNetwork::V4(b), IpNetwork::V4(parent));
                        had_updates = true;
                        prefixes[a_idx] = IpNetwork::V4(parent);
                        continue;
                    }
                }
                (IpNetwork::V6(a), IpNetwork::V6(b)) if a.prefix() > 0 => {
                    let parent = Ipv6Network::new(a.network(), a.prefix() - 1).unwrap();

                    if a.prefix() == b.prefix() && b.is_subnet_of(parent) {
                        mappings.insert(IpNetwork::V6(a), IpNetwork::V6(parent));
                        mappings.insert(IpNetwork::V6(b), IpNetwork::V6(parent));
                        had_updates = true;
                        prefixes[a_idx] = IpNetwork::V6(parent);
                        continue;
                    }
                }
                _ => {}
            }

            a_idx += 1;
            prefixes[a_idx] = prefixes[b_idx];
        }
        prefixes.truncate(a_idx);
    }

    mappings
}

#[derive(Default)]
struct Stats {
    num_files: AtomicU64,
    bytes_read: AtomicU64,
    bgp_msgs: AtomicU64,
    paths_to_destination: AtomicU64,
}

impl Display for Stats {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "Read {} paths to destination ASN in {} BGP messages from {} update files ({})",
            self.paths_to_destination.load(SeqCst),
            self.bgp_msgs.load(SeqCst),
            self.num_files.load(SeqCst),
            HumanBytes(self.bytes_read.load(SeqCst)),
        )
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct Response {
    probe: u8,
    asn_path: Vec<u32>,
    announced_prefixes: Vec<IpNetwork>,
}

fn search_file(
    probe_num: u8,
    file_url: &str,
    cache: &PersistentCache,
    stats: &Stats,
    sender: &Sender<Response>,
    progress: &ProgressCounter,
) {
    let file = {
        // let mut guard = cache.lock().unwrap();
        match cache.get(file_url) {
            Ok(v) => v,
            Err(e) => {
                error!("Failed to get file {}: {:?}", &file_url, e);
                return;
            }
        }
    };

    stats.num_files.fetch_add(1, SeqCst);

    let file = GzDecoder::new(BufReader::new(file));

    let mut total_msgs = 0;
    let mut parsed_msgs = 0;
    let mut skipped_errors = 0;
    let mut msg_types: HashMap<&'static str, u64> = HashMap::new();
    'main_loop: for msg in BGPChunks::new(file) {
        total_msgs += 1;
        let msg = match msg {
            Ok(v) => v,
            Err(e) => {
                error!("Encountered error reading source {}: {:?}", &file_url, e);
                break;
            }
        };

        stats.bgp_msgs.fetch_add(1, SeqCst);
        stats.bytes_read.fetch_add(msg.len() as u64, SeqCst);

        let mut buffer = &*msg;
        let header = match parse_common_header(&mut buffer) {
            Ok(v) => v,
            Err(e) => {
                error!("Unable to read header for source {}: {:?}", &file_url, e);
                break;
            }
        };

        progress.periodic(|_| {
            let (usage, limit) = cache.cache_space();
            info!(
                "{} (cache usage: {}/{})",
                &*stats,
                HumanBytes(usage),
                HumanBytes(limit)
            );
        });

        let mut buffer_cursor = DataBytes::new(buffer);

        let start_time = Instant::now();
        let record = match parse_mrt_record_bytes(&header, &mut buffer_cursor) {
            Ok(v) => v,
            Err(ParserError::IoError(e)) => {
                error!("Encountered IO error: {:?}", e);
                continue;
            }
            Err(_) => {
                skipped_errors += 1;
                continue;
            }
        };

        // *msg_types.entry(mrt_msg_type(&record)).or_default() += 1;

        if bgp_update_asn_target(&record) != Some(TARGET_ASN) {
            continue;
        }

        let bgp4mp = match &record {
            MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessage(x))
            | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4(x))
            | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4Local(x))
            | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageLocal(x)) => x,
            _ => continue,
        };

        let bgp_update = match &bgp4mp.bgp_message {
            BgpMessage::Update(v) => v,
            _ => continue,
        };

        if !bgp_update.withdrawn_prefixes.is_empty() {
            continue;
        }

        let mut prefixes: Vec<IpNetwork> = bgp_update
            .announced_prefixes
            .iter()
            .map(|x| x.prefix)
            .collect();
        let mut path = Vec::new();

        for attribute in &bgp_update.attributes {
            match &attribute.value {
                AttributeValue::AsPath(as_path) | AttributeValue::As4Path(as_path) => {
                    if !path.is_empty() {
                        warn!("BGP message had multiple AS_PATH attributes");
                        continue 'main_loop;
                    }

                    match standard_as_path(&as_path) {
                        Some(x) => path.extend(x.iter().map(|y| y.asn)),
                        None => continue 'main_loop,
                    }
                }
                AttributeValue::MpReachNlri(nlri) => {
                    if !prefixes.is_empty() {
                        warn!("BGP message had multiple MP_REACH_NLRI and/or announced prefixes");
                        continue 'main_loop;
                    }

                    prefixes.extend(nlri.prefixes.iter().map(|x| x.prefix))
                }
                AttributeValue::MpUnreachNlri(_) => continue 'main_loop,
                _ => {}
            }
        }

        if !path.is_empty() && !prefixes.is_empty() {
            stats.paths_to_destination.fetch_add(1, SeqCst);
            let response = Response {
                probe: probe_num,
                asn_path: path,
                announced_prefixes: prefixes,
            };

            sender.send(response).unwrap();
        }

        // match record {
        //     MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessage(x))
        //     | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4(x))
        //     | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4Local(x))
        //     | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageLocal(x)) => {
        //         if x.afi == Ipv6 {
        //             info!("{:?}", &x);
        //             exit(0);
        //         }
        //         if let BgpMessage::Update(y) = x.bgp_message {
        //             if y.announced_prefixes.is_empty() {
        //                 continue;
        //             }
        //
        //             parsed_msgs += 1;
        //             if y.announced_prefixes.iter().any(|z| z.prefix.is_ipv6()) {
        //                 info!("{:?}", &y);
        //                 exit(0);
        //             }
        //
        //             for attribute in &y.attributes {
        //                 if let AttributeValue::AsPath(path) = &attribute.value {
        //                     for item in &path.segments {
        //                         if let AsPathSegment::AsSequence(asn_sequence) = item {
        //                             if let Some(last_asn) = asn_sequence.last() {
        //                                 if last_asn.asn == TARGET_ASN {
        //                                     stats.paths_to_destination.fetch_add(1, SeqCst);
        //                                     let response = Response {
        //                                         probe: probe_num,
        //                                         asn_path: asn_sequence.iter().map(|x| x.asn).collect(),
        //                                         announced_prefixes: y.announced_prefixes.iter().map(|x| x.prefix).collect(),
        //                                     };
        //
        //                                     sender.send(response).unwrap();
        //                                 }
        //                             }
        //                         }
        //                     }
        //                 }
        //             }
        //         }
        //     }
        //     _ => {}  // Don't care
        // }
    }

    // info!("Parsed/Total: {}/{} ({:.03}%, {} errors)", parsed_msgs, total_msgs, 100.0 * parsed_msgs as f64 / total_msgs as f64, skipped_errors);
    // for (key, value) in msg_types {
    //     info!("    {}: {}", key, value);
    // }
}

#[inline]
fn bgp_update_asn_target(msg: &MrtMessage) -> Option<u32> {
    let bgp4mp = match msg {
        MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessage(x))
        | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4(x))
        | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4Local(x))
        | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageLocal(x)) => x,
        _ => return None,
    };

    let bgp_update = match &bgp4mp.bgp_message {
        BgpMessage::Update(v) => v,
        _ => return None,
    };

    bgp_update
        .attributes
        .iter()
        .filter_map(|x| match &x.value {
            AttributeValue::AsPath(v) => Some(v),
            _ => None,
        })
        .next()
        .and_then(|x| standard_as_path(x))
        .and_then(|x| x.last())
        .map(|x| x.asn)
}

#[inline]
fn standard_as_path(path: &AsPath) -> Option<&[Asn]> {
    if path.segments.is_empty() || path.segments.len() > 1 {
        return None;
    }

    match &path.segments[0] {
        AsPathSegment::AsSequence(items) => Some(&items[..]),
        _ => None,
    }
}

fn mrt_msg_type(msg: &MrtMessage) -> &'static str {
    match msg {
        MrtMessage::TableDumpMessage(_) => "TableDumpMessage",
        MrtMessage::TableDumpV2Message(x) => match x {
            TableDumpV2Message::PeerIndexTable(_) => "TableDumpV2Message::PeerIndexTable",
            TableDumpV2Message::RibAfiEntries(_) => "TableDumpV2Message::RibAfiEntries",
            TableDumpV2Message::RibGenericEntries(_) => "TableDumpV2Message::RibGenericEntries",
        },
        MrtMessage::Bgp4Mp(x) => match x {
            Bgp4Mp::Bgp4MpStateChange(_) => "Bgp4Mp::Bgp4MpStateChange",
            Bgp4Mp::Bgp4MpStateChangeAs4(_) => "Bgp4Mp::Bgp4MpStateChangeAs4",
            Bgp4Mp::Bgp4MpMessage(x) => match (&x.bgp_message, x.afi) {
                (BgpMessage::Open(_), Afi::Ipv4) => "Bgp4Mp::Bgp4MpMessage<Ipv4>::Open",
                (BgpMessage::Open(_), Afi::Ipv6) => "Bgp4Mp::Bgp4MpMessage<Ipv6>::Open",
                (BgpMessage::Update(_), Afi::Ipv4) => "Bgp4Mp::Bgp4MpMessage<Ipv4>::Update",
                (BgpMessage::Update(_), Afi::Ipv6) => "Bgp4Mp::Bgp4MpMessage<Ipv6>::Update",
                (BgpMessage::KeepAlive(_), Afi::Ipv4) => {
                    "Bgp4Mp::Bgp4MpMessage<Ipv4>::Notification"
                }
                (BgpMessage::KeepAlive(_), Afi::Ipv6) => {
                    "Bgp4Mp::Bgp4MpMessage<Ipv6>::Notification"
                }
                (BgpMessage::Notification(_), Afi::Ipv4) => {
                    "Bgp4Mp::Bgp4MpMessage<Ipv4>::KeepAlive"
                }
                (BgpMessage::Notification(_), Afi::Ipv6) => {
                    "Bgp4Mp::Bgp4MpMessage<Ipv6>::KeepAlive"
                }
            },
            Bgp4Mp::Bgp4MpMessageLocal(x) => match &x.bgp_message {
                BgpMessage::Open(_) => "Bgp4Mp::Bgp4MpMessageLocal::Open",
                BgpMessage::Update(_) => "Bgp4Mp::Bgp4MpMessageLocal::Update",
                BgpMessage::Notification(_) => "Bgp4Mp::Bgp4MpMessageLocal::Notification",
                BgpMessage::KeepAlive(_) => "Bgp4Mp::Bgp4MpMessageLocal::KeepAlive",
            },
            Bgp4Mp::Bgp4MpMessageAs4(x) => match &x.bgp_message {
                BgpMessage::Open(_) => "Bgp4Mp::Bgp4MpMessageAs4::Open",
                BgpMessage::Update(_) => "Bgp4Mp::Bgp4MpMessageAs4::Update",
                BgpMessage::Notification(_) => "Bgp4Mp::Bgp4MpMessageAs4::Notification",
                BgpMessage::KeepAlive(_) => "Bgp4Mp::Bgp4MpMessageAs4::KeepAlive",
            },
            Bgp4Mp::Bgp4MpMessageAs4Local(x) => match &x.bgp_message {
                BgpMessage::Open(_) => "Bgp4Mp::Bgp4MpMessageAs4Local::Open",
                BgpMessage::Update(_) => "Bgp4Mp::Bgp4MpMessageAs4Local::Update",
                BgpMessage::Notification(_) => "Bgp4Mp::Bgp4MpMessageAs4Local::Notification",
                BgpMessage::KeepAlive(_) => "Bgp4Mp::Bgp4MpMessageAs4Local::KeepAlive",
            },
        },
    }
}

fn run_probe(
    num: u8,
    cache: &Mutex<PersistentCache>,
    stats: &Stats,
    sender: &Sender<Response>,
    progress: &ProgressCounter,
) {
    let probe = format!("rrc{:02}", num);
    info!("Started probe {}", probe);
    let update_files = DataIter::recent_for_probe(probe.to_string());

    // #[derive(Hash, Ord, PartialOrd, Eq, PartialEq, Debug)]
    // struct Entry {
    //     entry: u16,
    //     subtype: u16,
    //     length: u32,
    // }

    // let mut counts: HashMap<Entry, u64> = HashMap::new();
    let mut num_files = 0;

    for file_url in update_files {
        num_files += 1;
        if num_files > 100 {
            break;
        }

        let file = {
            let mut guard = cache.lock().unwrap();
            match guard.get(&file_url) {
                Ok(v) => v,
                Err(e) => {
                    error!("Failed to get file {}: {:?}", &file_url, e);
                    continue;
                }
            }
        };

        stats.num_files.fetch_add(1, SeqCst);

        let file = GzDecoder::new(BufReader::new(file));

        for msg in BGPChunks::new(file) {
            let msg = match msg {
                Ok(v) => v,
                Err(e) => {
                    error!("Encountered error reading source {}: {:?}", &file_url, e);
                    break;
                }
            };

            stats.bgp_msgs.fetch_add(1, SeqCst);
            stats.bytes_read.fetch_add(msg.len() as u64, SeqCst);

            let mut buffer = &*msg;
            let header = match parse_common_header(&mut buffer) {
                Ok(v) => v,
                Err(e) => {
                    error!("Unable to read header for source {}: {:?}", &file_url, e);
                    break;
                }
            };

            progress.periodic(|_| {
                info!("{}", &*stats);
            });

            let mut buffer_cursor = DataBytes::new(buffer);

            let start_time = Instant::now();
            let record = match parse_mrt_record_bytes(&header, &mut buffer_cursor) {
                Ok(v) => v,
                // Ok(v) => println!("Took {:?} to parse:\n{:?}", start_time.elapsed(), v),
                Err(ParserError::IoError(e)) => {
                    error!("Encountered IO error: {:?}", e);
                    continue;
                }
                Err(_) => continue,
            };

            match record {
                MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessage(x))
                | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4(x))
                | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageAs4Local(x))
                | MrtMessage::Bgp4Mp(Bgp4Mp::Bgp4MpMessageLocal(x)) => {
                    if let BgpMessage::Update(y) = x.bgp_message {
                        if y.announced_prefixes.is_empty() {
                            continue;
                        }

                        // I hate this library's data structures. This could have been so much easier
                        for attribute in &y.attributes {
                            if let AttributeValue::AsPath(path) = &attribute.value {
                                for item in &path.segments {
                                    if let AsPathSegment::AsSequence(asn_sequence) = item {
                                        if let Some(last_asn) = asn_sequence.last() {
                                            if last_asn.asn == TARGET_ASN {
                                                stats.paths_to_destination.fetch_add(1, SeqCst);
                                                let response = Response {
                                                    probe: num,
                                                    asn_path: asn_sequence
                                                        .iter()
                                                        .map(|x| x.asn)
                                                        .collect(),
                                                    // announced_prefixes: y.announced_prefixes.clone(),
                                                    announced_prefixes: y
                                                        .announced_prefixes
                                                        .iter()
                                                        .map(|x| x.prefix)
                                                        .collect(),
                                                };

                                                sender.send(response).unwrap();
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    // println!("{:?}", x);
                }
                // MrtMessage::Bgp4Mp(Bgp4Mp::B(x)) => x.bgp_message,
                _ => {} // Don't care
            }

            // std::process::exit(1);

            // let entry = Entry {
            //     entry: header.entry_type.to_u16().unwrap(),
            //     subtype: header.entry_subtype,
            //     length: header.length,
            // };
            //
            // *counts.entry(entry).or_default() += 1;
            //
            // let total_length = stats.length_sum.fetch_add(header.length as u64, SeqCst) + header.length as u64;
            // let complete = stats.total.fetch_add(1, SeqCst) + 1;
            // if complete % 1000000 == 0 {
            //     info!("Completed {} msgs, average length: {}, hash map length: {}, num files: {}", complete, total_length / complete, counts.len(), num_files);
            // }
        }
    }

    // let mut file = BufWriter::new(File::create(format!("data/rrc{:02}.csv", num)).expect("no issues"));
    //
    // writeln!(&mut file, "probe,msg_type,sub_type,length,count").unwrap();
    //
    // for (entry, count) in counts {
    //     let entry_type = EntryType::from_u16(entry.entry).unwrap();
    //     writeln!(&mut file, "rcc{:02},{:?},{},{},{}", num, entry_type, entry.subtype, entry.length, count).unwrap();
    // }
    //
    // file.flush().unwrap();
}

fn setup_logging() {
    pretty_env_logger::formatted_builder()
        .format_timestamp(None)
        .filter_level(LevelFilter::Debug)
        .filter_module("rustls", LevelFilter::Warn)
        .filter_module("ureq", LevelFilter::Warn)
        .filter_module("bgpkit_parser", LevelFilter::Error)
        // .filter_module("reqwest", LevelFilter::Warn)
        // .filter_module("cookie_store", LevelFilter::Warn)
        .init();
}

struct DataIter {
    probe: String,
    date: Date<Utc>,
    time: u32,
}

impl DataIter {
    fn recent_for_probe(probe: String) -> Self {
        DataIter {
            probe,
            date: Utc::today().pred(),
            time: 0,
        }
    }
}

impl Iterator for DataIter {
    type Item = String;

    fn next(&mut self) -> Option<Self::Item> {
        if self.time == 0 {
            self.date = self.date.pred();
            self.time = 2355;
        } else {
            self.time -= 5;
            if self.time % 100 == 95 {
                self.time -= 40;
            }
        }

        Some(format!(
            "https://data.ris.ripe.net/{0}/{1}.{2:02}/updates.{1}{2:02}{3:02}.{4:04}.gz",
            &self.probe,
            self.date.year(),
            self.date.month(),
            self.date.day(),
            self.time
        ))
    }
}

// fn main() -> io::Result<()> {
//     let input_file = "C:\\Users\\Jasper\\Downloads\\bview.20220911.0800.gz";
//     let file = BufReader::new(File::open(input_file)?);
//
//     let file = GzDecoder::new(file);
//
//     let start_time = Instant::now();
//
//     let mut num_msgs = 0;
//     let mut total_length = 0;
//     let mut max_msg_len = 0;
//     let mut attr_counts: HashMap<AttrType, u64> = HashMap::new();
//
//     for msg in BGPChunks::new(file) {
//         let msg = msg?;
//         num_msgs += 1;
//         total_length += msg.len();
//         max_msg_len = max_msg_len.max(msg.len());
//
//         if let Err(e) = handle_bgp_message(msg, &mut attr_counts) {
//             eprintln!("{:?}", e);
//         }
//     }
//
//     // let mut attr_ordered = attr_counts.into_iter().map(|(k, v)| (v, k)).collect::<Vec<_>>();
//     // attr_ordered.sort_unstable();
//     for (k, v) in attr_counts {
//         println!("\t{:?}: {}", k, v);
//     }
//
//     println!("Number of messages: {}", num_msgs);
//     println!("Average message length: {}", total_length / num_msgs);
//     println!("Largest message length: {}", max_msg_len);
//     println!(
//         "Byte Count: {}GB",
//         (total_length as f64) / 1024.0 / 1024.0 / 1024.0
//     );
//
//     println!("Total elapsed time: {:?}", start_time.elapsed());
//     Ok(())
// }

/// I'm still not sure what data I want to collect from this
fn handle_bgp_message(data: BytesMut, attr_counts: &mut HashMap<AttrType, u64>) -> io::Result<()> {
    let mut buffer = &*data;
    let header = parse_common_header(&mut buffer)?;

    if header.entry_type == TABLE_DUMP_V2 {
        let subtype = TableDumpV2Type::from_u16(header.entry_subtype)
            .ok_or_else(|| Error::from(InvalidData))?;

        match subtype {
            TableDumpV2Type::PeerIndexTable => {
                let index_table = PeerIndexTable::try_from(buffer)?;
                for peer in index_table.peer_table {
                    println!("{:?}", peer);
                }

                println!("View name: {:?}", index_table.view_name);
                println!("Succeeded in building index table!");
            }
            TableDumpV2Type::RibIpv4Unicast | TableDumpV2Type::RibIpv6Unicast => {
                // let rib = parse_rib_subtypes(&mut buffer)?;
                // // println!("Unicast: {:?}", rib.entries.len());
                //
                // for entry in &rib.entries {
                //     for attribute in &entry.attribute {
                //         *attr_counts.entry(attribute.attr_type).or_default() += 1;
                //     }
                // }

                assert!(
                    buffer.is_empty(),
                    "Parse did not consume all bytes. {} remaining",
                    buffer.len()
                );
            }
            x => {
                return Err(Error::new(
                    Unsupported,
                    format!("Unexpected BGP Table dump subtype: {:?}", x),
                ));
            }
        }
    }

    Ok(())
}
