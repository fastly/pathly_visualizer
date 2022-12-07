package rest_api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"net/http"
	"net/netip"
)

type probeRequest struct {
	// You only need this field
	DestinationIp string `json:"destinationIp"`
	FilterAsns    []int  `json:"filterAsns"`
	FilterPrefix  string `json:"filterPrefix"`
	// Any other information for search
}

func (state DataRoute) GetProbes(ctx *gin.Context) {
	if request, ok := readJsonRequestBody[probeRequest](ctx); !ok {
		return
	} else {
		state.ProbeDataLock.Lock()
		defer state.ProbeDataLock.Unlock()
		destIP, err := netip.ParseAddr(request.DestinationIp)
		if destIP, err = netip.ParseAddr(request.DestinationIp); err != nil {
			ctx.String(http.StatusBadRequest, "Could not read destination IP")
			return
		}
		//Get the list of probes
		probesFromAddress := state.DestinationToProbeMap[destIP]
		var finalProbeList []*probe.Probe

		for _, probeDestination := range probesFromAddress {
			finalProbeList = append(finalProbeList, probeDestination.Probe)
		}

		ctx.JSON(http.StatusOK, finalProbeList)
	}
}
