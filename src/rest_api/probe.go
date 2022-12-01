package rest_api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
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
	request, ok := readJsonRequestBody[probeRequest](ctx, 512)
	if !ok {
		return
	}

	state.ProbeDataLock.Lock()
	defer state.ProbeDataLock.Unlock()
	destIP, err := netip.ParseAddr(request.DestinationIp)
	if destIP, err = netip.ParseAddr(request.DestinationIp); err != nil {
		ctx.String(http.StatusBadRequest, "Could not read destination IP")
		return
	}
	ctx.JSON(http.StatusOK, state.DestinationToProbeMap[destIP])

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
