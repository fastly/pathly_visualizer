package rest_api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/service"
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

	var err error
	if request.StartLiveCollection {
		err = state.EnableLiveMeasurementCollection(request.AtlasMeasurementId)
	}

	if request.LoadHistory && err == nil {
		err = state.CollectMeasurementHistory(request.AtlasMeasurementId)
	}

	if err != nil {
		if err == service.ErrMeasurementDoesNotExist {
			ctx.String(http.StatusBadRequest, "Specified measurement ID does not exist")
		} else {
			ctx.Status(http.StatusInternalServerError)
			_ = ctx.Error(err)
		}

		return
	}

	ctx.Status(http.StatusOK)
}

func (state DataRoute) StopTrackingMeasurement(ctx *gin.Context) {
	type Request struct {
		AtlasMeasurementId int  `json:"atlasMeasurementId"`
		DropStoredData     bool `json:"dropStoredData"`
	}

	request, ok := readJsonRequestBody[Request](ctx, 512)
	if !ok {
		return
	}

	if err := state.DisableLiveMeasurementCollection(request.AtlasMeasurementId); err != nil {
		if err != service.ErrMeasurementDoesNotExist {
			ctx.Status(http.StatusInternalServerError)
			_ = ctx.Error(err)
			return
		}
	}

	if request.DropStoredData {
		state.DropMeasurementData(request.AtlasMeasurementId)
	}

	ctx.Status(http.StatusOK)
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
