
function GraphNode(props) {
    return(
        <div className="gNode">
            <p>{props.from}</p>
            <p>{props.rtt}</p>
            <p>{props.ttl}</p>
        </div>
    )
}

export default GraphNode