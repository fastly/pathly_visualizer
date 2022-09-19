use crate::util::graphviz::{DigraphDotFile, NodeAttributes, NodeCluster};
use crate::{ASNTable, GeneralMeasurement, Traceroute};
use itertools::{iproduct, Itertools};
use std::collections::{HashMap, HashSet};

pub struct GraphConfig {
    pub probe_color: Option<String>,
    pub destination_color: Option<String>,
    pub omit_timeouts: bool,
    pub cluster_asn: bool,
    // duplicate_destination: bool,
}

impl Default for GraphConfig {
    fn default() -> Self {
        GraphConfig {
            probe_color: Some("lightblue".to_string()),
            destination_color: None,
            omit_timeouts: true,
            cluster_asn: true,
        }
    }
}

pub fn build_graph(
    traces: &[GeneralMeasurement<Traceroute>],
    asn_table: &ASNTable,
    config: &GraphConfig,
) -> DigraphDotFile {
    let mut dot_file = DigraphDotFile::default();

    if config.omit_timeouts {
        for trace in traces {
            trace
                .iter_route()
                .filter(|x| !x.is_empty())
                .tuple_windows()
                .flat_map(|(src, dst)| iproduct!(src, dst))
                .for_each(|(src, dst)| dot_file.edge(src.to_owned(), dst.to_owned()));
        }
    }

    if config.cluster_asn {
        let all_addresses = traces.iter().flat_map(|x| x.iter_route()).flatten();

        asn_clusters(all_addresses, asn_table)
            .into_iter()
            .map(|(asn, nodes)| NodeCluster::new(format!("AS{asn}"), nodes))
            .for_each(|x| dot_file.cluster(x));
    }

    if let Some(color) = &config.probe_color {
        for trace in traces {
            let attrs = NodeAttributes::new(trace.from.to_string())
                .style("filled")
                .color(color);

            dot_file.node(attrs)
        }
    }

    if let Some(color) = &config.destination_color {
        for trace in traces {
            let attrs = NodeAttributes::new(trace.dst_name.to_string())
                .style("filled")
                .color(color);

            dot_file.node(attrs)
        }
    }

    dot_file
}

pub fn asn_clusters<A>(addresses: A, asn_table: &ASNTable) -> HashMap<u32, HashSet<String>>
where
    A: IntoIterator,
    <A as IntoIterator>::Item: AsRef<str>,
{
    let mut clusters: HashMap<u32, HashSet<String>> = HashMap::new();

    for address in addresses {
        if let Ok(Some(asn)) = asn_table.asn_for_ip_str(address.as_ref()) {
            clusters
                .entry(asn.num)
                .or_default()
                .insert(address.as_ref().to_owned());
        }
    }

    clusters
}
