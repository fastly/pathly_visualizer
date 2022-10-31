package traceroute_data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const RipeAtlasApi string = "https://atlas.ripe.net"
const GetMeasurementsRoute string = "/api/v2/measurements/"

func GetTraceRouteData(startTime, endTime, measurmentID string) *[]Traceroute {

	var formatKey = ""
	if startTime == "" || endTime == "" {
		formatKey = "/results/?format=json&key="
	} else {
		formatKey = "/results/?start=" + startTime + "&stop=" + endTime + "&format=json&key="
	}
	// Get the data from url
	//url format: https://atlas.ripe.net/api/v2/measurements/<Measurement ID>/results/?format=json&key=<Your RIPE Atlas API Key>
	url := RipeAtlasApi + GetMeasurementsRoute + measurmentID + formatKey + os.Getenv("API_KEY")
	resp, err := http.Get(url)
	if err != nil {
		//TODO Figure out how to handle errors
		fmt.Println("Could not Get request")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			//TODO Figure out how to handle errors
			fmt.Println("Could not close file")
		}
	}(resp.Body)

	//Read in the JSON data
	var traceroute []Traceroute

	err = json.NewDecoder(resp.Body).Decode(&traceroute)
	if err != nil {
		//TODO Figure out how to handle errors
		fmt.Printf("cannot decode JSON: %v\n", err)
	}

	return &traceroute

}

func readJson(body []byte) (traceroute *Traceroute) {
	//Read the json into our Traceroute data structure
	traceroute = &Traceroute{}
	err := json.Unmarshal(body, &traceroute)
	if err != nil {
		//TODO Figure out how to handle errors
		fmt.Println("Could not unmarshal the json")
	}
	return traceroute
}
