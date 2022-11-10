import GraphNode from "./GraphNode"

function Graph(props) {
    
    let nodeArr = []
    // TODO creates graphnodes based on response
    // creating rn based on raw traceroute data --> needs to be fixed later to include ipv4 and 6 as well as the custom data structure we use
    var parsedRsp = JSON.parse(props.response)
    //loop through each hop
    for(let i=0; i<parsedRsp.result.length; i++) {
        //for each hop, loop through hop results to parse important info and create graph nodes
        for(let j=0; j<parsedRsp.result[i].result.length; j++){
            let currNode = parsedRsp.result[i].result[j]
            let from = currNode.from
            let rtt = currNode.rtt
            let ttl = currNode.ttl
            //going to need to fix this later when there's multiple graphs on the screen
            //not sure if this will work, need to test
            nodeArr.push(<GraphNode from={from} rtt={rtt} ttl={ttl}/>)
        }
    }
    
    return (
        <div id="graph">
            {nodeArr}
        </div>
    )
}

export default Graph