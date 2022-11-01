package traceroute_data

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const RipeAtlasApi string = "https://atlas.ripe.net"
const GetMeasurementsRoute string = "/api/v2/measurements/"

func GetTraceRouteData(measurementID, startTime, endTime string) ([]Traceroute, error) {

	var formatKey = ""
	if startTime == "" || endTime == "" {
		formatKey = "/results/?format=json&key="
	} else {
		formatKey = "/results/?start=" + startTime + "&stop=" + endTime + "&format=json"
	}
	// Get the data from url
	// format: https://atlas.ripe.net/api/v2/measurements/<Measurement ID>/results/?format=json
	url := RipeAtlasApi + GetMeasurementsRoute + measurementID + formatKey

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Could not open GET request for traceroute")
		return nil, err
	}

	defer closeAndLogErrors("Error while closing HTTP response for Traceroute ingestion", resp.Body)

	//Read in the JSON data
	var traceroute []Traceroute
	if err = json.NewDecoder(resp.Body).Decode(&traceroute); err != nil {
		log.Printf("cannot decode JSON: %v\n", err)
	}
	return traceroute, nil
}

func closeAndLogErrors(source string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Println(source, err)
	}
}
