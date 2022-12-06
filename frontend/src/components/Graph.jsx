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
    getRectOfNodes,
} from 'reactflow';
import dagre from 'dagre'
import { Typography, Popover } from "@material-ui/core";
import { toPng } from 'html-to-image';
import * as htmlToImage from 'html-to-image';

// linking reactflow stylesheet
import 'reactflow/dist/style.css';

// predetermined array of colors tested for colorblindness and contrast
import { asColors } from './ColorData';

// default node width and height
const nodeWidth = 140;
const nodeHeight = 70;

// default position for all nodes --> changed for nodes later in getLayout
const position = { x: 0, y: 0 }

// default flowkey --> used for storing flow data locally later
const flowKey = 'example-flow';

let asnColorMap = new Map()
let nxtColorArr = asColors

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
        for(const currNode of props.response.nodes) {
            let probeIpSplit = props.response.probeIp.split(" / ")
            
            let asnString = undefined
            if(currNode.asn !== undefined){
                asnString = currNode.asn.toString()
            }
            let asnParent = undefined
            // verify user wants asn boxes to be rendered before defining asn boxes as parent nodes
            if(!props.asnSetting){
                if(currNode.asn !== undefined){
                    asnParent = asnString
                }
            }
            
            // clean traceroute data nodes
            if(props.clean){

                let nodeStyle = {
                    background: '#5DCFE7'
                }
                //if asns by color --> set to random color
                if(props.asnSetting){
                    if(asnColorMap.has(asnString)){
                        nodeStyle = asnColorMap.get(asnString)
                    } else{
                        if(nxtColorArr.length > 0){
                            asnColorMap.set(asnString, nxtColorArr[0])
                            nodeStyle = nxtColorArr[0]
                            nxtColorArr.shift()
                        } else{
                            nxtColorArr = asColors
                            asnColorMap.set(asnString, nxtColorArr[0])
                            nodeStyle = nxtColorArr[0]
                            nxtColorArr.shift()
                        }
                    }
                }
                
                let nodeObj = {
                    id: currNode.id,
                    data: {
                        label: currNode.id,
                        type: 'ip',
                        asn: asnString,
                        avgRtt: currNode.averageRtt,
                        lastUsed: currNode.lastUsed,
                        avgPathLifespan: currNode.averagePathLifespan,
                    },
                    className: 'circle',
                    style: nodeStyle,
                    parentNode: asnParent,
                    extent: 'parent',
                    zIndex: 1,
                    position,
                }

                // set node to input node if the ip == starting probe ip    
                if(probeIpSplit.includes(currNode.id)) {
                    let inputObj = nodeObj
                    inputObj.type = 'input'
                    inputObj.style = {
                        background: '#B1E6D6',
                    }
                    responseNodes.push(inputObj)
                    
                }
                // if not starting probe, push as normal node without 'type: input'
                else{
                    responseNodes.push(nodeObj)
                }
            }

            // full traceroute data nodes
            else{
                // set node to input node if the ip == starting probe ip --> also need to verify that the amount of timesinceknown is 0
                if((probeIpSplit.includes(currNode.id.ip)) && (currNode.id.timeSinceKnown === 0)) {
                    responseNodes.push(
                        {
                            id: currNode.id.ip,
                            type: 'input',
                            data: {
                                label: currNode.id.ip,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: currNode.averageRtt,
                                lastUsed: currNode.lastUsed,
                                avgPathLifespan: currNode.averagePathLifespan,
                            },
                            className: 'circle',
                            style: {
                                background: '#B1E6D6',
                            },
                            parentNode: asnParent,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
                // if not starting probe, push as normal node without 'type: input'
                else{
                    // need to check if there are any timeouts in order to set proper id
                    let nodeId = currNode.id.ip
                    //change node color and label based on if node is timeout or not
                    let nodeLabel = nodeId
                    let nodeStyle = {
                        background: '#5DCFE7'
                    }
                    // set node colors according to asn if requested
                    if(props.asnSetting){
                        if(asnColorMap.has(asnString)){
                            nodeStyle = asnColorMap.get(asnString)
                        } else{
                            if(nxtColorArr.length > 0){
                                asnColorMap.set(asnString, nxtColorArr[0])
                                nodeStyle = nxtColorArr[0]
                                nxtColorArr.shift()
                            } else{
                                nxtColorArr = asColors
                                asnColorMap.set(asnString, nxtColorArr[0])
                                nodeStyle = nxtColorArr[0]
                                nxtColorArr.shift()
                            }
                        }
                    }
                    if(currNode.id.timeSinceKnown > 0){
                        // concat number of timeouts since known onto id
                        nodeId = nodeId + "-" + currNode.id.timeSinceKnown
                        nodeLabel = "*"
                        nodeStyle = {
                            background: "#E98F91",
                        }
                    }
                    responseNodes.push(
                        {
                            id: nodeId,
                            data: {
                                label: nodeLabel,
                                type: 'ip',
                                asn: asnString,
                                avgRtt: currNode.averageRtt,
                                lastUsed: currNode.lastUsed,
                                avgPathLifespan: currNode.averagePathLifespan,
                            },
                            className: 'circle',
                            style: nodeStyle,
                            parentNode: asnParent,
                            extent: 'parent',
                            zIndex: 1,
                            position,
                        }
                    )
                }
            }
            // push asn nodes onto node arr --> make sure to only push one of each asn
            // verify they want asn boxes to be rendered
            if(!props.asnSetting){
                if(!asnNodes.includes(currNode.asn) && currNode.asn !== undefined){
                    responseNodes.push(
                        {
                            id: currNode.asn.toString(),
                            data: {
                                label: currNode.asn,
                                type: 'asn',
                            },
                            className: 'group',
                            zIndex: 0,
                            position,
                        }
                    )
        
                    asnNodes.push(currNode.asn)
                }
            }
        }

        // populate edges using response data
        for(let i = 0; i < props.response.edges.length; i++) {
            // clean traceroute data edges
            if (props.clean) {
                let labelWeight = (props.response.edges[i].outboundCoverage * 100).toString() + "%"
                let lineWeight = (props.response.edges[i].outboundCoverage).toString() + "%"
                responseEdges.push(
                    {
                        id: props.response.edges[i].start + "-" + props.response.edges[i].end,
                        source: props.response.edges[i].start,
                        target: props.response.edges[i].end,
                        label: labelWeight,
                        style: {strokeWidth: lineWeight},
                        zIndex: 1,
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
            // using dagre library to auto format graph --> no need to position anything
            const dagreGraph = new dagre.graphlib.Graph();
            dagreGraph.setDefaultEdgeLabel(() => ({}));
            // set default layout to "left to right"
            dagreGraph.setGraph({ rankdir: "LR" })
            // set nodes and edges in dagre graph
            nodes.forEach((node) => {
                dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
            })
            if(!(edges === undefined)){
                edges.forEach((edge) => {
                    dagreGraph.setEdge(edge.source, edge.target)
                })
            }

            dagre.layout(dagreGraph)

            // loop through each node
            nodes.forEach((node) => {
                const nodeWithPosition = dagreGraph.node(node.id)
                node.targetPosition = "left"
                node.sourcePosition = "right"
                  
                if(node.data.type === "asn"){
                    node.position = {
                        x: nodeWithPosition.x - (nodes.length * 20) / 2,
                        y: nodeWithPosition.y - nodeHeight / 2,
                    }
                }
                else{
                    node.position = {
                        x: nodeWithPosition.x - nodeWidth / 2,
                        y: nodeWithPosition.y - nodeHeight / 2,
                    }
                }
                
                
                return node
            })
    
            return {nodes, edges}
        }
    
        // initialize nodes and edges w/ dagre auto layout
        // check if asn boxes or colors
        if(!props.asnSetting){
            let resNodesAsnMap = new Map()
            let asnGroups = []
            let asnNodesMap = new Map()
            let allNodesMap = new Map()
            // Loop through nodes and group them into several arrays based on their asns
            responseNodes.forEach((node) => {
                let nodeAsn = node.data.asn
                if(node.data.type === "asn"){
                    asnGroups.push(node.id)
                } 

                if(nodeAsn !== undefined){
                    asnNodesMap.set(node.id, nodeAsn)
                }
                
                if(resNodesAsnMap.has(nodeAsn)){
                    let nodeArr = resNodesAsnMap.get(nodeAsn)
                    nodeArr.push(node)
                    resNodesAsnMap.set(nodeAsn, nodeArr)
                } else{
                    resNodesAsnMap.set(nodeAsn, [node])
                }

                // need this to create edges between asns and other nodes to get the layout algorithm working
                allNodesMap.set(node.id, nodeAsn)
            })

            let addInvisEdges = responseEdges
            let addedEdges = []
            // adding edges between asns and other nodes so the layout algorithm works
            // edges are invisible, mainly used for auto placement
            responseEdges.forEach((edge) => {
                let src = allNodesMap.get(edge.source)
                let edgeSrc = src
                if(src === undefined){
                    edgeSrc = edge.source               
                }
                let tgt = allNodesMap.get(edge.target)
                let edgeTgt = tgt
                if(tgt === undefined){
                    edgeTgt = edge.target
                }

                let edgeId = edgeSrc + "-" + edgeTgt
                if(!addedEdges.includes(edgeId)){
                    // checking if the source === target (aka is the other node outside of the current asn)
                    if(src !== tgt){
                        addInvisEdges.push({
                            id: edgeId,
                            source: edgeSrc,
                            target: edgeTgt,
                            style: {strokeWidth: 0},
                        })
                        addedEdges.push(edgeId)
                    }
                }
            })

            let resEdgesAsnMap = new Map()
            // loop through edges and group them into multiple arrays based on the asns of their source nodes
            addInvisEdges.forEach((edge) => {
                if(asnNodesMap.has(edge.source)){
                    let asn = asnNodesMap.get(edge.source)
                    if(resEdgesAsnMap.has(asn)){
                        let asnEdges = resEdgesAsnMap.get(asn)
                        asnEdges.push(edge)
                        resEdgesAsnMap.set(asn, asnEdges)
                    } else{
                        resEdgesAsnMap.set(asn, [edge])
                    }
                } else{
                    if(resEdgesAsnMap.has(undefined)){
                        let nonAsnEdges = resEdgesAsnMap.get(undefined)
                        nonAsnEdges.push(edge)
                        resEdgesAsnMap.set(undefined, nonAsnEdges)
                    } else{
                        resEdgesAsnMap.set(undefined, [edge])
                    }
                }
            })

            let lNodes = []
            let lEdges = []
            // push undefined for nodes that don't have asn
            asnGroups.push(undefined)

            let asnSizeMap = new Map()
            // loop through all asns (including undefined aka no asn) and perform auto layout algorithm on all groups of nodes and edges
            for(const asn of asnGroups){
                let currAsnNodes = resNodesAsnMap.get(asn)
                let currAsnEdges = resEdgesAsnMap.get(asn)
                let asnLayout = getLayout(currAsnNodes, currAsnEdges)
                lNodes.push(...asnLayout.nodes)
                if(asnLayout.edges !== undefined){
                    lEdges.push(...asnLayout.edges)
                }
                if(asn !== undefined){
                    asnSizeMap.set(asn, getRectOfNodes(asnLayout.nodes))
                }
            }
            lNodes.forEach((node) => {
                if(node.data.type === "asn"){
                    let size = asnSizeMap.get(node.id)
                    node.style = {
                        width: size.width + nodeWidth,
                        height: size.height + nodeHeight,
                    }
                }
            })
            //let layout = getLayout(responseNodes, responseEdges)
            layoutedNodes = lNodes
            layoutedEdges = lEdges
        }
        // layout without asn boxes, just layout nodes and edges normally
        else{
            let layout = getLayout(responseNodes, responseEdges)
            layoutedNodes = layout.nodes
            layoutedEdges = layout.edges
        }
    // Just a reminder down here that everything in constructNodesEdges is within this function to avoid react rerenders which slow down the webpage
    }, [])

    // need these for graph props later
    // use layout nodes and edges from auto layout function
    const [nodes, setNodes, onNodesChange] = useNodesState(layoutedNodes);
    console.log(nodes)
    const [edges, setEdges, onEdgesChange] = useEdgesState(layoutedEdges);
    const [rfInstance, setRfInstance] = useState(null);
    const { setViewport } = useReactFlow();

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
                if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
                    // create blob of xhr response to create json file with
                    let blob = new Blob([xhr.response], {type: "application/json"})
                    let filename = ""
                    // get content-disposition attachment
                    let disp = xhr.getResponseHeader('Content-Disposition') 
                    
                    if(disp && disp.indexOf('attachment') !== -1){
                        let filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
                        let matches = filenameRegex.exec(disp);
                        if (matches != null && matches[1]) filename = matches[1].replace(/['"]/g, '');
                    }

                    if(typeof window.navigator.msSaveBlob !== 'undefined') {
                        window.navigator.msSaveBlob(blob, filename)
                    }
                    else {
                        let url = window.URL || window.webkitURL
                        let downloadUrl = url.createObjectURL(blob)

                        if(filename) {
                            //create download using new window location
                            let a = document.createElement("a")
                            if(typeof a.download === 'undefined'){
                                window.location.href = downloadUrl
                            }
                            else {
                                a.href = downloadUrl
                                a.download = filename
                                document.body.appendChild(a)
                                a.click()
                            }
                        }
                        else {
                            window.location.href = downloadUrl
                        }

                        //cleanup 
                        setTimeout(function () {
                            url.revokeObjectURL(downloadUrl)
                        }, 100)
                    }
                    //get twice (one for both ipv4 and 6)
                    loop(i+1, length)
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

    const open = Boolean(anchorEl);
    const id = open ? "simple-popover" : undefined;

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
        <div style={{height: 600, width: 1200, marginBottom: 100}}>
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
                </Popover>
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
