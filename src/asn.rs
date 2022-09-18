use crate::ip::{IPv4Address, IPv6Address, IpRange};
use flate2::bufread::GzDecoder;
use log::info;
use std::cmp::Ordering;
use std::collections::{BTreeMap, BTreeSet};
use std::fmt::{Display, Formatter};
use std::fs::File;
use std::io;
use std::io::{BufRead, BufReader};
use std::path::Path;
use std::rc::Rc;
use std::str::FromStr;
use tokio::io::{AsyncWriteExt, BufWriter};

const ASN_TABLE_API: &str = "https://iptoasn.com/data/ip2asn-combined.tsv.gz";
const CACHE_PATH: &str = ".cache/ip2asn-combined.tsv.gz";

pub async fn cache_latest_asn() -> anyhow::Result<()> {
    let file = tokio::fs::File::create(CACHE_PATH);

    info!("Fetching latest ASN tables from {}", ASN_TABLE_API);
    let mut response = reqwest::get(ASN_TABLE_API).await?;

    // tokio equivalent just spawns another thread to do this with regular blocking code
    std::fs::create_dir_all(".cache")?;

    let mut file = file.await?;
    {
        let mut cache_file = BufWriter::new(&mut file);
        while let Some(chunk) = response.chunk().await? {
            cache_file.write_all(chunk.as_ref()).await?;
        }

        cache_file.flush().await?;
    }

    file.sync_data().await?;
    Ok(())
}

#[derive(Debug)]
pub struct AutonomousSystem {
    num: u32,
    country_code: Option<String>,
    name: String,
}

impl PartialEq for AutonomousSystem {
    fn eq(&self, other: &Self) -> bool {
        self.num == other.num
    }
}

impl Eq for AutonomousSystem {}

impl PartialOrd for AutonomousSystem {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        self.num.partial_cmp(&other.num)
    }
}

impl Ord for AutonomousSystem {
    fn cmp(&self, other: &Self) -> Ordering {
        self.num.cmp(&other.num)
    }
}

impl Display for AutonomousSystem {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "AS{} - {}", self.num, &self.name)
    }
}

pub struct ASNTable {
    asns: BTreeSet<Rc<AutonomousSystem>>,
    ipv4: BTreeMap<IpRange<IPv4Address>, Rc<AutonomousSystem>>,
    ipv6: BTreeMap<IpRange<IPv6Address>, Rc<AutonomousSystem>>,
}

impl ASNTable {
    pub async fn fetch_and_load() -> anyhow::Result<Self> {
        cache_latest_asn().await?;
        Ok(Self::from_cached(CACHE_PATH)?)
    }

    pub fn from_cached<P: AsRef<Path>>(path: P) -> io::Result<Self> {
        let file = BufReader::new(File::open(path)?);

        let mut this = ASNTable {
            asns: BTreeSet::new(),
            ipv4: BTreeMap::new(),
            ipv6: BTreeMap::new(),
        };

        let mut reader = BufReader::new(GzDecoder::new(file));
        let mut buffer = String::new();
        while reader.read_line(&mut buffer)? > 0 {
            match this.add_asn_entry(&buffer) {
                Ok(_) => {}                         // Succeeded
                Err(ASNParseError::NotRouted) => {} // Not routed so no need to record it
                Err(e) => println!("Got Error {:?} while reading: {:?}", e, &buffer),
            }

            buffer.clear();
        }

        Ok(this)
    }

    fn add_asn_entry(&mut self, line: &str) -> Result<(), ASNParseError> {
        let mut columns = line.splitn(5, '\t');

        let range_start = columns.next().ok_or(ASNParseError::MissingSection)?;
        let range_end = columns.next().ok_or(ASNParseError::MissingSection)?;
        let asn = columns.next().ok_or(ASNParseError::MissingSection)?;
        let country_code = columns.next().ok_or(ASNParseError::MissingSection)?;
        let name = columns.next().ok_or(ASNParseError::MissingSection)?;

        let asn = match asn.parse() {
            Ok(0) => return Err(ASNParseError::NotRouted),
            Ok(x) => x,
            Err(_) => return Err(ASNParseError::InvalidASN),
        };

        // Create a placeholder on the stack to compare to while searching for existing entries
        let check_placeholder = AutonomousSystem {
            num: asn,
            country_code: None,
            name: String::with_capacity(0),
        };

        let asn_entry = match self.asns.get(&check_placeholder) {
            Some(v) => v.clone(),
            None => {
                let asn = Rc::new(AutonomousSystem {
                    num: asn,
                    country_code: (country_code != "None").then(|| country_code.to_owned()),
                    name: name.to_owned(),
                });

                self.asns.insert(asn.clone());
                asn
            }
        };

        // First attempt to parse addresses as ipv4 then ipv6
        if let (Ok(a), Ok(b)) = (
            IPv4Address::from_str(range_start),
            IPv4Address::from_str(range_end),
        ) {
            self.ipv4.insert(IpRange::new(a, b), asn_entry);
        } else if let (Ok(a), Ok(b)) = (
            IPv6Address::from_str(range_start),
            IPv6Address::from_str(range_end),
        ) {
            self.ipv6.insert(IpRange::new(a, b), asn_entry);
        } else {
            return Err(ASNParseError::FailedToParseIP);
        }

        Ok(())
    }

    pub fn asn_for_ipv4(&self, address: IPv4Address) -> Option<&AutonomousSystem> {
        let mut search_area = self.ipv4.range(..=IpRange::single(address));

        // It should only ever search 1 or 2 entries, but use loop for simplicity
        while let Some((ip_range, asn)) = search_area.next_back() {
            if ip_range.contains(&address) {
                return Some(&*asn);
            } else if ip_range.end < address {
                break;
            }
        }

        None
    }

    pub fn asn_for_ipv6(&self, address: IPv6Address) -> Option<&AutonomousSystem> {
        let mut search_area = self.ipv6.range(..=IpRange::single(address));

        // It should only ever search 1 or 2 entries, but use loop for simplicity
        while let Some((ip_range, asn)) = search_area.next_back() {
            if ip_range.contains(&address) {
                return Some(&*asn);
            } else if ip_range.end < address {
                break;
            }
        }

        None
    }

    pub fn asn_for_ip_str(&self, x: &str) -> Result<Option<&AutonomousSystem>, RequiresDNSLookup> {
        if let Ok(ip) = IPv4Address::from_str(x) {
            return Ok(self.asn_for_ipv4(ip));
        }

        if let Ok(ip) = IPv6Address::from_str(x) {
            return Ok(self.asn_for_ipv6(ip));
        }

        Err(RequiresDNSLookup)
    }
}

pub struct RequiresDNSLookup;

#[derive(Debug, Clone)]
enum ASNParseError {
    MissingSection,
    NotRouted,
    InvalidASN,
    FailedToParseIP,
}
