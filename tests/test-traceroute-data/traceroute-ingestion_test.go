package test_traceroute_data

import (
	tracerouteData "github.com/jmeggitt/fastly_anycast_experiments.git/traceroute-data"
	"reflect"
	"testing"
)

func TestGetTraceRouteData(t *testing.T) {
	expectedVal := 5040
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("46320619", "1666897839", "1666904714")
	actualVal := actualTraceRoute[0].Fw

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataWithNoTime(t *testing.T) {
	expectedVal := 5040
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("46320619", "", "")
	actualVal := actualTraceRoute[0].Fw

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv4(t *testing.T) {
	expectedVal := "193.0.14.129"
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("5001", "1666915200", "1667001599")
	actualVal := actualTraceRoute[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataKRootIPv6(t *testing.T) {
	expectedVal := "2001:7fd::1"
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("6001", "1667001600", "1667087999")
	actualVal := actualTraceRoute[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv4(t *testing.T) {
	expectedVal := "199.9.14.201"
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("5010", "1667001600", "1667087999")
	actualVal := actualTraceRoute[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}

func TestGetTraceRouteDataBRootIPv6(t *testing.T) {
	expectedVal := "2001:500:200::b"
	actualTraceRoute, _ := tracerouteData.GetTraceRouteData("6010", "1667001600", "1667087999")
	actualVal := actualTraceRoute[0].Dst_addr

	if !reflect.DeepEqual(expectedVal, actualVal) {
		t.Errorf("Got %+v want %+v", actualVal, expectedVal)
	}
}
