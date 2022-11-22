package rest_api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (state DataRoute) StartTrackingMeasurement(ctx *gin.Context) {
	type Request struct {
		AtlasMeasurementId  int  `json:"atlasMeasurementId"`
		LoadHistory         bool `json:"loadHistory"`
		StartLiveCollection bool `json:"startLiveCollection"`
	}

	request, ok := readJsonRequestBody[Request](ctx, 512)
	if !ok {
		return
	}

	if !request.LoadHistory && !request.StartLiveCollection {
		ctx.String(http.StatusBadRequest, "One or more of LoadHistory or StartLiveCollection must be enabled")
		return
	}

	ctx.JSON(http.StatusOK, "")
}

func (state DataRoute) StopTrackingMeasurement(ctx *gin.Context) {
	type Request struct {
		AtlasMeasurementId int  `json:"atlasMeasurementId"`
		DropStoredData     bool `json:"dropStoredData"`
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": "world"})
}

func (state DataRoute) ListTrackedMeasurement(ctx *gin.Context) {
	type Response struct {
		AtlasMeasurementId     int   `json:"atlasMeasurementId"`
		MeasurementPeriodStart int64 `json:"measurementPeriodStart"`
		MeasurementPeriodStop  int64 `json:"measurementPeriodStop"`
		IsLoadingHistory       bool  `json:"isLoadingHistory"`
		UsesLiveCollection     bool  `json:"usesLiveCollection"`
	}
	ctx.JSON(http.StatusOK, gin.H{"msg": "world"})
}
