use std::borrow::Cow;
use serde::{Serialize, Deserialize};
use serde_repr::{Serialize_repr, Deserialize_repr};

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct Measurement<'a> {
    is_public: bool,
    description: Option<Cow<'a, str>>,
    pub id: i64,
    result: Cow<'a, str>,
    pub group_id: Option<i64>,
    af: Option<u8>,
    is_oneoff: bool,
    spread: Option<u64>,
    resolve_on_probe: bool,
    start_time: u64,
    stop_time: Option<u64>,
    r#type: Cow<'a, str>,
    status: MeasurementStatus<'a>,
    is_all_scheduled: bool,
    participant_count: Option<u64>,
    target_asn: Option<i64>,
    target_prefix: Option<Cow<'a, str>>,
    target_ip: Option<Cow<'a, str>>,
    creation_time: u64,
    in_wifi_group: bool,
    resolved_ips: Option<Vec<Cow<'a, str>>>,
    probes_requested: Option<i64>,
    probes_scheduled: Option<i64>,
    group: Option<Cow<'a, str>>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    #[serde(default)]
    probes: Vec<Probe<'a>>,
    estimated_results_per_day: i64,
    credits_per_result: i64,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    #[serde(default)]
    probe_sources: Vec<ProbeSource<'a>>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    #[serde(default)]
    participation_requests: Vec<ParticipationRequest<'a>>,
    tags: Vec<Cow<'a, str>>,
    port: Option<u16>,
    packets: Option<u8>,
    first_hop: Option<u64>,
    max_hops: Option<u64>,
    paris:Option<u8>,
    size: Option<u16>,
    protocol: Option<Cow<'a, str>>,
    response_timeout: Option<u64>,
    duplicate_timeout: Option<u64>,
    hop_by_hop_option_size: Option<u64>,
    destination_option_size: Option<u64>,
    dont_fragment: Option<bool>,
    traffic_class: Option<i64>,
    target: Cow<'a, str>,
    interval: u64,
}


#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct MeasurementStatus<'a> {
    /// Numeric ID of this status
    id: Status,
    /// Human-readable description of this status
    name: Cow<'a, str>,
    /// When the measurement entered this status (not available for every status)
    when: Option<u64>,
}

#[derive(Clone, Serialize, Deserialize, Debug, Default)]
pub struct Probe<'a> {
    /// ID of this probe
    id: i64,
    /// The URL that contains the details of this probe
    url: Cow<'a, str>,
}


#[derive(Clone, Serialize, Deserialize, Debug, Default)]
pub struct ProbeSource<'a> {
    /// Number of probes that have to be added or removed
    requested: u64,
    /// `['area' or 'country' or 'probes' or 'asn' or 'prefix' or 'msm' or '1' or '2' or '3' or '4'
    /// or '5' or '6']` Probe selector. Options are: `area` allows a compass quarter of the world,
    /// `asn` selects an Autonomous System, `country` selects a country, `msm` selects the probes
    /// used in another measurement, `prefix` selects probes based on prefix, `probes` selects
    /// probes directly
    r#type: Cow<'a, str>,
    /// Value for given selector type.
    ///  - `area`: ['WW','West','North-Central','South-Central','North-East','South-East'].
    ///  - `asn`: ASN (integer).
    ///  - `country`: two-letter country code according to ISO 3166-1 alpha-2, e.g. GR.
    ///  - `msm`: measurement id (integer).
    ///  - `prefix`: prefix in CIDR notation, e.g. 193.0.0/16.
    ///  - `probes`: comma-separated list of probe IDs
    value: Cow<'a, str>,
    /// Comma-separated list of probe tags. Only probes with all these tags attached will be
    /// selected from this participation request
    tags_include: Cow<'a, str>,
    /// Comma-separated list of probe tags. Probes with any of these tags attached will be excluded
    /// from this participation request
    tags_exclude: Cow<'a, str>,
}

#[derive(Clone, Serialize, Deserialize, Debug, Default)]
pub struct ParticipationRequest<'a> {
    #[serde(flatten)]
    details: ProbeSource<'a>,
    /// ['add' or 'remove' or '1' or '2']: Action to be applied, or that was applied to the
    /// measurement involved.'add': add probe to the measurement, 'remove': remove probe from the
    /// measurement
    action: Cow<'a, str>,
    /// The unique ID for this participation request
    id: i64,
    /// The creation date and time of the participations request (Defaults to unix timestamp format)
    created_at: Cow<'a, str>,
    /// The (direct) URL of this participations request
    #[serde(rename = "self")]
    self_url: Cow<'a, str>,
}




#[derive(Copy, Clone, Serialize_repr, Deserialize_repr, Debug)]
#[repr(u8)]
pub enum Status {
    Specified = 0,
    Scheduled = 1,
    Ongoing = 2,
    Stopped = 4,
    ForcedToStop = 5,
    NoSuitableProbes = 6,
    Failed = 7,
    Archived = 8,
}
