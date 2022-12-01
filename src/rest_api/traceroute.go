package rest_api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/ripe_atlas"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"io"
	"net/http"
	"net/netip"
	"time"
)

type tracerouteRequest struct {
	ProbeId       int
	DestinationIp netip.Addr
}

func (state DataRoute) GetTracerouteRaw(ctx *gin.Context) {
	request, ok := readJsonRequestBody[tracerouteRequest](ctx, 512)
	if !ok {
		return
	}

	state.TracerouteDataLock.Lock()
	routeData, ok := state.TracerouteData.GetRouteData(request.ProbeId, request.DestinationIp)
	state.TracerouteDataLock.Unlock()
	if !ok {
		ctx.String(http.StatusBadRequest, "unable to find combination of probe and IP: %+v\n", request)
		return
	}

	var dataUrls []string
	for measurement, timeRange := range routeData.Metrics.MeasurementRanges {
		url := fmt.Sprintf("%s/%d/results?format=txt&start=%d&stop=%d",
			ripe_atlas.MeasurementsUrl, measurement, timeRange.Start.Unix(), timeRange.End.Unix())

		dataUrls = append(dataUrls, url)
	}

	// If possible redirect to RIPE Atlas
	if len(dataUrls) == 1 {
		ctx.Redirect(http.StatusFound, dataUrls[0])
		return
	}

	fileName := fmt.Sprintf("raw_traceroute_%d.json", request.ProbeId)

	header := ctx.Writer.Header()
	header["Content-Type"] = []string{"application/octet-stream"}
	header["Content-Disposition"] = []string{"attachment; filename=" + fileName}

	for _, url := range dataUrls {
		response, err := http.Get(url)
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if _, err := io.Copy(ctx.Writer, response.Body); err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := response.Body.Close(); err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	ctx.Writer.Flush()
}

func (state DataRoute) GetTracerouteClean(ctx *gin.Context) {
	request, ok := readJsonRequestBody[tracerouteRequest](ctx, 512)
	if !ok {
		return
	}

	state.TracerouteDataLock.Lock()
	defer state.TracerouteDataLock.Unlock()
	routeData, ok := state.TracerouteData.GetRouteData(request.ProbeId, request.DestinationIp)
	if !ok {
		ctx.String(http.StatusBadRequest, "unable to find combination of probe and IP: %+v\n", request)
		return
	}

	if routeData.IsEmpty() {
		ctx.String(http.StatusServiceUnavailable, "no error-free data to provide: %+v\n", request)
		return
	}

	// Align statistics so the edge statistics make sense
	routeData.AlignStatisticsEndTime(time.Now())

	type NodeData struct {
		Id                  string  `json:"id"`
		Asn                 uint32  `json:"asn,omitempty"`
		AverageRtt          float64 `json:"averageRtt"`
		LastUsed            int64   `json:"lastUsed"`
		AveragePathLifespan float64 `json:"averagePathLifespan"`
	}

	var nodes []NodeData

	for id, storedNode := range routeData.Nodes {
		if id.IsTimeout() {
			continue
		}

		asn := uint32(0)
		if foundAsn, ok := state.GetIpToAsn(id.Ip); ok {
			asn = foundAsn
		}

		nodes = append(nodes, NodeData{
			Id:         id.Ip.String(),
			Asn:        asn,
			AverageRtt: storedNode.GetAverageRtt(),
			LastUsed:   storedNode.GetLastUsed().Unix(),
			// TODO: Replace with occurrences in output?
			AveragePathLifespan: 0,
		})
	}

	type EdgeData struct {
		Start                string  `json:"start"`
		End                  string  `json:"end"`
		OutboundCoverage     float64 `json:"outboundCoverage"`
		TotalTrafficCoverage float64 `json:"totalTrafficCoverage"`
		LastUsed             int64   `json:"lastUsed"`
	}
	var edges []EdgeData

	// Count how many outbound edges each node has
	parentCounts := make(map[netip.Addr]uint)
	for endpoints := range routeData.CleanEdges {
		if _, ok := parentCounts[endpoints.Start.Ip]; !ok {
			parentCounts[endpoints.Start.Ip] = 0
		}

		parentCounts[endpoints.Start.Ip] = parentCounts[endpoints.Start.Ip] + 1
	}

	minEdgeWeight := util.GetEnvFloat(util.MinCleanEdgeWeight, 0.1)

	for endpoints, edge := range routeData.CleanEdges {
		outboundCoverage := float64(edge.GetUsage()) / float64(routeData.Nodes[endpoints.Start].GetCleanOutboundUsages())

		minCoverage := minEdgeWeight / float64(parentCounts[endpoints.Start.Ip])
		if outboundCoverage < minCoverage {
			continue
		}

		edges = append(edges, EdgeData{
			Start:                endpoints.Start.Ip.String(),
			End:                  endpoints.Stop.Ip.String(),
			OutboundCoverage:     outboundCoverage,
			TotalTrafficCoverage: edge.GetNetUsage() / float64(routeData.GetTotalUsages()),
			LastUsed:             edge.GetLastUsed().Unix(),
		})
	}

	var probeIps []string
	for _, ip := range routeData.GetProbeIps() {
		probeIps = append(probeIps, ip.String())
	}

	ctx.JSON(http.StatusOK, gin.H{
		"probeIps": probeIps,
		"nodes":    nodes,
		"edges":    edges,
	})
}

func (state DataRoute) GetTracerouteFull(ctx *gin.Context) {
	request, ok := readJsonRequestBody[tracerouteRequest](ctx, 512)
	if !ok {
		return
	}

	state.TracerouteDataLock.Lock()
	defer state.TracerouteDataLock.Unlock()
	routeData, ok := state.TracerouteData.GetRouteData(request.ProbeId, request.DestinationIp)
	if !ok {
		ctx.String(http.StatusBadRequest, "unable to find combination of probe and IP: %+v\n", request)
		return
	}

	if routeData.IsEmpty() {
		ctx.String(http.StatusServiceUnavailable, "no error-free data to provide: %+v\n", request)
		return
	}

	// Align statistics so the edge statistics make sense
	routeData.AlignStatisticsEndTime(time.Now())

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

	var probeIds []NodeId
	for _, ip := range routeData.GetProbeIps() {
		probeIds = append(probeIds, NodeId{
			Ip:             ip.String(),
			TimeSinceKnown: 0,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"probeIds": probeIds,
		"nodes":    nodes,
		"edges":    edges,
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
		if err == util.ErrMessageTooLong {
			ctx.String(http.StatusBadRequest, "Request too long\n")
		} else {
			ctx.Status(http.StatusInternalServerError)
			_ = ctx.Error(err)
		}
		return
	}

	if err := json.Unmarshal(requestBytes, &value); err != nil {
		ctx.String(http.StatusBadRequest, "Request is not valid JSON: %s\n", err.Error())
		return
	}

	ok = true
	return
}
