import React, { useState, useCallback } from 'react';
import Popup from 'reactjs-popup';
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
import { Typography, Popover } from "@material-ui/core";
import { toPng } from 'html-to-image';
import * as htmlToImage from 'html-to-image';


import { Typography, Popover } from "@material-ui/core";

//  below nodes and edges used for testing purposes
import { nodes as initialNodes, edges as initialEdges } from './testElements';

// linking stylesheet
import 'reactflow/dist/style.css';

// using dagre library to auto format graph --> no need to position anything
const dagreGraph = new dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

// default node width and height
const nodeWidth = 172;
const nodeHeight = 36;

// default position for all nodes --> changed for nodes later in getLayout
const position = { x: 0, y: 0 }

// default flowkey --> used for storing flow data locally later
const flowKey = 'example-flow';

function Graph(props) {

    // define here not globally --> avoid rerenders adding multiple of same element into array
    let responseNodes = []
    let responseEdges = []

    let asnNodes = []

    // define here --> set to be auto layouted nodes and edges later
    let layoutedNodes
    let layoutedEdges

    // init nodes and edges from passed in props
    // need to do so in constructNodesEdges to avoid rerenders when nodes are moved in graph
    const constructNodesEdges = React.useMemo(() => {
        // loop through all nodes
        for (let i = 0; i < props.response.nodes.length; i++) {
            let probeIpSplit = props.response.probeIp.split(" / ")

            let asnString
            if (props.response.nodes[i].asn === undefined) {
                asnString = undefined
            }
            else {
                asnString = props.response.nodes[i].asn.toString()
            }

            // clean traceroute data nodes
            if(props.clean){
                // set node to input node if the ip == starting probe ip
                if(props.response.nodes[i].ip === probeIpSplit[0] || props.response.nodes[i].ip === probeIpSplit[1]) {
                    responseNodes.push(
                        {
                            id: props.response.nodes[i].ip,
                            type: 'input',
                            data: {
                                label: props.response.nodes[i].ip,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: props.response.nodes[i].averageRtt,
                                lastUsed: props.response.nodes[i].lastUsed,
                                avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                            },
                            className: 'circle',
                            style: {
                                background: '#E98F91',
                            },
                            parentNode: asnString,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
                // if not starting probe, push as normal node without 'type: input'
                else{
                    responseNodes.push(
                        {
                            id: props.response.nodes[i].ip,
                            data: {
                                label: props.response.nodes[i].ip,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: props.response.nodes[i].averageRtt,
                                lastUsed: props.response.nodes[i].lastUsed,
                                avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                            },
                            className: 'circle',
                            style: {
                                background: '#5DCFE7',
                            },
                            parentNode: asnString,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
            }

            // full traceroute data nodes
            else{
                // set node to input node if the ip == starting probe ip --> also need to verify that the amount of timesinceknown is 0
                if((props.response.nodes[i].id.ip === probeIpSplit[0] || props.response.nodes[i].id.ip === probeIpSplit[1]) && (props.response.nodes[i].id.timeSinceKnown === 0)) {
                    responseNodes.push(
                        {
                            id: props.response.nodes[i].id.ip,
                            type: 'input',
                            data: {
                                label: props.response.nodes[i].id.ip,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: props.response.nodes[i].averageRtt,
                                lastUsed: props.response.nodes[i].lastUsed,
                                avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                            },
                            className: 'circle',
                            style: {
                                background: '#B1E6D6',
                            },
                            parentNode: asnString,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
                // if not starting probe, push as normal node without 'type: input'
                else{
                    // need to check if there are any timeouts in order to set proper id
                    let nodeId = props.response.nodes[i].id.ip
                    //change node color and label based on if node is timeout or not
                    let nodeLabel = nodeId
                    let nodeColor = '#5DCFE7'
                    if(props.response.nodes[i].id.timeSinceKnown > 0){
                        // concat number of timeouts since known onto id
                        nodeId = nodeId + "-" + props.response.nodes[i].id.timeSinceKnown
                        nodeLabel = "*"
                        nodeColor = "#E98F91"
                    }
                    responseNodes.push(
                        {
                            id: nodeId,
                            data: {
                                label: nodeLabel,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: props.response.nodes[i].averageRtt,
                                lastUsed: props.response.nodes[i].lastUsed,
                                avgPathLifespan: props.response.nodes[i].averagePathLifespan,
                            },
                            className: 'circle',
                            style: {
                                background: nodeColor,
                            },
                            parentNode: asnString,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
            }
            // push asn nodes onto node arr --> make sure to only push one of each asn
            if(!asnNodes.includes(props.response.nodes[i].asn) && props.response.nodes[i].asn !== undefined){
                responseNodes.push(
                    {
                        id: props.response.nodes[i].asn.toString(),
                        data: {
                            label: props.response.nodes[i].asn,
                            type: 'asn',
                        },
                        className: 'group',
                        zIndex: 0,
                        position,
                    }
                )
                asnNodes.push(props.response.nodes[i].asn)
            }
        }

        // populate edges using response data
        for(let i = 0; i < props.response.edges.length; i++) {
            // clean traceroute data edges
            if (props.clean) {
                responseEdges.push(
                    {
                        id: props.response.edges[i].start + "-" + props.response.edges[i].end,
                        source: props.response.edges[i].start,
                        target: props.response.edges[i].end,
                        // Add more down here about line weight, etc.
                    }
                )
            }
            // full traceroute data edges
            else {
                // need to change id based on how many timeouts since known
                let edgeSource = props.response.edges[i].start.ip
                let edgeTarget = props.response.edges[i].end.ip
                // get line weight dependent on the amount of outbound coverage coming through that edge
                let labelWeight = (props.response.edges[i].outboundCoverage * 100).toString() + "%"
                let lineWeight = (props.response.edges[i].outboundCoverage).toString() + "%"
                // specify source and target ids, id will be edgeIp + timesinceknown (if timesinceknown > 0)
                if(props.response.edges[i].start.timeSinceKnown > 0) {
                    edgeSource = edgeSource + "-" + props.response.edges[i].start.timeSinceKnown
                }
                if (props.response.edges[i].end.timeSinceKnown > 0) {
                    edgeTarget = edgeTarget + "-" + props.response.edges[i].end.timeSinceKnown
                }
                responseEdges.push(
                    {
                        id: edgeSource + "-" + edgeTarget,
                        source: edgeSource,
                        target: edgeTarget,
                        label: labelWeight,
                        style: {strokeWidth: lineWeight},
                        zIndex: 1,
                    }
                )
            }
        }

        // auto layout function using dagre layout algorithm
        const getLayout = (nodes, edges) => {
            // set default layout to "left to right"
            dagreGraph.setGraph({ rankdir: "LR" });
        
            // set nodes and edges in dagre graph
            nodes.forEach((node) => {
                dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
            })
            edges.forEach((edge) => {
                dagreGraph.setEdge(edge.source, edge.target)
            })

            dagre.layout(dagreGraph)
    
            // used for determining asn positioning 

            // layout positioning
            // sets arrow coming out of source from right and into target from left
            // sets position of each node

            let asnPosMap = new Map()
            //used for determining size of asn boxes
            let asnSizeMap = new Map()
            let asnGroups = []
    
            // loop through each node
            nodes.forEach((node) => {
                const nodeWithPosition = dagreGraph.node(node.id)
                node.targetPosition = "left"
                node.sourcePosition = "right"
                
                // check if node is asn, if not, follow steps:
                // 1) set node position using dagre algorithm
                // 2) check if the asn is undefined, if not, set the asn position in the asnPosMap equal to the position from step 1
                // 3) check if asn is in the posMap, if so, verify that the node is not further outwards (left/up) than the specified asn position
                //      --> if so, set asn position to position of more outwards node
                // 4) set node position to it's current position minus the position of the asn. Nodes in this library are positioned respectively
                //      --> within their specified groups so the positions need to be shrunk down to fit inside the asns
                if(node.data.type !== "asn"){
                    // Step 1
                    node.position = {
                        x: nodeWithPosition.x - nodeWidth / 2,
                        y: nodeWithPosition.y - nodeHeight / 2,
                    }
                    // Step 2
                    if (node.data.asn !== undefined) {
                        if (!asnPosMap.has(node.data.asn)) {
                            asnPosMap.set(node.data.asn, node.position)
                        }
                        // Step 3
                        else if (asnPosMap.get(node.data.asn).y > node.position.y) {
                            asnPosMap.set(node.data.asn, {
                                x: asnPosMap.get(node.data.asn).x,
                                y: node.position.y,
                            })
                        }
                        else if (asnPosMap.get(node.data.asn).x > node.position.x) {
                            asnPosMap.set(node.data.asn, {
                                x: node.position.x,
                                y: asnPosMap.get(node.data.asn).y,
                            })
                        }
                        // Step 4
                        node.position = {
                            x: node.position.x - asnPosMap.get(node.data.asn).x,
                            y: node.position.y - asnPosMap.get(node.data.asn).y,
                        }
                    }
                    if (!asnSizeMap.has(node.data.asn)) {
                        asnSizeMap.set(node.data.asn, {
                            lowWidth: node.position.x + nodeWidth,
                            highWidth: node.position.x + nodeWidth,
                            lowHeight: node.position.y + nodeHeight,
                            highHeight: node.position.y + nodeHeight,
                        })
                    }
                    // Step 5
                    // Need to set asn position by finding the most north, east, south, and westward nodes
                    else {
                        let widthPlusPos = node.position.x + nodeWidth
                        let heightPlusPos = node.position.y + nodeHeight
                        let nodeAsn = asnSizeMap.get(node.data.asn)
                        // westwards
                        if (nodeAsn.lowWidth > widthPlusPos) {
                            asnSizeMap.set(node.data.asn, {
                                lowWidth: widthPlusPos,
                                highWidth: nodeAsn.highWidth,
                                lowHeight: nodeAsn.lowHeight,
                                highHeight: nodeAsn.highHeight,
                            })
                        }
                        // eastwards
                        else if (nodeAsn.highWidth < widthPlusPos) {
                            asnSizeMap.set(node.data.asn, {
                                lowWidth: nodeAsn.lowWidth,
                                highWidth: widthPlusPos,
                                lowHeight: nodeAsn.lowHeight,
                                highHeight: nodeAsn.highHeight,
                            })
                        }
                        // northwards
                        else if (nodeAsn.lowHeight > heightPlusPos) {
                            asnSizeMap.set(node.data.asn, {
                                lowWidth: nodeAsn.lowWidth,
                                highWidth: nodeAsn.highWidth,
                                lowHeight: heightPlusPos,
                                highHeight: nodeAsn.highHeight,
                            })
                        }
                        // southwards
                        else if (nodeAsn.highHeight < heightPlusPos) {
                            asnSizeMap.set(node.data.asn, {
                                lowWidth: nodeAsn.lowWidth,
                                highWidth: nodeAsn.highWidth,
                                lowHeight: nodeAsn.lowHeight,
                                highHeight: heightPlusPos,
                            })
                        }
                    }
                }
                else {
                    asnGroups.push(node)
                }

                return node
            })

            // set asn positions in auto layout using asnPosMap and size using asnSizeMap
            asnGroups.forEach((node) => {
                node.targetPosition = "left"
                node.sourcePosition = "right"

                node.position = {
                    x: asnPosMap.get(node.id).x,
                    y: asnPosMap.get(node.id).y,
                }

                node.style = {
                    width: asnSizeMap.get(node.id).highWidth,
                    height: asnSizeMap.get(node.id).lowHeight + asnSizeMap.get(node.id).highHeight,
                }

                return node
            })

            return { nodes, edges }
        }

        // initialize nodes and edges w/ dagre auto layout
        let layout = getLayout(responseNodes, responseEdges)
        layoutedNodes = layout.nodes
        layoutedEdges = layout.edges
    // Just a reminder down here that everything in constructNodesEdges is within this function to avoid react rerenders which slow down the webpage
    }, [])

    // need these for graph props later
    // use layout nodes and edges from auto layout function
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

    const [anchorEl, setAnchorEl] = React.useState(null);
    const [test, setTest] = React.useState("test");

    //on node click --> create popup w/ information
    const onNodeClick = (event, node) => {
        // set anchor element on node click
        setAnchorEl(event.currentTarget);
        // get node data to display
        setTest(JSON.stringify(node.data));
 
        return node.data
    }

    // need to reset all edges in order to change color of edge
    const onEdgeClick = (event, edge) => {
        let newEdges = []
        // loop through current edges
        for(let i = 0; i < edges.length; i++) {
            // need to check here if the edge is already highlighted, if so, get rid of highlight on click
            let edgeStyle = {stroke: '#FCA119', strokeWidth: edge.style.strokeWidth}
            if(edge.style.stroke === '#FCA119'){
                edgeStyle = {strokeWidth: edge.style.strokeWidth}
            }
            // if the current edge id is the same as the edge id in the loop, then set color of edge id to highlighted color
            if(edges[i].id === edge.id){
                newEdges.push(
                    {
                        id: edge.id,
                        label: edge.label,
                        selected: edge.selected,
                        source: edge.source,
                        style: edgeStyle,
                        target: edge.target,
                        zIndex: 1,
                    }
                )
            }
            // don't change any other edge properties
            else{
                newEdges.push(
                    {
                        id: edges[i].id,
                        label: edges[i].label,
                        selected: edges[i].selected,
                        source: edges[i].source,
                        style: edges[i].style,
                        target: edges[i].target,
                        zIndex: 1,  
                    }
                )
            }
        }
        setEdges(newEdges)
    }

    const onConnect = useCallback((params) => setEdges((eds) => addEdge(params, eds)), []);

    //  saves current graph state to be restored later
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
            // parses flow data from local storage
            const flow = JSON.parse(localStorage.getItem(flowKey));

            // sets nodes, edges, and viewport using saved flow data
            if (flow) {
                const { x = 0, y = 0, zoom = 1 } = flow.viewport;
                setNodes(flow.nodes || []);
                setEdges(flow.edges || []);
                setViewport({ x, y, zoom });
            }

        };

        restoreFlow();
    }, [setNodes, setViewport]);

    // get raw traceroute data from button onclick
    const getRaw = () => {

        // split form object into two (like before) in order to make two requests for ipv4 and 6
        let ipSplit = props.form.destinationIp.split(" / ")
        const ipv4Form = {
            probeId: parseInt(props.form.probeId),
            destinationIp: ipSplit[0]
        }
        const ipv6Form = {
            probeId: parseInt(props.form.probeId),
            destinationIp: ipSplit[1]
        }

        const sendingObjs = [ipv4Form, ipv6Form]

        const xhr = new XMLHttpRequest();

        // loop through to send requests for both ipv4 and 6
        (function loop(i, length) {
            if (i >= length) {
                return
            }

            xhr.open("POST", "http://localhost:8080/api/traceroute/raw", true)
            xhr.setRequestHeader("Content-Type", "application/json")
            // what happens when response is received
            xhr.onreadystatechange = () => {
                if (xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
                    console.log(xhr.response)
                    // TODO download attachment
                    loop(i + 1, length)
                }
            }

            console.log(JSON.stringify(sendingObjs[i]))
            xhr.send(JSON.stringify(sendingObjs[i]))
        })(0, sendingObjs.length)
    }

    // // pass smth in to determine download, save, or reset
    // const handlePopupOnclick = () => {

    // }
    const myFunction = () => {
        var popup = document.getElementById("myPopup");
        popup.classList.toggle("show");
    }

    const nodeTypes = {
        selectorNode: Node
    };


    // const onElementClick = (event, element) => {
    //     setAnchorEl(event.currentTarget);
    //     setTest(JSON.stringify(element.data));
    //     setTest(element.data);
    //     console.log(element.data)
    //     console.log(element.selected)
    // };

    const handleClose = () => {

        setAnchorEl(null);
    };

    // function downloadImage(dataUrl) {
    //     const a = document.createElement('a');

    //     a.setAttribute('download', 'reactflow.png');
    //     a.setAttribute('href', dataUrl);
    //     a.click();
    // }
    // const onDownloadClick = () => {
    //     toPng(document.querySelector('.react-flow'), {
    //         filter: (node) => {
    //             // we don't want to add the minimap and the controls to the image
    //             if (
    //                 node?.classList?.contains('react-flow__minimap') ||
    //                 node?.classList?.contains('react-flow__controls')
    //             ) {
    //                 return false;
    //             }

    //             return true;
    //         },
    //     }).then(downloadImage);
    // }


    return (
        <div style={{height: 600, width: 600, marginBottom: 100}}>
            <h2>{props.response.probeIp} to {props.form.destinationIp}</h2>
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                nodesDraggable={false}
                onNodeClick={onNodeClick}
                onEdgeClick={onEdgeClick}
                onInit={setRfInstance}
                fitView
                attributionPosition="top-right"
            >

                <Controls />
                <MiniMap
                    nodeColor={(n) => {
                        {/* change minimap node color based on node type */}
                        if (n.data.label === "*") return "#E98F91"
                        if (n.type === "input") return "#B1E6D6"
                        else if (n.type === "output") return "#B1E6D6"
                        else if (n.data.type === "asn") return "#aeaeae33"
                        else return "#5DCFE7"
                    }}
                    nodeStrokeWidth={3} zoomable pannable />
                <Background color="#a6b0b4" gap={16} style={{ backgroundColor: "#E8EEF1" }} />
            </ReactFlow>

            <div className="controls">
                <Popup trigger={<button> Download Raw Data </button>}
                    position="top center">
                    <div id="confirmPopup">Download as .txt file?<br></br>
                        <button onClick={onRestore}>Confirm</button></div>
                </Popup>

                <Popup trigger={<button> Save View </button>}
                    position="top center">
                    <div id="confirmPopup">Save current view?<br></br>
                        <button onClick={onSave}>Confirm</button></div>
                </Popup>

                {/* controller for Restore button and popups */}
                <Popup trigger={<button> Restore </button>}
                    position="top center">
                    <div id="confirmPopup">Reset to saved view?<br></br>
                        <button id="confirmButton" onClick={onRestore}>Confirm</button></div>
                </Popup>

                <div id="nodePopups" key={id}>
                    {popupList}
                </div>

                {/* <div class="popup"> <button onClick={multiSelectPopUp}> Tester Node </button>
                    <span class="popuptext" id="myPopup">
                        <div class="popup" id="confirmPopup">Information about Node<br></br>
                    </div></span>
                </div> */}

                {/* component for popups within React Flow window */}
                <Popover
                    id={id}
                    open={open}
                    anchorEl={anchorEl}
                    onClose={handleClose}
                    anchorOrigin={{
                        vertical: "bottom",
                        horizontal: "center"
                    }}
                    transformOrigin={{
                        vertical: "top",
                        horizontal: "center"
                    }}
                >
                    <Typography id="typography">{test}</Typography>
                </Popover> */
            </div>

            {/* don't delete! experimenting to get multiple pop ups to stay on screen 
            <div class="popup" onClick={popupTester}>Pretend NODE
                <span class="popuptext" id="myPopup">Insert Node Info</span>
            </div> */}

        </div >
    )
}

// need this to utilize useReactFlow hook 
function GraphWithProvider(props) {
    return (
        <ReactFlowProvider>
            <Graph {...props} />
        </ReactFlowProvider>
    )
}



export default GraphWithProvider
