package traceroute

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/DNS-OARC/ripeatlas/measurement/traceroute"
	"log"
	"net/netip"
	"sort"
	"time"
)

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

func (routeData *RouteData) AppendMeasurement(measurement *measurement.Result) {
	// Skip measurements with errors
	if checkMeasurementForErrors(measurement) {
		return
	}

	probeIp, err := netip.ParseAddr(measurement.SrcAddr())
	if err != nil {
		log.Printf("Failed to parse probe IP %q: %v\n", measurement.SrcAddr(), err)
		return
	}

	if !routeData.probeIp.IsValid() {
		routeData.probeIp = probeIp
	}

	// Get Traceroute replies that don't contain errors
	validReplies := filterValidReplies(measurement.TracerouteResults())

	// Add the filtered replies as Nodes
	internalFormat := toNodeId(probeIp, validReplies)

	// Apply updates to edges
	timestamp := time.Unix(int64(measurement.Timestamp()), 0)
	routeData.addNodesToGraph(probeIp, validReplies, timestamp)
	routeData.addHopsToGraph(internalFormat, timestamp)

	probeNode := routeData.getOrCreateNode(NodeId{
		Ip:                 probeIp,
		TimeoutsSinceKnown: 0,
	})

	probeNode.averageRtt.Append(0.0, timestamp)
	probeNode.totalUsage.Append(1.0, timestamp)
	probeNode.lastUsed = timestamp

	// Increment route usage
	routeData.routeUsage.Append(1.0, timestamp)
}

func uniqueNodeIdsForLayer(replies []*traceroute.Reply, prevLayerCount int) int {
	layerNodeCount := 0
	foundTimeout := false

	for _, reply := range replies {
		if reply.X() == "*" {
			if !foundTimeout {
				layerNodeCount += prevLayerCount
				foundTimeout = true
			}
		} else {
			layerNodeCount += 1
		}
	}

	return layerNodeCount
}

func dedupNodeIds(list []NodeId) []NodeId {
	if len(list) == 0 {
		return list
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Ip == list[j].Ip {
			return list[i].TimeoutsSinceKnown < list[j].TimeoutsSinceKnown
		}

		return list[i].Ip.Less(list[j].Ip)
	})

	i := 1
	for j := 1; j < len(list); j++ {
		if list[i-1] != list[j] {
			list[i] = list[j]
			i += 1
		}
	}

	return list[:i]
}

func (routeData *RouteData) addNodesToGraph(probeAddr netip.Addr, replies [][]*traceroute.Reply, timestamp time.Time) {
	previousHop := []NodeId{WrapAddr(probeAddr)}
	visitedNodes := map[NodeId]struct{}{}

	for _, hop := range replies {
		var nextHop []NodeId
		expectedLayerNodes := uniqueNodeIdsForLayer(hop, len(previousHop))
		handledTimeout := false

		for _, reply := range hop {
			if reply.X() != "" {
				for _, prevNodeId := range previousHop {
					prevNodeId.TimeoutsSinceKnown += 1
					routeData.updateGraphNode(prevNodeId, reply, timestamp, visitedNodes)

					if !handledTimeout {
						nextHop = append(nextHop, prevNodeId)
					}
				}

				handledTimeout = true
				continue
			}

			// We know that the address must be valid because we verified it while checking reply for errors
			ip := netip.MustParseAddr(reply.From())
			nodeId := WrapAddr(ip)
			routeData.updateGraphNode(nodeId, reply, timestamp, visitedNodes)
			nextHop = append(nextHop, nodeId)
		}

		if expectedLayerNodes != len(nextHop) {
			log.Println("Violated expectation for number of connected nodes; Found", len(nextHop), "Expected", expectedLayerNodes)
		}
		previousHop = dedupNodeIds(nextHop)
	}
}

func (routeData *RouteData) updateGraphNode(id NodeId, reply *traceroute.Reply, timestamp time.Time, visitedNodes map[NodeId]struct{}) {
	//Get the Node related to this id
	node := routeData.getOrCreateNode(id)
	//Update the moving statistics of the node
	node.lastUsed = timestamp

	node.averageRtt.Append(reply.Rtt(), timestamp)

	if _, ok := visitedNodes[id]; !ok {
		node.totalUsage.Append(1.0, timestamp)
		visitedNodes[id] = struct{}{}
	}
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

				routeData.getOrCreateNode(src).totalOutboundUsage.Append(1.0, timestamp)
				targetEdge.usage.Append(1.0, timestamp)
				targetEdge.netUsage.Append(1.0/float64(len(nextHop)), timestamp)
			}
		}

		previousHop = nextHop
	}
}

func toNodeId(probeAddr netip.Addr, hops [][]*traceroute.Reply) (res [][]NodeId) {
	//Create the Source Node layer with the probeAddr
	previousHop := []NodeId{WrapAddr(probeAddr)}
	res = append(res, previousHop)

	//Go through each hop and create a NodeId array
	for _, hop := range hops {
		var currentHop []NodeId

		addedTimeouts := false
		//Check each reply in a hop
		for _, reply := range hop {
			//Normal reply means we create a NodeId and add it to our list
			if reply.X() != "*" {
				replyAddr, err := netip.ParseAddr(reply.From())

				if err != nil {
					log.Printf("Got error while parsing address %q: %v\n", reply.From(), err)
					continue
				}

				currentHop = append(currentHop, WrapAddr(replyAddr))
				continue
			}

			if addedTimeouts {
				continue
			}
			addedTimeouts = true

			// We hit a timeout, so now we need to copy the previous hop and apply that timeout to each node
			for _, previousAddr := range previousHop {
				previousAddr.TimeoutsSinceKnown += 1
				currentHop = append(currentHop, previousAddr)
			}
		}

		currentHop = dedupNodeIds(currentHop)
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

	// Allow timeouts
	if reply.X() == "*" {
		return false
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
	// Check if it was unable to resolve the source or destination addresses
	if measurement.SrcAddr() == "" || measurement.DstAddr() == "" {
		return true
	}

	//TODO Check for same IP showing up in a path at multiple points
	return false
}
