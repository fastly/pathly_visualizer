package rest_api

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"net/http"
	"net/netip"
)

type tracerouteRequest struct {
	ProbeId       int
	DestinationIp netip.Addr
}

var ErrProbeDestinationPairNotFound = errors.New("unable to find this combination of probe and IP")
var ErrNoData = errors.New("no error-free data to provide")

func (state DataRoute) GetTracerouteFull(ctx *gin.Context) {
	request, ok := readJsonRequestBody[tracerouteRequest](ctx, 512)
	if !ok {
		return
	}

	state.TracerouteDataLock.Lock()
	defer state.TracerouteDataLock.Unlock()
	routeData, ok := state.TracerouteData.GetRouteData(request.ProbeId, request.DestinationIp)
	if !ok {
		_ = ctx.Error(ErrProbeDestinationPairNotFound)
		return
	}

	if routeData.IsEmpty() {
		_ = ctx.Error(ErrNoData)
		return
	}

	type NodeId struct {
		Ip             string `json:"ip"`
		TimeSinceKnown int    `json:"timeSinceKnown"`
	}

	type NodeData struct {
		Id                  NodeId  `json:"id"`
		Asn                 uint32  `json:"asn,omitempty"`
		AverageRtt          float64 `json:"averageRtt"`
		LastUsed            int64   `json:"lastUsed"`
		AveragePathLifespan float64 `json:"averagePathLifespan"`
	}

	var nodes []NodeData

	for id, storedNode := range routeData.Nodes {
		asn := uint32(0)
		if !id.IsTimeout() {
			if foundAsn, ok := state.GetIpToAsn(id.Ip); ok {
				asn = foundAsn
			}
		}

		nodes = append(nodes, NodeData{
			Id: NodeId{
				Ip:             id.Ip.String(),
				TimeSinceKnown: id.TimeoutsSinceKnown,
			},
			Asn:        asn,
			AverageRtt: storedNode.GetAverageRtt(),
			LastUsed:   storedNode.GetLastUsed().Unix(),
			// TODO: Replace with occurrences in output?
			AveragePathLifespan: 0,
		})
	}

	type EdgeData struct {
		Start                NodeId  `json:"start"`
		End                  NodeId  `json:"end"`
		OutboundCoverage     float64 `json:"outboundCoverage"`
		TotalTrafficCoverage float64 `json:"totalTrafficCoverage"`
		LastUsed             int64   `json:"lastUsed"`
	}

	var edges []EdgeData

	for endpoints, edge := range routeData.Edges {
		edges = append(edges, EdgeData{
			Start: NodeId{
				Ip:             endpoints.Start.Ip.String(),
				TimeSinceKnown: endpoints.Start.TimeoutsSinceKnown,
			},
			End: NodeId{
				Ip:             endpoints.Stop.Ip.String(),
				TimeSinceKnown: endpoints.Stop.TimeoutsSinceKnown,
			},
			OutboundCoverage:     float64(edge.GetUsage()) / float64(routeData.Nodes[endpoints.Start].GetOutboundUsages()),
			TotalTrafficCoverage: edge.GetNetUsage() / float64(routeData.GetTotalUsages()),
			LastUsed:             edge.GetLastUsed().Unix(),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"probeIp": routeData.GetProbeIp().String(),
		"nodes":   nodes,
		"edges":   edges,
	})
}

func (request *tracerouteRequest) UnmarshalJSON(bytes []byte) (err error) {
	var buffer struct {
		ProbeId       int    `json:"probeId"`
		DestinationIp string `json:"destinationIp"`
	}

	if err = json.Unmarshal(bytes, &buffer); err != nil {
		return
	}

	request.ProbeId = buffer.ProbeId
	request.DestinationIp, err = netip.ParseAddr(buffer.DestinationIp)
	return
}

func readJsonRequestBody[T any](ctx *gin.Context, limit int) (value T, ok bool) {
	requestBytes, err := util.ReadAtMost(ctx.Request.Body, limit)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	if err := json.Unmarshal(requestBytes, &value); err != nil {
		_ = ctx.Error(err)
		return
	}

	ok = true
	return
}
