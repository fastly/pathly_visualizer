use crate::util::graphviz::{DigraphDotFile, DirectedEdge, NodeAttributes, NodeCluster};
use crate::{ASNTable, GeneralMeasurement, Traceroute};
use itertools::{iproduct, Itertools};
use smallvec::{smallvec, SmallVec};
use std::borrow::Cow;
use std::collections::{HashMap, HashSet};

pub struct GraphConfig {
    pub probe_color: Option<String>,
    pub destination_color: Option<String>,
    pub omit_timeouts: bool,
    pub cluster_asn: bool,
    pub weighted_edges: bool,
    pub outbound_weighted_edges: bool,
    // duplicate_destination: bool,
}

impl Default for GraphConfig {
    fn default() -> Self {
        GraphConfig {
            probe_color: Some("lightblue".to_string()),
            destination_color: None,
            omit_timeouts: true,
            cluster_asn: true,
            weighted_edges: false,
            outbound_weighted_edges: false,
        }
    }
}

pub fn build_graph(
    traces: &[GeneralMeasurement<Traceroute>],
    asn_table: &ASNTable,
    config: &GraphConfig,
) -> DigraphDotFile {
    let mut dot_file = DigraphDotFile::default();

    // Delegate adding edges to the relevant edge handler
    match config {
        GraphConfig {
            omit_timeouts: false,
            outbound_weighted_edges: true,
            ..
        } => apply_edges::<OutboundTimeoutWeightEdgeHandler>(
            traces,
            asn_table,
            config,
            &mut dot_file,
        ),
        GraphConfig {
            omit_timeouts: false,
            ..
        } => apply_edges::<DefaultTimeoutsEdgeHandler>(traces, asn_table, config, &mut dot_file),
        GraphConfig {
            outbound_weighted_edges: true,
            ..
        } => apply_edges::<OutboundWeightEdgeHandler>(traces, asn_table, config, &mut dot_file),
        GraphConfig {
            weighted_edges: true,
            ..
        } => apply_edges::<WeightedEdgeHandler>(traces, asn_table, config, &mut dot_file),
        _ => apply_edges::<DefaultEdgeHandler>(traces, asn_table, config, &mut dot_file),
    };

    if !config.omit_timeouts {
        traces
            .iter()
            .flat_map(|x| x.iter_route_with_timeouts())
            .flatten()
            .filter(|x| x.starts_with("Timeout"))
            .for_each(|x| {
                dot_file.node(
                    NodeAttributes::new(x.to_string()).label("*").color("gray"), // .attr("shape", "point")
                )
            });
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
                .color(color)
                .label(format!("Probe {}", trace.prb_id));

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

fn apply_edges<'a, H: EdgeHandler<'a>>(
    traces: &'a [GeneralMeasurement<Traceroute>],
    asn_table: &'a ASNTable,
    config: &GraphConfig,
    graph: &mut DigraphDotFile,
) {
    let handler = H::new(traces, asn_table, config);
    handler.apply_edges(traces, graph);
}

/// This trait isn't working well, I need to rewrite it so I can do more re-use of related functions
trait EdgeHandler<'a>: Sized {
    fn new(
        traces: &'a [GeneralMeasurement<Traceroute>],
        asn_table: &'a ASNTable,
        config: &GraphConfig,
    ) -> Self;

    fn apply_edges(
        &self,
        traces: &'a [GeneralMeasurement<Traceroute>],
        graph: &mut DigraphDotFile,
    ) {
        for trace in traces {
            trace
                .iter_route()
                .filter(|x| !x.is_empty())
                .dedup()
                .tuple_windows()
                .flat_map(|(src, dst)| iproduct!(src, dst))
                .filter_map(|(src, dst)| self.edge_for(trace, src, dst))
                .for_each(|edge| graph.insert_edge(edge));
        }
    }

    fn edge_for(
        &self,
        _trace: &'a GeneralMeasurement<Traceroute>,
        src: &'a str,
        dst: &'a str,
    ) -> Option<DirectedEdge> {
        Some(DirectedEdge::new(src.to_owned(), dst.to_owned()))
    }
}

struct DefaultEdgeHandler;

impl<'a> EdgeHandler<'a> for DefaultEdgeHandler {
    fn new(
        _traces: &'a [GeneralMeasurement<Traceroute>],
        _asn_table: &'a ASNTable,
        _config: &GraphConfig,
    ) -> Self {
        DefaultEdgeHandler
    }
}

struct WeightedEdgeHandler<'a> {
    edge_weights: HashMap<(&'a str, &'a str), f32>,
    max_weight: f32,
}

impl<'a> EdgeHandler<'a> for WeightedEdgeHandler<'a> {
    fn new(
        traces: &'a [GeneralMeasurement<Traceroute>],
        _asn_table: &'a ASNTable,
        _config: &GraphConfig,
    ) -> Self {
        let mut edge_weights: HashMap<(&str, &str), f32> = HashMap::new();

        for trace in traces {
            trace
                .iter_route()
                .filter(|x| !x.is_empty())
                .dedup()
                .tuple_windows()
                .for_each(|(src, dst)| {
                    let weight = 1.0 / (src.len() * dst.len()) as f32;

                    iproduct!(src, dst).for_each(|x| {
                        *edge_weights.entry(x).or_default() += weight;
                    });
                });
        }

        let max_weight = edge_weights
            .values()
            .copied()
            .reduce(f32::max)
            .unwrap_or_default();
        WeightedEdgeHandler {
            edge_weights,
            max_weight,
        }
    }

    fn edge_for(
        &self,
        _trace: &'a GeneralMeasurement<Traceroute>,
        src: &'a str,
        dst: &'a str,
    ) -> Option<DirectedEdge> {
        let weight = *self.edge_weights.get(&(src, dst))?;

        Some(
            DirectedEdge::new(src.to_owned(), dst.to_owned())
                .weight(weight)
                .line_weight(0.5 + 4.0 * weight / self.max_weight),
        )
    }
}

struct OutboundWeightEdgeHandler<'a> {
    edge_weights: HashMap<&'a str, HashMap<&'a str, f32>>,
}

impl<'a> EdgeHandler<'a> for OutboundWeightEdgeHandler<'a> {
    fn new(
        traces: &'a [GeneralMeasurement<Traceroute>],
        _asn_table: &'a ASNTable,
        _config: &GraphConfig,
    ) -> Self {
        let mut edge_weights = HashMap::new();

        for trace in traces {
            trace
                .iter_route()
                .filter(|x| !x.is_empty())
                .dedup()
                .tuple_windows()
                .flat_map(|(src, dst)| iproduct!(src, dst))
                .for_each(|(src, dst)| {
                    let src_connections: &mut HashMap<&str, f32> =
                        edge_weights.entry(src).or_default();
                    *src_connections.entry(dst).or_default() += 1.0;
                });
        }

        // Convert total usages to average outbound connections
        edge_weights.values_mut().for_each(|map| {
            let total_connections: f32 = map.values().copied().sum();
            map.values_mut().for_each(|x| *x /= total_connections);
        });

        OutboundWeightEdgeHandler { edge_weights }
    }

    fn edge_for(
        &self,
        _trace: &'a GeneralMeasurement<Traceroute>,
        src: &'a str,
        dst: &'a str,
    ) -> Option<DirectedEdge> {
        let weight = *self.edge_weights.get(src)?.get(dst)?;

        Some(
            DirectedEdge::new(src.to_owned(), dst.to_owned())
                .weight(weight)
                .line_weight(0.2 + 5.0 * weight),
        )
    }
}

struct DefaultTimeoutsEdgeHandler;

impl<'a> EdgeHandler<'a> for DefaultTimeoutsEdgeHandler {
    fn new(
        _traces: &'a [GeneralMeasurement<Traceroute>],
        _asn_table: &'a ASNTable,
        _config: &GraphConfig,
    ) -> Self {
        DefaultTimeoutsEdgeHandler
    }

    fn apply_edges(
        &self,
        traces: &'a [GeneralMeasurement<Traceroute>],
        graph: &mut DigraphDotFile,
    ) {
        for trace in traces {
            std::iter::once(SmallVec::<[Cow<str>; 3]>::from_elem(
                trace.from.to_owned(),
                1,
            ))
            .chain(
                trace
                    .iter_route()
                    .map(|x| x.into_iter().map(|y| y.into()).collect()),
            )
            .dedup_with_count()
            .tuple_windows()
            .flat_map(
                |((_, prev), (count, x))| -> Box<dyn Iterator<Item = SmallVec<[Cow<str>; 3]>>> {
                    if !x.is_empty() {
                        Box::new(std::iter::repeat(x).take(count))
                    } else {
                        Box::new(
                            (0..count).map(move |n| {
                                smallvec![format!("{}-{}", prev.join(","), n).into()]
                            }),
                        )
                    }
                },
            )
            .tuple_windows()
            .flat_map(|(src, dst)| iproduct!(src, dst))
            .filter_map(|(src, dst)| self.edge_for(trace, &src, &dst))
            .for_each(|edge| graph.insert_edge(edge));
        }
    }
}

struct OutboundTimeoutWeightEdgeHandler<'a> {
    edge_weights: HashMap<Cow<'a, str>, HashMap<Cow<'a, str>, f32>>,
}

impl<'a> EdgeHandler<'a> for OutboundTimeoutWeightEdgeHandler<'a> {
    fn new(
        traces: &'a [GeneralMeasurement<Traceroute>],
        _asn_table: &'a ASNTable,
        _config: &GraphConfig,
    ) -> Self {
        let mut edge_weights = HashMap::new();

        for trace in traces {
            trace
                .iter_route_with_timeouts()
                .dedup()
                .tuple_windows()
                .flat_map(|(src, dst)| iproduct!(src, dst))
                .for_each(|(src, dst)| {
                    let src_connections: &mut HashMap<Cow<'a, str>, f32> =
                        edge_weights.entry(src).or_default();
                    *src_connections.entry(dst).or_default() += 1.0;
                });
        }

        // Convert total usages to average outbound connections
        edge_weights.values_mut().for_each(|map| {
            let total_connections: f32 = map.values().copied().sum();
            map.values_mut().for_each(|x| *x /= total_connections);
        });

        OutboundTimeoutWeightEdgeHandler { edge_weights }
    }

    fn apply_edges(
        &self,
        traces: &'a [GeneralMeasurement<Traceroute>],
        graph: &mut DigraphDotFile,
    ) {
        for trace in traces {
            trace
                .iter_route_with_timeouts()
                // .iter_route()
                // .filter(|x| !x.is_empty())
                .dedup()
                .tuple_windows()
                .flat_map(|(src, dst)| iproduct!(src, dst))
                .filter_map(|(src, dst)| self.edge_for(trace, &src, &dst))
                .for_each(|edge| graph.insert_edge(edge));
        }
    }

    fn edge_for(
        &self,
        _trace: &'a GeneralMeasurement<Traceroute>,
        src: &'a str,
        dst: &'a str,
    ) -> Option<DirectedEdge> {
        let weight = *self.edge_weights.get(src)?.get(dst)?;

        Some(
            DirectedEdge::new(src.to_owned(), dst.to_owned())
                .weight(weight.powf(1.5))
                .line_weight(0.2 + 5.0 * weight),
        )
    }
}
