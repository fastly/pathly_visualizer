use crate::ripe_atlas::measurement::Measurement;
use serde::{Deserialize, Serialize};
use serde_repr::{Deserialize_repr, Serialize_repr};
use std::borrow::Cow;
use std::ops::Deref;

pub mod api;
pub mod dns;
pub mod measurement;
mod serde_utils;
pub mod traceroute;

/// A signed value for the unix timestamp just in case some values are negative. This should never
/// happen though, so it may be changed to unsigned.
pub type UnixTimestamp = i64;

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct MeasurementResponse<'a> {
    pub count: u64,
    pub next: Option<String>,
    pub previous: Option<String>,
    pub results: Vec<Measurement<'a>>,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct GeneralMeasurement<'a, T> {
    #[serde(flatten)]
    inner: T,
    pub fw: u32,
    /// IP address of the probe as know by controller (string)
    pub from: Cow<'a, str>,
    /// IP address of the probe as know by controller (string)
    pub group_id: Option<i64>,
    /// last time synchronised. How long ago (in seconds) the clock of the probe was found to be in
    /// sync with that of a controller. The value -1 is used to indicate that the probe does not
    /// know whether it is in sync (int)
    ///
    /// > Note: This value may not be available for systems with firmware prior to version 4749.
    pub lts: Option<i64>,
    /// measurement identifier (int)
    pub msm_id: i64,
    /// measurement type (string)
    pub msm_name: Cow<'a, str>,
    /// source probe ID (int)
    pub prb_id: i64,
    pub src_addr: Option<Cow<'a, str>>,
    pub timestamp: UnixTimestamp,
    pub r#type: Cow<'a, str>,
}

impl<'a, T> Deref for GeneralMeasurement<'a, T> {
    type Target = T;

    fn deref(&self) -> &Self::Target {
        &self.inner
    }
}

#[derive(Copy, Clone, Serialize_repr, Deserialize_repr, Debug)]
#[repr(u8)]
pub enum AddressFamily {
    IPv4 = 4,
    IPv6 = 6,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
#[serde(rename_all = "UPPERCASE")]
pub enum Protocol {
    Udp,
    Tcp,
    Icmp,
}

#[cfg(feature = "bench")]
pub use bench::*;

#[cfg(feature = "bench")]
mod bench {
    use super::*;
    use format_serde_error::SerdeError;
    use log::{error, info};
    use rayon::prelude::*;
    use serde_json::Value;
    use std::io::BufRead;

    use crate::util::bench::{BenchMark, ProgressCounter};

    /// Experimental function to debug the parsing of measurement values and record the time parsing
    /// takes.
    pub fn debug_read<T, R: BufRead>(reader: &mut R) -> anyhow::Result<()>
    where
        for<'a> T: Deserialize<'a>,
    {
        let progress = ProgressCounter::default();

        let read_line = BenchMark::new();
        let parse_json = BenchMark::new();

        let mut buffer = String::new();
        while read_line.on(|| reader.read_line(&mut buffer).map(|x| x != 0))? {
            if let Err(e) = parse_json.on(|| serde_json::from_str::<T>(&buffer)) {
                let err = SerdeError::new(buffer.to_owned(), e);

                let raw_json = serde_json::from_str::<Value>(&buffer)?;
                let prettified = serde_json::to_string_pretty(&raw_json)?;

                error!("{}\nItem #{}:\n{}", prettified, progress.count(), err);
                return Ok(());
            }

            buffer.clear();

            progress.periodic(|count| {
                info!("Working... {}", count);
            });
        }

        info!("Successfully parsed all values in {:?}", progress.elapsed());
        info!("Line Read Time:  {}", read_line);
        info!("JSON Parse Time: {}", parse_json);

        Ok(())
    }

    pub fn debug_read_rayon<T, R: BufRead + Send>(reader: &mut R) -> anyhow::Result<()>
    where
        for<'a> T: Deserialize<'a>,
    {
        let progress = ProgressCounter::default();
        let parse_json = BenchMark::new();

        reader
            .lines()
            .par_bridge()
            .try_for_each(|buffer| -> std::io::Result<()> {
                let buffer = buffer?;

                if let Err(e) = parse_json.on(|| serde_json::from_str::<T>(&buffer)) {
                    let err = SerdeError::new(buffer.to_owned(), e);

                    let raw_json = serde_json::from_str::<Value>(&buffer)?;
                    let prettified = serde_json::to_string_pretty(&raw_json)?;

                    error!("{}\nItem #{}:\n{}", prettified, progress.count(), err);
                    return Ok(());
                }

                progress.periodic(|count| {
                    info!("Working... {}", count);
                });

                Ok(())
            })?;

        info!("Successfully parsed all values in {:?}", progress.elapsed());
        // info!("Line Read Time:  {}", read_line);
        info!("JSON Parse Time: {}", parse_json);

        Ok(())
    }
}
