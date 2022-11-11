import GraphNode from "./GraphNode"
import React, { useCallback } from 'react';
import ReactFlow, {
    addEdge,
    // Can create minimap of graph if needed, not included right now
    MiniMap,
    Controls,
    Background,
    useNodesState,
    useEdgesState,
  } from 'reactflow';
import dagre from 'dagre'
//below nodes and edges used for testing purposes
import { nodes as initialNodes, edges as initialEdges } from './testElements';

import 'reactflow/dist/style.css';

//using dagre library to auto format graph --> no need to position anything
const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

//default node width and height
const nodeWidth = 172;
const nodeHeight = 36;

function Graph(props) {

    //TODO init nodes and edges from passed in props

    const getLayout = (nodes, edges) => {
        //set default layout to "left to right"
        dagreGraph.setGraph({ rankdir: "LR" });
    
        //set nodes and edges in dagre graph
        nodes.forEach((node) => {
            dagreGraph.setNode(node.id, {width: nodeWidth, height: nodeHeight})
        })
        edges.forEach((edge) => {
            dagreGraph.setEdge(edge.source, edge.target)
        })
    
        dagre.layout(dagreGraph)
    
        // layout positioning
        // sets arrow coming out of source from right and into target from left
        // sets position of each node
        nodes.forEach((node) => {
            const nodeWithPosition = dagreGraph.node(node.id)
            node.targetPosition = "left"
            node.sourcePosition = "right"
    
            node.position = {
                x: nodeWithPosition.x - nodeWidth / 2,
                y: nodeWithPosition.y - nodeHeight / 2,
            }
            
            return node
        })

        return {nodes, edges}
    }

    // initialize nodes and edges w/ dagre auto layout
    const { nodes: layoutedNodes, edges: layoutedEdges} = getLayout(initialNodes, initialEdges)
    
    // need these for graph props later
    const [nodes, setNodes, onNodesChange] = useNodesState(layoutedNodes);
    const [edges, setEdges, onEdgesChange] = useEdgesState(layoutedEdges);
    const onConnect = useCallback((params) => setEdges((eds) => addEdge(params, eds)), []);

    // used for different edge types, taken from documentation
    // we are using a bit of a shortcut here to adjust the edge type
    // this could also be done with a custom edge for example
    const edgesWithUpdatedTypes = edges.map((edge) => {
        if (edge.sourceHandle) {
        const edgeType = nodes.find((node) => node.type === 'custom').data.selects[edge.sourceHandle];
        edge.type = edgeType;
        }

        return edge;
    });

    //on node click --> create popup w/ information
    const onNodeClick = (event, node) => {
        console.log(node.data)
        // TODO create popup
    }
    
    return (
        <ReactFlow
            nodes={nodes}
            edges={edgesWithUpdatedTypes}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            fitView
            attributionPosition="top-right"
        >
            <Controls/>
            <Background color="#a6b0b4" gap={16} style={{backgroundColor: "#E8EEF1"}}/>
        </ReactFlow>
        
    )
}

export default Graph