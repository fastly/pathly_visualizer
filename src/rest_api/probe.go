package rest_api

import (
	"github.com/gin-gonic/gin"
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
		ctx.JSON(http.StatusOK, state.DestinationToProbeMap[destIP])
	}
}
