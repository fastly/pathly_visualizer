package test_traceroute_data

import (
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
	expectedVal := 5040
	actualVal := (*tracerouteData.GetTraceRouteData("46320619", "1666897839", "1666904714"))[0].Fw

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataWithNoTime(t *testing.T) {
	expectedVal := 5040
	actualVal := (*tracerouteData.GetTraceRouteData("46320619", "", ""))[0].Fw

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv4(t *testing.T) {
	expectedVal := "193.0.14.129"
	actualVal := (*tracerouteData.GetTraceRouteData("5001", "1666915200", "1667001599"))[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv6(t *testing.T) {
	expectedVal := "2001:7fd::1"
	actualVal := (*tracerouteData.GetTraceRouteData("6001", "1667001600", "1667087999"))[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv4(t *testing.T) {
	expectedVal := "199.9.14.201"
	actualVal := (*tracerouteData.GetTraceRouteData("5010", "1667001600", "1667087999"))[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv6(t *testing.T) {
	expectedVal := "2001:500:200::b"
	actualVal := (*tracerouteData.GetTraceRouteData("6010", "1667001600", "1667087999"))[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}
