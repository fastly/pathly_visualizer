package test_traceroute_data

import (
	tracerouteData "github.com/jmeggitt/fastly_anycast_experiments.git/traceroute-data"
	"log"
	"reflect"
	"testing"
)

func TestGetTraceRouteData(t *testing.T) {
	expectedVal := 5040
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("46320619", 1666897839, 1666904714)
	actualVal := actualTraceRoute[0].Fw()

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv4(t *testing.T) {
	expectedVal := "193.0.14.129"
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("5001", 1666915200, 1667001599)
	actualVal := actualTraceRoute[0].DstAddr()

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv6(t *testing.T) {
	expectedVal := "2001:7fd::1"
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("6001", 1667001600, 1667087999)
	actualVal := actualTraceRoute[0].DstAddr()

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv4(t *testing.T) {
	expectedVal := "199.9.14.201"
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("5010", 1667001600, 1667087999)
	actualVal := actualTraceRoute[0].DstAddr()

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv6(t *testing.T) {
	expectedVal := "2001:500:200::b"
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("6010", 1667001600, 1667087999)
	actualVal := actualTraceRoute[0].DstAddr()

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestStreamTracerouteData(t *testing.T) {
	actualTraceRoute := tracerouteData.GetStreamingTraceRouteData(6010)
	log.Printf("%+v", actualTraceRoute)

}
