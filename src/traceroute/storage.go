package traceroute

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/DNS-OARC/ripeatlas/measurement/traceroute"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"net/netip"
	"time"
)

// This file is a stub for where traceroute API routes will be handled

// IPv4: 46320619
type TracerouteData struct {
	inner map[probeDestinationPair]*RouteData
}

func MakeTracerouteData() TracerouteData {
	return TracerouteData{
		inner: make(map[probeDestinationPair]*RouteData),
	}
}

func (tracerouteData *TracerouteData) getOrCreateRouteData(probeId int, destination netip.Addr) *RouteData {
	//Create the key using the source and destination
	key := probeDestinationPair{probeId, destination}

	//If there already is a value associated with the key, return that value
	if data, ok := tracerouteData.inner[key]; ok {
		return data
	}

	//Else, create a new Route data structure
	newData := &RouteData{
		Nodes: make(map[NodeId]*Node),
		Edges: make(map[directedGraphEdge]*Edge),
	}

	//Set the empty Route data for the key and return the data
	tracerouteData.inner[key] = newData
	return newData
}

func (tracerouteData *TracerouteData) AppendMeasurement(measurement *measurement.Result) {
	//Check if the measurement actually exists
	if measurement == nil {
		log.Println("Measurement was nil?")
		return
	}

	//Get the netip from the destination of the measurement
	destination, err := netip.ParseAddr(measurement.DstName())
	if err != nil {
		log.Println("Unable to parse measurement ( id:", measurement.MsmId(), " timestamp: ", measurement.Timestamp(), "):", err)
		return
	}

	//Get the traceroute path information from the source and destination addresses
	data := tracerouteData.getOrCreateRouteData(measurement.PrbId(), destination)
	//Add the measurement to the existing traceroute path information
	data.AppendMeasurement(measurement)
}

type probeDestinationPair struct {
	probeId     int
	destination netip.Addr
}

const StatisticsPeriod time.Duration = 3 * 24 * time.Hour

type RouteData struct {
	routeUsage util.MovingSummation
	Nodes      map[NodeId]*Node
	Edges      map[directedGraphEdge]*Edge
}

type Node struct {
	// It would be easier to compute the ASN when emitting to ripe atlas since we do not have access to the IpToAsn
	// service here.
	//asn                 int // Optional
	averageRtt          util.MovingAverage
	lastUsed            time.Time
	averagePathLifespan util.MovingAverage // in seconds
	// Used to determine the outboundCoverage of outbound edges
	totalOutboundUsage util.MovingSummation
}

type NodeId struct {
	ip                 netip.Addr
	timeoutsSinceKnown int // zero on known node
}

type Edge struct {
	// outboundCoverage = usage / srcNode.totalOutboundUsage
	// totalTrafficCoverage = usage / RouteData.routeUsage
	usage    util.MovingSummation
	lastUsed time.Time
}

func wrapAddr(addr netip.Addr) NodeId {
	return NodeId{
		ip:                 addr,
		timeoutsSinceKnown: 0,
	}
}

func (hopOrTimeout NodeId) IsTimeout() bool {
	return hopOrTimeout.timeoutsSinceKnown > 0
}

type directedGraphEdge struct {
	start, stop NodeId
}

func (routeData *RouteData) AppendMeasurement(measurement *measurement.Result) {
	// Skip measurements with errors
	if checkMeasurementForErrors(measurement) {
		return
	}

	probeIp, err := netip.ParseAddr(measurement.From())
	if err != nil {
		log.Printf("Failed to parse probe IP %q: %v\n", measurement.From(), err)
		return
	}

	// Get Traceroute replies that don't contain errors
	validReplies := filterValidReplies(measurement.TracerouteResults())

	// Add the filtered replies as Nodes
	internalFormat := toNodeId(probeIp, validReplies)

	// Apply updates to edges
	timestamp := time.Unix(int64(measurement.Timestamp()), 0)
	routeData.addHopsToGraph(internalFormat, timestamp)

	// Increment route usage
	routeData.routeUsage.IncrementUpperBound(timestamp)
	routeData.routeUsage.Append(1.0, timestamp)
}

func (routeData *RouteData) addNodesToGraph(probeAddr netip.Addr, replies [][]*traceroute.Reply, timestamp time.Time) {
	previousHop := []NodeId{wrapAddr(probeAddr)}

	for _, hop := range replies {
		var nextHop []NodeId

		for _, reply := range hop {

			if reply.X() == "*" {
				for _, prevNodeId := range previousHop {
					prevNodeId.timeoutsSinceKnown += 1
					routeData.updateGraphNode(prevNodeId, reply, timestamp)
					nextHop = append(nextHop, prevNodeId)
				}

				continue
			}

			// We know that the address must be valid because we verified it while checking reply for errors
			ip := netip.MustParseAddr(reply.From())
			nodeId := wrapAddr(ip)
			routeData.updateGraphNode(nodeId, reply, timestamp)
			nextHop = append(nextHop, nodeId)
		}

		previousHop = nextHop
	}
}

func (routeData *RouteData) updateGraphNode(id NodeId, reply *traceroute.Reply, timestamp time.Time) {
	//Get the Node related to this id
	node := routeData.getOrCreateNode(id)
	//Update the moving statistics of the node
	node.lastUsed = timestamp

	node.averageRtt.IncrementUpperBound(timestamp)
	node.averageRtt.Append(reply.Rtt(), timestamp)
}

func (routeData *RouteData) getOrCreateEdge(src, dst NodeId) *Edge {
	//Create the default edge
	edgeKey := directedGraphEdge{
		start: src,
		stop:  dst,
	}

	// Return edge if present
	if edge, ok := routeData.Edges[edgeKey]; ok {
		return edge
	}

	// fill in with default edge
	newEdge := &Edge{
		usage:    util.MakeMovingSummation(StatisticsPeriod),
		lastUsed: time.Unix(0, 0),
	}

	routeData.Edges[edgeKey] = newEdge
	return newEdge
}

func (routeData *RouteData) getOrCreateNode(id NodeId) *Node {
	// Return node if present
	if node, ok := routeData.Nodes[id]; ok {
		return node
	}

	// fill in with default edge
	newNode := &Node{
		averageRtt:          util.MakeMovingAverage(StatisticsPeriod),
		lastUsed:            time.Unix(0, 0),
		averagePathLifespan: util.MakeMovingAverage(StatisticsPeriod),
		totalOutboundUsage:  util.MakeMovingSummation(StatisticsPeriod),
	}

	routeData.Nodes[id] = newNode
	return newNode
}

func (routeData *RouteData) addHopsToGraph(res [][]NodeId, timestamp time.Time) {
	//The starting layer is the source probe or considered as Hop 0
	previousHop := res[0]

	for _, nextHop := range res[1:] {
		//Set the edges from the Cartesian product of the previous hop and the next hop
		for _, src := range previousHop {
			for _, dst := range nextHop {
				//Get the edge if it already exists or make a new one
				targetEdge := routeData.getOrCreateEdge(src, dst)
				//Update the last used attribute to this measurement's timestamp if it is newer
				if targetEdge.lastUsed.Before(timestamp) {
					targetEdge.lastUsed = timestamp
				}

				targetEdge.usage.IncrementUpperBound(timestamp)
				targetEdge.usage.Append(1.0/float64(len(nextHop)), timestamp)
			}
		}

		previousHop = nextHop
	}
}

func toNodeId(probeAddr netip.Addr, hops [][]*traceroute.Reply) (res [][]NodeId) {
	//Create the Source Node layer with the probeAddr
	previousHop := []NodeId{wrapAddr(probeAddr)}
	res = append(res, previousHop)

	//Go through each hop and create a NodeId array
	for _, hop := range hops {
		var currentHop []NodeId

		//Check each reply in a hop
		for _, reply := range hop {
			//Normal reply means we create a NodeId and add it to our list
			if reply.X() != "*" {
				replyAddr, err := netip.ParseAddr(reply.From())

				if err != nil {
					log.Printf("Got error while parsing address %q: %v\n", reply.From(), err)
					continue
				}

				currentHop = append(currentHop, wrapAddr(replyAddr))
				continue
			}

			// We hit a timeout, so now we need to copy the previous hop and apply that timeout to each node
			for _, previousAddr := range previousHop {
				previousAddr.timeoutsSinceKnown += 1
				currentHop = append(currentHop, previousAddr)
			}
		}

		//Add the current hop's results and prepare for the next hop
		res = append(res, currentHop)
		previousHop = currentHop
	}

	return res
}

func filterValidReplies(results []*traceroute.Result) (res [][]*traceroute.Reply) {
	for _, hop := range results {
		var hopReplies []*traceroute.Reply

		if hop.Error() != "" {
			// This hop was an error. What do we do with this information? Should this disqualify a measurement?
			// For now, just add an empty slice for this hop
			res = append(res, hopReplies)
			continue
		}

		//Iterate through each reply in a hop and only add the ones that don't have errors
		for _, reply := range hop.Replies() {
			if !checkReplyForErrors(reply) {
				hopReplies = append(hopReplies, reply)
			}
		}

		res = append(res, hopReplies)
	}

	return
}

func checkReplyForErrors(reply *traceroute.Reply) bool {
	// Check for ICMP errors
	if reply.Err() != "" {
		return true
	}

	// We can't completely tell for sure if a reply was late or not, but we can guess based on which of the fields is
	// zero initialized.
	if reply.Late() != 0 || reply.Rtt() == 0.0 {
		return true
	}

	// Check that reply IP is a valid address
	if _, err := netip.ParseAddr(reply.From()); reply.X() != "*" && err != nil {
		return true
	}

	return false
}

func checkMeasurementForErrors(measurement *measurement.Result) bool {
	// I checked the documentation, but I'm not sure if there are any errors recorded on the measurement level.
	//TODO Check for same IP showing up in a path at multiple points
	return false
}
