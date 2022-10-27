package test_traceroute_data

import (
	tracerouteData "github.com/jmeggitt/fastly_anycast_experiments.git/traceroute-data"
	"reflect"
	"testing"
)

func TestGetTraceRouteData(t *testing.T) {
	expectedJSON := tracerouteData.Traceroute{}
	actualJSON := tracerouteData.GetTraceRouteData("5001")

	if reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Errorf("Got %v want %v", actualJSON, expectedJSON)
	}
}
