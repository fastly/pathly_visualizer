#![cfg_attr(all(feature = "bench", feature = "nightly"), feature(bench_black_box))]
// TODO: Remove this once I begin making an actual usable program instead of random tests
#![allow(dead_code)]

use crate::asn::ASNTable;
use crate::env::setup_dotenv;
use crate::rate_limit::UsageLimiter;
use crate::ripe_atlas::MeasurementResponse;
use log::{info, LevelFilter};
use tokio::runtime::Builder;

mod asn;
mod env;
mod ip;
mod rate_limit;
mod ripe_atlas;
mod util;

fn main() {
    // Setup utilities
    setup_logging();
    setup_dotenv();

    //
    Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("Failed to build Tokio async runtime")
        .block_on(async_main());
}

async fn async_main() {
    info!("Fetching ASN table...");
    let _ = ASNTable::fetch_and_load().await.unwrap();
}

fn setup_logging() {
    pretty_env_logger::formatted_builder()
        .format_timestamp(None)
        .filter_level(LevelFilter::Debug)
        .filter_module("reqwest", LevelFilter::Warn)
        .filter_module("cookie_store", LevelFilter::Warn)
        .init();
}
