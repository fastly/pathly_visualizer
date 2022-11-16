import React, { useState, useCallback } from 'react';
import ReactFlow, {
    addEdge,
    MiniMap,
    Controls,
    Background,
    useNodesState,
    useEdgesState,
    useReactFlow,
    ReactFlowProvider,
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

//default position for all nodes --> changed for nodes later in getLayout
const position = {x: 0, y: 0}

//default flowkey --> used for storing flow data locally later
const flowKey = 'example-flow';

function Graph(props) {
        
    //define here not globally --> avoid rerenders adding multiple of same element into array
    let responseNodes = []
    let responseEdges = []

    //init nodes and edges from passed in props
    for(let i = 0; i < props.response.nodes.length; i++) {
        let probeIpSplit = props.response.probeIp.split(" / ")
        if(props.response.nodes[i].ip === probeIpSplit[0] || props.response.nodes[i].ip === probeIpSplit[1]) {
            responseNodes.push(
                {
                    id: props.response.nodes[i].ip,
                    type: 'input',
                    data: {
                        label: props.response.nodes[i].ip,
                        asn: props.response.nodes[i].asn,
                        avgRtt: props.response.nodes[i].averageRtt,
                        lastUsed: props.response.nodes[i].lastUsed,
                        avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                    },
                    className: 'circle',
                    style: {
                        background: '#E98F91',
                    },
                    position,
                }
            )
        }
        else{
            responseNodes.push(
                {
                    id: props.response.nodes[i].ip,
                    data: {
                        label: props.response.nodes[i].ip,
                        asn: props.response.nodes[i].asn,
                        avgRtt: props.response.nodes[i].averageRtt,
                        lastUsed: props.response.nodes[i].lastUsed,
                        avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                    },
                    className: 'circle',
                    style: {
                        background: '#5DCFE7',
                    },
                    position,
                }
            )
        }
    }

    for(let i = 0; i < props.response.edges.length; i++) {
        responseEdges.push(
            {
                id: props.response.edges[i].start + "-" + props.response.edges[i].end,
                source: props.response.edges[i].start,
                target: props.response.edges[i].end,
                // Add more down here about line weight, etc.
            }
        )
    }
    

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
    const { nodes: layoutedNodes, edges: layoutedEdges} = getLayout(responseNodes, responseEdges)

    // need these for graph props later
    const [nodes, setNodes, onNodesChange] = useNodesState(layoutedNodes);
    const [edges, setEdges, onEdgesChange] = useEdgesState(layoutedEdges);
    const [rfInstance, setRfInstance] = useState(null);
    const { setViewport } = useReactFlow();

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
    
    const onConnect = useCallback((params) => setEdges((eds) => addEdge(params, eds)), []);

    //saves current graph state to be restored later
    const onSave = useCallback(() => {
        if (rfInstance) {
          const flow = rfInstance.toObject();
          // save flow data in local storage to be used later
          localStorage.setItem(flowKey, JSON.stringify(flow));
        }
      }, [rfInstance]);
    
    //restores saved graph state 
    const onRestore = useCallback(() => {
    const restoreFlow = async () => {
        //parses flow data from local storage
        const flow = JSON.parse(localStorage.getItem(flowKey));

        //sets nodes, edges, and viewport using saved flow data
        if (flow) {
            const { x = 0, y = 0, zoom = 1 } = flow.viewport;
            setNodes(flow.nodes || []);
            setEdges(flow.edges || []);
            setViewport({ x, y, zoom });
        }
    };

    restoreFlow();
    }, [setNodes, setViewport]);

    //get raw traceroute data from button onclick
    const getRaw = () => {
        const xhr = new XMLHttpRequest
        xhr.open("POST", "http://localhost:8080/api/traceroute/download", true)
        xhr.setRequestHeader("Content-Type", "application/json")
        // what happens when response is received
        xhr.onreadystatechange = () => {
            if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
                console.log(xhr.response)
                // TODO download attachment
            }
        }

        xhr.send(JSON.stringify(props.form))
    }

    return (
        <div style={{height: 600, width: 600, marginBottom: 100}}>
            <h2>{props.response.probeIp} to Destination</h2>
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                onNodeClick={onNodeClick}
                onInit={setRfInstance}
                fitView
                attributionPosition="top-right"
            >
                <Controls/>
                <MiniMap 
                    nodeColor={(n) => {
                        if(n.type === "input") return "#E98F91"
                        else if(n.type === "output") return "#B1E6D6"
                        else return "#5DCFE7"
                    }}
                    nodeStrokeWidth={3} zoomable pannable />
                <Background color="#a6b0b4" gap={16} style={{backgroundColor: "#E8EEF1"}}/>
            </ReactFlow>
            <div className="controls">
                <button onClick={getRaw}>Raw Data</button>
                <button onClick={onSave}>Save</button>
                <button onClick={onRestore}>Restore</button>
            </div>
        </div>
    )
}

// need this to utilize useReactFlow hook 
function GraphWithProvider(props) {
    return(
        <ReactFlowProvider>
            <Graph {...props}/>
        </ReactFlowProvider>
    )
}

export default GraphWithProvider