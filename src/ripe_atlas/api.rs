use crate::env::var;
use crate::ripe_atlas::serde_utils::{is_false, optional_comma_seperated};
use crate::ripe_atlas::{GeneralMeasurement, UnixTimestamp};
use reqwest::Client;
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use std::borrow::Cow;

#[derive(Serialize, Deserialize, Debug, Default)]
pub struct MeasurementResultsRequest<'a> {
    #[serde(flatten)]
    format: MeasurementFormat<'a>,
    #[serde(skip_serializing_if = "Option::is_none")]
    start: Option<UnixTimestamp>,
    #[serde(skip_serializing_if = "Option::is_none")]
    stop: Option<UnixTimestamp>,
    #[serde(
        with = "optional_comma_seperated",
        skip_serializing_if = "Option::is_none"
    )]
    probe_ids: Option<Vec<i64>>,
    #[serde(rename = "anchors-only", skip_serializing_if = "is_false", default)]
    anchors_only: bool,
    #[serde(rename = "public-only", skip_serializing_if = "is_false", default)]
    public_only: bool,
}

#[derive(Serialize, Deserialize, Debug)]
#[serde(rename_all = "lowercase", tag = "format")]
pub enum MeasurementFormat<'a> {
    Json,
    Jsonp {
        #[serde(skip_serializing_if = "Option::is_none")]
        callback: Option<Cow<'a, str>>,
    },
    Txt,
}

impl<'a> Default for MeasurementFormat<'a> {
    fn default() -> Self {
        MeasurementFormat::Json
    }
}

#[derive(Serialize, Deserialize, Debug)]
struct WithApiKey<'a, T> {
    key: Cow<'a, str>,
    #[serde(flatten)]
    data: T,
}

impl<'a, T> From<T> for WithApiKey<'a, T> {
    fn from(data: T) -> Self {
        WithApiKey {
            key: var("ripe_atlas_api_key").into(),
            data,
        }
    }
}

const RIPE_ATLAS_API: &str = "https://atlas.ripe.net";
const GET_MEASUREMENTS_ROUTE: &str = "/api/v2/measurements/";

pub async fn fetch_measurement_results<T: DeserializeOwned>(
    client: &Client,
    pk: u64,
    parameters: &MeasurementResultsRequest<'_>,
) -> reqwest::Result<Vec<GeneralMeasurement<'static, T>>> {
    let url = format!(
        "{}{}{}/results/",
        RIPE_ATLAS_API, GET_MEASUREMENTS_ROUTE, pk
    );

    let with_key = WithApiKey::from(parameters);
    client.get(url).form(&with_key).send().await?.json().await
}
