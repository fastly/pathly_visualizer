use std::collections::HashSet;
use std::fs::File;
use std::hash::{Hash, Hasher};
use std::io;
use std::io::{BufWriter, ErrorKind, Write};
use std::path::Path;

#[derive(Default, Debug)]
pub struct DigraphDotFile {
    clusters: Vec<NodeCluster>,
    edges: HashSet<DirectedEdge>,
    decorated_nodes: HashSet<NodeAttributes>,
}

impl DigraphDotFile {
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

    pub fn save<P: AsRef<Path>>(&self, path: P) -> io::Result<()> {
        let mut file = BufWriter::new(File::create(path)?);
        self.to_writer(&mut file)?;
        file.flush()
    }

    pub fn to_writer<W: Write>(&self, buffer: &mut W) -> io::Result<()> {
        writeln!(buffer, "digraph G {{")?;

        let mut indented = Indented::from(buffer.by_ref());

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
}

impl DirectedEdge {
    pub fn new(src: String, dst: String) -> Self {
        DirectedEdge {
            src,
            dst,
            label: None,
            line_weight: None,
        }
    }

    pub fn has_attributes(&self) -> bool {
        self.label.is_some() || self.line_weight.is_some()
    }

    pub fn label(&mut self, label: String) {
        self.label.replace(label);
    }

    pub fn line_weight(&mut self, weight: f32) {
        self.line_weight.replace(weight);
    }

    fn write<B: Write>(&self, buffer: &mut B) -> io::Result<()> {
        let mut attributes = Vec::new();
        if let Some(label) = &self.label {
            write!(&mut attributes, "label={:?} ", label)?;
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
}

impl NodeCluster {
    pub fn new(name: String, nodes: HashSet<String>) -> Self {
        NodeCluster { name, nodes }
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
            return Err(io::Error::from(ErrorKind::Interrupted));
        }

        Ok(index)
    }

    fn flush(&mut self) -> io::Result<()> {
        self.inner.flush()
    }
}
