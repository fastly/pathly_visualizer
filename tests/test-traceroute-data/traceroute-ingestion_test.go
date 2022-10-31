package test_traceroute_data

import (
	"fmt"
	tracerouteData "github.com/jmeggitt/fastly_anycast_experiments.git/traceroute-data"
	"github.com/joho/godotenv"
	"log"
	"reflect"
	"testing"
)

func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Printf("Error loading .env file: %s\n", err.Error())
		log.Println("Configuration will be loaded from environment variables instead")
	}

	// Anything else that should be set up before main
}

func TestGetTraceRouteData(t *testing.T) {
	expectedJSON := tracerouteData.Traceroute{}
	actualJSON := tracerouteData.GetTraceRouteData("1666897839", "1666904714", "46320619")

	fmt.Printf("%+v\n", (*actualJSON)[0])

	if reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Errorf("Got %v want %v", actualJSON, expectedJSON)
	}
}
