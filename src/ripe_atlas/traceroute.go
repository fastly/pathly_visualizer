package ripe_atlas

import (
	"bufio"
	"encoding/json"
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"os"
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

func GetTraceRouteDataFromFile(path string) (<-chan *measurement.Result, error) {
	channel := make(chan *measurement.Result, 8)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	go func() {
		defer util.CloseAndLogErrors("Failed to close file after reading traceroute data", file)
		bufferedRead := bufio.NewReader(file)
		scanner := bufio.NewScanner(bufferedRead)

		for scanner.Scan() {
			line := scanner.Text()
			var found measurement.Result
			if err := json.Unmarshal([]byte(line), &found); err != nil {
				log.Println("Got error on line while unmarshalling JSON:", err)
			}

			channel <- &found
		}
		
		if err := scanner.Err(); err != nil {
			log.Println("Got error while reading traceroute data from file:", err)
		}
	}()

	return channel, nil
}
