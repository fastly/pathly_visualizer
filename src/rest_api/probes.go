package rest_api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// geoJsonStub is just the example value for https://geojson.org/. It gets included so that the API stub can work with
// frontend even if there are some minor issues in some areas.
var geoJsonStub = gin.H{
	"type": "Feature",
	"geometry": gin.H{
		"type":        "Point",
		"coordinates": []float64{125.6, 10.1},
	},
	"properties": gin.H{
		"name": "Dinagat Islands",
	},
}

func (state DataRoute) GetProbesList(ctx *gin.Context) {
	type ProbeData struct {
		Id          int    `json:"id"`
		Ipv4        any    `json:"ipv4"`
		Ipv6        any    `json:"ipv6"`
		CountryCode string `json:"countryCode"`
		Asn4        any    `json:"asn4,omitempty"`
		Asn6        any    `json:"asn6,omitempty"`
		Location    any    `json:"location"`
	}

	var probes []ProbeData

	state.ProbeDataLock.RLock()
	defer state.ProbeDataLock.RUnlock()

	for id, value := range state.ProbeData {
		probeData := ProbeData{
			Id:          id,
			Ipv4:        nil,
			Ipv6:        nil,
			CountryCode: "AQ", // Antarctica
			Asn4:        nil,
			Asn6:        nil,
			Location:    geoJsonStub,
		}

		if value.Ipv4.IsValid() {
			probeData.Ipv4 = value.Ipv4.String()
		}

		if value.Ipv6.IsValid() {
			probeData.Ipv6 = value.Ipv6.String()
		}

		if value.Asn4 != 0 {
			probeData.Asn4 = value.Asn4
		}

		if value.Asn6 != 0 {
			probeData.Asn6 = value.Asn6
		}

		probes = append(probes, probeData)
	}

	ctx.JSON(http.StatusOK, probes)
}
