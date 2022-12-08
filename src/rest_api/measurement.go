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

	request, ok := readJsonRequestBody[Request](ctx)
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
		ctx.String(http.StatusInternalServerError, "Failed due to Error: %w", err)
		return
	}

	ctx.Status(http.StatusOK)
}

func (state DataRoute) StopTrackingMeasurement(ctx *gin.Context) {
	type Request struct {
		AtlasMeasurementId int  `json:"atlasMeasurementId"`
		DropStoredData     bool `json:"dropStoredData"`
	}

	request, ok := readJsonRequestBody[Request](ctx)
	if !ok {
		return
	}

	if err := state.DisableLiveMeasurementCollection(request.AtlasMeasurementId); err != nil {
		ctx.String(http.StatusInternalServerError, "Failed due to Error: %w", err)
		return
	}

	if request.DropStoredData {
		if err := state.DropMeasurementData(request.AtlasMeasurementId); err != nil {
			ctx.String(http.StatusInternalServerError, "Failed due to Error: %w", err)
			return
		}
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

	var measurements []Response

	state.StoredMeasurements.TrackedMeasurements.Range(func(key any, value any) bool {
		data := value.(*service.MeasurementCollectionInfo)
		data.Lock.Lock()

		next := Response{
			AtlasMeasurementId:     data.Id,
			MeasurementPeriodStart: data.OldestData.Unix(),
			MeasurementPeriodStop:  data.LatestData.Unix(),
			IsLoadingHistory:       data.CollectingHistory,
			UsesLiveCollection:     data.PerformingLiveCollection,
		}

		data.Lock.Unlock()
		measurements = append(measurements, next)
		return true
	})

	ctx.JSON(http.StatusOK, measurements)
}
