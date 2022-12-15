package ripe_atlas

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const MeasurementsUrl = "https://atlas.ripe.net/api/v2/measurements"

const pkParam = "pk"
const startParam = "start"
const stopParam = "stop"

const typeParam = "type"
const msmParam = "msm"

func GetStaticTraceRouteData(measurementID int, startTime, endTime time.Time) ([]measurement.Result, error) {
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())
	channel, err := a.MeasurementResults(ripeatlas.Params{
		pkParam:    strconv.Itoa(measurementID),
		startParam: startTime.Unix(),
		stopParam:  endTime.Unix(),
	})

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
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	byteChannel, outputChannel := util.MakeWorkGroup(64, func(bytes []byte, output chan *measurement.Result) {
		var out *measurement.Result
		if err := json.Unmarshal(bytes, &out); err != nil {
			log.Println("Received error while reading input JSON:", err)
		} else {
			output <- out
		}
	})

	go breakReaderIntoLines(file, byteChannel)

	return outputChannel, nil
}

// GetLatestTraceRouteData is similar to the ripeatlas equivalent, but this version uses a thread pool to speed up
// parsing messages
func GetLatestTraceRouteData(measurementID int) (<-chan *measurement.Result, io.Closer, error) {
	startTime := time.Now().Add(-config.StatisticsPeriod.GetDuration())

	url := fmt.Sprintf("%s/%d/results?format=txt&start=%d", MeasurementsUrl, measurementID, startTime.Unix())
	res, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}

	byteChannel, outputChannel := util.MakeWorkGroup(64, func(bytes []byte, output chan *measurement.Result) {
		var out *measurement.Result
		if err := json.Unmarshal(bytes, &out); err != nil {
			log.Println("Received error while reading input JSON:", err)
		} else {
			output <- out
		}
	})

	go breakReaderIntoLines(res.Body, byteChannel)

	return outputChannel, res.Body, nil
}

func breakReaderIntoLines(input io.ReadCloser, lineBytesOutput chan []byte) {
	defer util.CloseAndLogErrors("Failed to close input after reading traceroute data", input)
	defer close(lineBytesOutput)
	bufferedRead := bufio.NewReader(input)
	scanner := bufio.NewScanner(bufferedRead)

	// Break input into lines and distribute them to workers
	for scanner.Scan() {
		line := scanner.Bytes()
		buffer := make([]byte, len(line), len(line))
		copy(buffer, line)
		lineBytesOutput <- buffer
	}

	if err := scanner.Err(); err != nil {
		log.Println("Got error while reading traceroute data from input:", err)
	}
}

func updateCacheFile(measurementID int, cacheFile string) error {
	file, err := os.Create(cacheFile)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	defer util.CloseAndLogErrors("Failed to close cache file writer", file)

	startTime := time.Now().Add(-config.StatisticsPeriod.GetDuration())

	url := fmt.Sprintf("%s/%d/results?format=txt&start=%d", MeasurementsUrl, measurementID, startTime.Unix())
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	defer util.CloseAndLogErrors("Failed to close request for measurement results", res.Body)
	if _, err := io.Copy(writer, res.Body); err != nil {
		return err
	}

	return writer.Flush()
}

func CachedGetTraceRouteData(measurementID int) (channel <-chan *measurement.Result, err error) {
	var cachePath string
	if cachePath, err = util.GetCacheDir(); err != nil {
		return
	}

	cacheFile := filepath.Join(cachePath, fmt.Sprintf("%d.ndjson", measurementID))
	var stat os.FileInfo
	stat, err = os.Stat(cacheFile)

	if err != nil && !os.IsNotExist(err) {
		return
	}

	cacheDuration := config.CacheStoreDuration.GetDuration()

	if err != nil || stat.ModTime().Add(cacheDuration).Before(time.Now()) {
		log.Println("Refreshing cache entry for measurement", measurementID)

		if err = updateCacheFile(measurementID, cacheFile); err != nil {
			return
		}
	}

	return GetTraceRouteDataFromFile(cacheFile)
}
