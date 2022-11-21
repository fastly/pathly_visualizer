package traceroute

import (
	"fmt"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"net/netip"
	"time"
)

type TracerouteData struct {
	inner map[probeDestinationPair]*RouteData
}

func MakeTracerouteData() TracerouteData {
	return TracerouteData{
		inner: make(map[probeDestinationPair]*RouteData),
	}
}

func (tracerouteData *TracerouteData) getOrCreateRouteData(probeId int, destination netip.Addr) *RouteData {
	// Create the key using the source and destination
	key := probeDestinationPair{probeId, destination}
	return util.MapGetOrCreate(tracerouteData.inner, key, MakeRouteData)
}

func (tracerouteData *TracerouteData) GetRouteData(probe int, destination netip.Addr) (*RouteData, bool) {
	routeData, ok := tracerouteData.inner[probeDestinationPair{
		probeId:     probe,
		destination: destination,
	}]

	return routeData, ok
}

type probeDestinationPair struct {
	probeId     int
	destination netip.Addr
}

const StatisticsPeriod = 3 * 24 * time.Hour

type RouteData struct {
	probeIp    netip.Addr
	routeUsage util.MovingSummation
	Nodes      map[NodeId]*Node
	Edges      map[DirectedGraphEdge]*Edge
}

func MakeRouteData() *RouteData {
	return &RouteData{
		// probeIp: nil,
		routeUsage: util.MakeMovingSummation(StatisticsPeriod),
		Nodes:      make(map[NodeId]*Node),
		Edges:      make(map[DirectedGraphEdge]*Edge),
	}
}

func (routeData *RouteData) GetTotalUsages() int64 {
	return int64(routeData.routeUsage.Sum())
}

func (routeData *RouteData) GetProbeIp() netip.Addr {
	return routeData.probeIp
}

func (routeData *RouteData) IsEmpty() bool {
	return !routeData.probeIp.IsValid()
}

func (routeData *RouteData) getOrCreateEdge(src, dst NodeId) *Edge {
	edgeKey := DirectedGraphEdge{
		Start: src,
		Stop:  dst,
	}

	return util.MapGetOrCreate(routeData.Edges, edgeKey, MakeEdge)
}

func (routeData *RouteData) getOrCreateNode(id NodeId) *Node {
	return util.MapGetOrCreate(routeData.Nodes, id, MakeNode)
}

type Node struct {
	// It would be easier to compute the ASN when emitting to ripe atlas since we do not have access to the IpToAsn
	// service here.
	averageRtt util.MovingAverage
	lastUsed   time.Time

	// Used to determine the outboundCoverage of outbound edges
	totalOutboundUsage util.MovingSummation
	totalUsage         util.MovingSummation
}

func MakeNode() *Node {
	return &Node{
		averageRtt:         util.MakeMovingAverage(StatisticsPeriod),
		lastUsed:           time.Unix(0, 0),
		totalOutboundUsage: util.MakeMovingSummation(StatisticsPeriod),
		totalUsage:         util.MakeMovingSummation(StatisticsPeriod),
	}
}

func (node *Node) GetAverageRtt() float64 {
	return node.averageRtt.Average()
}

func (node *Node) GetLastUsed() time.Time {
	return node.lastUsed
}

func (node *Node) GetNumUsages() int64 {
	return int64(node.totalUsage.Sum())
}

func (node *Node) GetOutboundUsages() int64 {
	return int64(node.totalOutboundUsage.Sum())
}

type NodeId struct {
	Ip                 netip.Addr
	TimeoutsSinceKnown int // zero on known node
}

func WrapAddr(addr netip.Addr) NodeId {
	return NodeId{
		Ip:                 addr,
		TimeoutsSinceKnown: 0,
	}
}

func (nodeId NodeId) IsTimeout() bool {
	return nodeId.TimeoutsSinceKnown > 0
}

func (nodeId NodeId) String() string {
	return fmt.Sprintf("[%v, Timeout=%d]", nodeId.Ip, nodeId.TimeoutsSinceKnown)
}

type DirectedGraphEdge struct {
	Start, Stop NodeId
}

type Edge struct {
	// outboundCoverage = usage / srcNode.totalOutboundUsage
	// totalTrafficCoverage = usage / RouteData.routeUsage
	usage    util.MovingSummation
	netUsage util.MovingSummation
	lastUsed time.Time
}

func MakeEdge() *Edge {
	return &Edge{
		usage:    util.MakeMovingSummation(StatisticsPeriod),
		netUsage: util.MakeMovingSummation(StatisticsPeriod),
		lastUsed: time.Unix(0, 0),
	}
}

func (edge *Edge) GetLastUsed() time.Time {
	return edge.lastUsed
}

func (edge *Edge) GetUsage() int64 {
	return int64(edge.usage.Sum())
}

func (edge *Edge) GetNetUsage() float64 {
	return edge.usage.Sum()
}
