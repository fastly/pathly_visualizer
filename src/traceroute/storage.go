package traceroute

import (
	"fmt"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"net/netip"
	"sync"
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

func (tracerouteData *TracerouteData) EvictOutdatedData() {
	var stats EvictionStats
	evictionTime := time.Now()

	for _, route := range tracerouteData.inner {
		routeStats := route.EvictToStatisticsPeriod(evictionTime)
		stats = stats.Add(routeStats)
	}
	
	log.Println("Evicted outdated data:", stats.Nodes, "nodes,", stats.RawEdges+stats.CleanEdges, "edges")
}

func (tracerouteData *TracerouteData) getOrCreateRouteData(probeId int, destination netip.Addr) *RouteData {
	// Create the key using the source and destination
	key := probeDestinationPair{probeId, destination}
	route := util.MapGetOrCreate(tracerouteData.inner, key, MakeRouteData)
	route.probeId = probeId
	return route
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

var statisticsPeriod time.Duration
var statisticsPeriodLoader sync.Once

func getStatisticsPeriod() time.Duration {
	statisticsPeriodLoader.Do(func() {
		statisticsPeriod = util.GetEnvDuration(util.StatisticsPeriod, 3*24*time.Hour)
	})

	return statisticsPeriod
}

type RouteData struct {
	probeId    int
	probeIp    netip.Addr
	routeUsage util.MovingSummation
	Nodes      map[NodeId]*Node
	Edges      map[DirectedGraphEdge]*Edge
	CleanEdges map[DirectedGraphEdge]*Edge
	Metrics    RouteUsageMetrics
}

type EvictionStats struct {
	Nodes      uint
	CleanEdges uint
	RawEdges   uint
}

func (stats EvictionStats) Add(other EvictionStats) EvictionStats {
	stats.Nodes += other.Nodes
	stats.RawEdges += other.RawEdges
	stats.CleanEdges += other.CleanEdges

	return stats
}

func (routeData *RouteData) EvictToStatisticsPeriod(timestamp time.Time) EvictionStats {
	oldestAllowed := timestamp.Add(-getStatisticsPeriod())
	routeData.Metrics.EvictMetricsUpTo(oldestAllowed)

	var stats EvictionStats

	for id, node := range routeData.Nodes {
		if node.lastUsed.Before(oldestAllowed) {
			delete(routeData.Nodes, id)
			stats.Nodes += 1
		}
	}
	for id, edge := range routeData.Edges {
		if edge.lastUsed.Before(oldestAllowed) {
			delete(routeData.Edges, id)
			stats.RawEdges += 1
		}
	}
	for id, edge := range routeData.CleanEdges {
		if edge.lastUsed.Before(oldestAllowed) {
			delete(routeData.CleanEdges, id)
			stats.CleanEdges += 1
		}
	}

	routeData.AlignStatisticsEndTime(timestamp)
	return stats
}

func (routeData *RouteData) AlignStatisticsEndTime(timestamp time.Time) {
	routeData.routeUsage.IncrementUpperBound(timestamp)

	for _, node := range routeData.Nodes {
		node.averageRtt.IncrementUpperBound(timestamp)
		node.totalOutboundUsage.IncrementUpperBound(timestamp)
		node.totalCleanOutboundUsage.IncrementUpperBound(timestamp)
		node.totalUsage.IncrementUpperBound(timestamp)
	}

	for _, edge := range routeData.Edges {
		edge.usage.IncrementUpperBound(timestamp)
		edge.netUsage.IncrementUpperBound(timestamp)
	}

	for _, edge := range routeData.CleanEdges {
		edge.usage.IncrementUpperBound(timestamp)
		edge.netUsage.IncrementUpperBound(timestamp)
	}
}

func MakeRouteData() *RouteData {
	return &RouteData{
		// probeIp: nil,
		routeUsage: util.MakeMovingSummation(getStatisticsPeriod()),
		Nodes:      make(map[NodeId]*Node),
		Edges:      make(map[DirectedGraphEdge]*Edge),
		CleanEdges: make(map[DirectedGraphEdge]*Edge),
		Metrics:    makeRouteUsageMetrics(),
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
	totalOutboundUsage      util.MovingSummation
	totalCleanOutboundUsage util.MovingSummation
	totalUsage              util.MovingSummation
}

func MakeNode() *Node {
	return &Node{
		averageRtt:              util.MakeMovingAverage(getStatisticsPeriod()),
		lastUsed:                time.Unix(0, 0),
		totalOutboundUsage:      util.MakeMovingSummation(getStatisticsPeriod()),
		totalCleanOutboundUsage: util.MakeMovingSummation(getStatisticsPeriod()),
		totalUsage:              util.MakeMovingSummation(getStatisticsPeriod()),
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
		usage:    util.MakeMovingSummation(getStatisticsPeriod()),
		netUsage: util.MakeMovingSummation(getStatisticsPeriod()),
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
