#![cfg_attr(all(feature = "bench", feature = "nightly"), feature(bench_black_box))]
// TODO: Remove this once I begin making an actual usable program instead of random tests
#![allow(dead_code)]

use crate::asn::ASNTable;
use crate::env::setup_dotenv;
use crate::ripe_atlas::api::{fetch_measurement_results, MeasurementResultsRequest};
use crate::ripe_atlas::traceroute::Traceroute;
use crate::ripe_atlas::GeneralMeasurement;
use crate::traceroute_layout::{build_graph, GraphConfig};
use log::{info, LevelFilter};
use reqwest::Client;
use std::collections::HashSet;
use tokio::runtime::Builder;

mod asn;
mod env;
mod ip;
mod rate_limit;
mod ripe_atlas;
mod traceroute_layout;
mod util;

fn main() {
    // Setup utilities
    setup_logging();
    setup_dotenv();

    // Start async runtime and begin execution of the main application
    Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("Failed to build Tokio async runtime")
        .block_on(async_main());
}

async fn async_main() {
    info!("Fetching ASN table...");
    let asn_table = ASNTable::fetch_and_load();

    let http_client = Client::new();

    let results_request = MeasurementResultsRequest::default();
    let results: Vec<GeneralMeasurement<Traceroute>> =
        fetch_measurement_results(&http_client, 45059282, &results_request)
            .await
            .expect("Successfully retrieved results");

    info!("Received {} results!", results.len());

    let probes: HashSet<_> = results.iter().map(|x| x.prb_id).collect();
    info!("Received results from {} probes", probes.len());

    let asn_table = asn_table.await.expect("was able to fetch ASN table");

    let config = GraphConfig {
        probe_color: Some("lightblue".to_string()),
        cluster_asn: true,
        ..GraphConfig::default()
    };

    let graph = build_graph(&results, &asn_table, &config);

    graph.save("trace_graph.viz").expect("Able to save file");
}

fn setup_logging() {
    pretty_env_logger::formatted_builder()
        .format_timestamp(None)
        .filter_level(LevelFilter::Debug)
        .filter_module("reqwest", LevelFilter::Warn)
        .filter_module("cookie_store", LevelFilter::Warn)
        .init();
}
