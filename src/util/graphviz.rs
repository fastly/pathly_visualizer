use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::hash::{Hash, Hasher};
use std::io;
use std::io::ErrorKind::{BrokenPipe, Other};
use std::io::{BufWriter, Error, ErrorKind, Write};
use std::path::Path;
use std::process::{Command, Stdio};

#[derive(Default, Debug)]
pub struct DigraphDotFile {
    clusters: Vec<NodeCluster>,
    edges: HashSet<DirectedEdge>,
    decorated_nodes: HashSet<NodeAttributes>,
    misc_properties: Vec<String>,
}

impl DigraphDotFile {
    pub fn internalize_cluster_edges(&mut self) {
        let cluster_map: HashMap<String, usize> = self
            .clusters
            .iter()
            .enumerate()
            .flat_map(|(index, cluster)| cluster.nodes.iter().map(move |x| (x.to_owned(), index)))
            .collect();

        let mut old_edges = HashSet::with_capacity(self.edges.capacity());
        std::mem::swap(&mut old_edges, &mut self.edges);

        for edge in old_edges {
            match (cluster_map.get(&edge.src), cluster_map.get(&edge.dst)) {
                (Some(x), Some(y)) if x == y => self.clusters[*x].edges.insert(edge),
                _ => self.edges.insert(edge),
            };
        }
    }

    pub fn remove_global_edge_constraints(&mut self) {
        // I don't think this should require std::mem::take? It should be fine to move out of a
        // mutable reference so long as the value is replaced. Did something get pinned?
        self.edges = std::mem::take(&mut self.edges)
            .into_iter()
            .map(|x| x.constraint(false))
            .collect();
    }

    pub fn set_graph_properties(&mut self, properties: &[&str]) {
        self.misc_properties = properties.iter().copied().map(str::to_owned).collect();
    }

    pub fn add_graph_properties<S: AsRef<str>>(&mut self, property: S) {
        self.misc_properties.push(property.as_ref().to_owned());
    }

    /// Adds a new directed edge to the graph, but will not overwrite existing edges
    pub fn edge(&mut self, src: String, dst: String) {
        let edge = DirectedEdge::new(src, dst);

        if !self.edges.contains(&edge) {
            self.edges.insert(edge);
        }
    }

    /// Add a directed edge. Any existing edges will be overwritten.
    pub fn insert_edge(&mut self, edge: DirectedEdge) {
        self.edges.insert(edge);
    }

    /// Adds a attributes to a node on the graph. Any previous attributes will be replaced.
    pub fn node(&mut self, attr: NodeAttributes) {
        self.decorated_nodes.insert(attr);
    }

    pub fn cluster(&mut self, cluster: NodeCluster) {
        self.clusters.push(cluster);
    }

    pub fn save_png<P: AsRef<Path>>(&self, path: P) -> io::Result<()> {
        let mut child = Command::new("dot")
            .arg("-Tpng")
            .arg(format!("-o{}", path.as_ref().display()))
            .stdout(Stdio::null())
            .stdin(Stdio::piped())
            .spawn()?;

        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| Error::new(BrokenPipe, "Unable to connect to child stdin"))?;

        let mut file = BufWriter::new(stdin);
        self.to_writer(&mut file)?;
        file.flush()?;

        // Explicitly drop file so it get closed and the child process gets EOF
        drop(file);

        match child.wait()?.code() {
            Some(0) => Ok(()),
            None => Err(Error::new(Other, "graphviz exited prematurely")),
            Some(x) => Err(Error::new(
                Other,
                format!("graphviz exited with status code {}", x),
            )),
        }
    }

    pub fn save<P: AsRef<Path>>(&self, path: P) -> io::Result<()> {
        let mut file = BufWriter::new(File::create(path)?);
        self.to_writer(&mut file)?;
        file.flush()
    }

    pub fn to_writer<W: Write>(&self, buffer: &mut W) -> io::Result<()> {
        writeln!(buffer, "digraph G {{")?;

        let mut indented = Indented::from(buffer.by_ref());

        // Add misc properties
        self.misc_properties
            .iter()
            .try_for_each(|x| writeln!(&mut indented, "{};", x))?;

        // Write clusters
        self.clusters
            .iter()
            .enumerate()
            .try_for_each(|(index, x)| x.write_indexed(&mut indented, index))?;

        // Write edges
        self.edges.iter().try_for_each(|x| x.write(&mut indented))?;

        // Write decorated nodes
        self.decorated_nodes
            .iter()
            .try_for_each(|x| x.write(&mut indented))?;

        writeln!(buffer, "}}")
    }
}

#[derive(Debug, Clone)]
pub struct DirectedEdge {
    src: String,
    dst: String,
    label: Option<String>,
    line_weight: Option<f32>,
    constraint: bool,
}

impl DirectedEdge {
    pub fn new(src: String, dst: String) -> Self {
        DirectedEdge {
            src,
            dst,
            label: None,
            line_weight: None,
            constraint: true,
        }
    }

    pub fn has_attributes(&self) -> bool {
        self.label.is_some() || self.line_weight.is_some()
    }

    pub fn label(mut self, label: String) -> Self {
        self.label.replace(label);
        self
    }

    pub fn line_weight(mut self, weight: f32) -> Self {
        self.line_weight.replace(weight);
        self
    }

    pub fn constraint(mut self, constraint: bool) -> Self {
        self.constraint = constraint;
        self
    }

    fn write<B: Write>(&self, buffer: &mut B) -> io::Result<()> {
        let mut attributes = Vec::new();
        if let Some(label) = &self.label {
            write!(&mut attributes, "label={:?} ", label)?;
        }

        if !self.constraint {
            attributes.extend_from_slice(b"constraint=false ");
        }

        if let Some(weight) = self.line_weight {
            if weight > 1.0 {
                write!(&mut attributes, "penwidth={:?} ", weight)?;
            } else if weight < 1.0 {
                write!(&mut attributes, "arrowsize={:?} penwidth={0:?} ", weight)?;
            }
        }

        if !attributes.is_empty() {
            write!(buffer, "{:?} -> {:?} [", &self.src, &self.dst)?;
            buffer.write_all(&attributes[..attributes.len() - " ".len()])?;
            writeln!(buffer, "];")
        } else {
            writeln!(buffer, "{:?} -> {:?};", &self.src, &self.dst)
        }
    }
}

impl Hash for DirectedEdge {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.src.hash(state);
        self.dst.hash(state);
    }
}

impl PartialEq for DirectedEdge {
    fn eq(&self, other: &Self) -> bool {
        (&self.src, &self.dst) == (&other.src, &other.dst)
    }
}

impl Eq for DirectedEdge {}

#[derive(Debug)]
pub struct NodeCluster {
    name: String,
    nodes: HashSet<String>,
    edges: HashSet<DirectedEdge>,
}

impl NodeCluster {
    pub fn new(name: String, nodes: HashSet<String>) -> Self {
        NodeCluster {
            name,
            nodes,
            edges: HashSet::new(),
        }
    }

    pub fn push_node(&mut self, node: String) {
        self.nodes.insert(node);
    }

    fn write_indexed<B: Write>(&self, buffer: &mut B, index: usize) -> io::Result<()> {
        writeln!(buffer, "subgraph cluster_{} {{", index)?;

        let mut indented = Indented::from(buffer.by_ref());
        writeln!(indented, "style=filled;")?;
        writeln!(indented, "color=lightgrey;")?;
        writeln!(indented, "node [style=filled,color=white];")?;
        writeln!(indented, "label={:?};", &self.name)?;

        for node in &self.nodes {
            writeln!(indented, "{:?};", node)?;
        }

        for edge in &self.edges {
            edge.write(&mut indented)?;
        }

        writeln!(buffer, "}}")?;
        Ok(())
    }
}

#[derive(Debug)]
pub struct NodeAttributes {
    node: String,
    label: Option<String>,
    style: Option<String>,
    color: Option<String>,
}

impl Hash for NodeAttributes {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.node.hash(state);
    }
}

impl PartialEq for NodeAttributes {
    fn eq(&self, other: &Self) -> bool {
        self.node == other.node
    }
}

impl Eq for NodeAttributes {}

impl NodeAttributes {
    pub fn new(node: String) -> Self {
        NodeAttributes {
            node,
            label: None,
            style: None,
            color: None,
        }
    }

    pub fn label<S: AsRef<str>>(mut self, label: S) -> Self {
        self.label.replace(label.as_ref().to_owned());
        self
    }

    pub fn style<S: AsRef<str>>(mut self, style: S) -> Self {
        self.style.replace(style.as_ref().to_owned());
        self
    }

    pub fn color<S: AsRef<str>>(mut self, fill: S) -> Self {
        self.color.replace(fill.as_ref().to_owned());
        self
    }

    fn write<B: Write>(&self, buffer: &mut B) -> io::Result<()> {
        let mut attributes = Vec::new();

        self.label
            .as_ref()
            .map(|x| write!(&mut attributes, "label={:?} ", x))
            .transpose()?;
        self.style
            .as_ref()
            .map(|x| write!(&mut attributes, "style={:?} ", x))
            .transpose()?;
        self.color
            .as_ref()
            .map(|x| write!(&mut attributes, "color={:?} ", x))
            .transpose()?;

        if !attributes.is_empty() {
            write!(buffer, "{:?} [", &self.node)?;
            buffer.write_all(&attributes[..attributes.len() - " ".len()])?;
            writeln!(buffer, "];")?;
        }
        Ok(())
    }
}

/// A simple struct which wraps around another writer and puts a tab before each line written
struct Indented<W> {
    inner: W,
    line_break: bool,
}

impl<W> From<W> for Indented<W> {
    fn from(writer: W) -> Self {
        Indented {
            inner: writer,
            line_break: true,
        }
    }
}

impl<W: Write> Write for Indented<W> {
    fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        let mut hidden_bytes_written = 0;
        let mut index = 0;

        while index < buf.len() {
            if self.line_break {
                let bytes_written = self.inner.write(&[b'\t'])?;
                if bytes_written == 0 {
                    break;
                }
                hidden_bytes_written += bytes_written;
                self.line_break = false;
            }

            match buf[index..].iter().position(|x| *x == b'\n') {
                None => {
                    index += self.inner.write(&buf[index..])?;
                    break;
                }
                Some(next_break) => {
                    let bytes_written = self.inner.write(&buf[index..index + next_break + 1])?;
                    index += bytes_written;
                    if bytes_written < next_break + 1 {
                        break;
                    }
                    self.line_break = true;
                }
            }
        }

        // If we return Ok(0) when we were actually able to write some hidden bytes, then the caller
        // may get the incorrect impression that no more bytes can be written. In this case give an
        // interrupted error to request that the operation be retried.
        if index == 0 && hidden_bytes_written > 0 {
            return Err(Error::from(ErrorKind::Interrupted));
        }

        Ok(index)
    }

    fn flush(&mut self) -> io::Result<()> {
        self.inner.flush()
    }
}
