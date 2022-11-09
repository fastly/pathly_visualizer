package traceroute_data

import (
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"log"
)

const pkParam = "pk"
const startParam = "start"
const stopParam = "stop"

const typeParam = "type"
const msmParam = "msm"

func GetStaticTraceRouteData(measurementID string, startTime, endTime int64) ([]measurement.Result, error) {
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())
	channel, err := a.MeasurementResults(ripeatlas.Params{pkParam: measurementID, startParam: startTime, stopParam: endTime})
	if err != nil {
		log.Printf("Cannot get measurment results from Ripe Atlas Streaming API: %v\n", err)
		return nil, err
	}
	var traceroutes []measurement.Result
	for measurementTraceroute := range channel {
		if measurementTraceroute.ParseError != nil {
			log.Printf("Measurement could not be parsed: %v\n", measurementTraceroute.ParseError)
		} else {
			traceroutes = append(traceroutes, *measurementTraceroute)
		}
	}

	return traceroutes, nil
}

func GetStreamingTraceRouteData(measurementID int) (<-chan *measurement.Result, error) {
	// Read Atlas results using Streaming API
	a := ripeatlas.Atlaser(ripeatlas.NewStream())
	channel, err := a.MeasurementResults(ripeatlas.Params{typeParam: "traceroute", msmParam: measurementID})
	if err != nil {
		log.Printf("Cannot get measurment results from Ripe Atlas Streaming API: %v\n", err)
		return nil, err
	}

	return channel, nil
}
