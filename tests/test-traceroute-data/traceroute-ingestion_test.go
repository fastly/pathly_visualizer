package test_traceroute_data

import (
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	tracerouteData "github.com/jmeggitt/fastly_anycast_experiments.git/traceroute-data"
	"log"
	"reflect"
	"testing"
)

func TestGetTraceRouteData(t *testing.T) {
	expectedLength := 34
	actualTraceRoute, err := tracerouteData.GetStaticTraceRouteData("46320619", 1666897839, 1666897839)

	if err != nil {
		t.Errorf("Failed to collect static Traceroute data: %+v\n", err)
	}
	actualLength := len(actualTraceRoute)

	if !reflect.DeepEqual(expectedLength, actualLength) {
		t.Errorf("Got %+v want %+v\n", actualLength, expectedLength)
	}

	expectedFirmwareCounts := make(map[int]int)
	expectedFirmwareCounts[4790] = 1
	expectedFirmwareCounts[5020] = 4
	expectedFirmwareCounts[5040] = 10
	expectedFirmwareCounts[5070] = 18
	expectedFirmwareCounts[5080] = 1

	actualFirmwareCounts := make(map[int]int)

	for _, traceroute := range actualTraceRoute {
		actualFirmwareCounts[traceroute.Fw()] = actualFirmwareCounts[traceroute.Fw()] + 1
	}

	for fw, count := range expectedFirmwareCounts {
		if actualFirmwareCounts[fw] != count {
			t.Errorf("Did not get correct count for firmware %v: Expected %v but got %v\n", fw, count, actualFirmwareCounts[fw])
		}
	}
}

func TestGetTraceRouteDataKRootIPv4(t *testing.T) {
	expectedLength := 7
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("5001", 1666915200, 1666915200)
	actualLength := len(actualTraceRoute)

	if !reflect.DeepEqual(expectedLength, actualLength) {
		t.Errorf("Got %+v want %+v", actualLength, expectedLength)
	}

	expectedSrcAddr := []string{"10.10.66.104", "172.16.55.202", "96.45.167.27", "10.92.0.4", "192.168.29.83", "10.254.3.33", "192.168.29.40"}

	for i, traceroute := range actualTraceRoute {
		if traceroute.SrcAddr() != expectedSrcAddr[i] {
			t.Errorf("Incorrect Src Addr for KRootIPv4. Expected %v, but got %v\n", expectedSrcAddr[i], traceroute.SrcAddr())
		}
	}
}

func TestGetTraceRouteDataKRootIPv6(t *testing.T) {
	expectedLength := 2
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("6001", 1667001600, 1667001600)
	actualLength := len(actualTraceRoute)

	if !reflect.DeepEqual(expectedLength, actualLength) {
		t.Errorf("Got %+v want %+v", actualLength, expectedLength)
	}

	expectedSrcAddr := []string{"2a01:cb19:182:b100:1:b2ff:fe02:4bb5", "2600:1700:7aa1:9080:1:19ff:fead:fd17"}

	for i, traceroute := range actualTraceRoute {
		if traceroute.SrcAddr() != expectedSrcAddr[i] {
			t.Errorf("Incorrect Src Addr for KRootIPv6. Expected %v, but got %v\n", expectedSrcAddr[i], traceroute.SrcAddr())
		}
	}
}

func TestGetTraceRouteDataBRootIPv4(t *testing.T) {
	expectedVal := 3
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("5010", 1667001600, 1667001600)
	actualLength := len(actualTraceRoute)

	if !reflect.DeepEqual(expectedVal, actualLength) {
		t.Errorf("Got %+v want %+v", actualLength, expectedVal)
	}

	expectedSrcAddr := []string{"10.68.31.14", "176.98.68.54", "192.168.180.116"}

	for i, traceroute := range actualTraceRoute {
		if traceroute.SrcAddr() != expectedSrcAddr[i] {
			t.Errorf("Incorrect Src Addr for BRootIPv4. Expected %v, but got %v\n", expectedSrcAddr[i], traceroute.SrcAddr())
		}
	}
}

func TestGetTraceRouteDataBRootIPv6(t *testing.T) {
	expectedLength := 4
	actualTraceRoute, _ := tracerouteData.GetStaticTraceRouteData("6010", 1667001600, 1667001600)
	actualLength := len(actualTraceRoute)

	if !reflect.DeepEqual(expectedLength, actualLength) {
		t.Errorf("Got %+v want %+v", actualLength, expectedLength)
	}

	expectedSrcAddr := []string{"2a01:4f8:1c17:6262::1", "2a10:3781:2393:1:220:4aff:fec8:2099", "2a04:6480:204:0:1:44ff:fe1d:53b2", "2a02:a465:8ead:1:1:dff:fe08:5119"}

	for i, traceroute := range actualTraceRoute {
		if traceroute.SrcAddr() != expectedSrcAddr[i] {
			t.Errorf("Incorrect Src Addr for BRootIPv6. Expected %v, but got %v\n", expectedSrcAddr[i], traceroute.SrcAddr())
		}
	}
}

func TestStreamTracerouteData(t *testing.T) {
	actualMeasurementResult1, err1 := tracerouteData.GetStreamingTraceRouteData(6010)
	actualMeasurementResult2, err2 := tracerouteData.GetStreamingTraceRouteData(5010)

	if err1 != nil {
		t.Errorf("Received an error from streaming data: %v\n", err1)
	}
	if err2 != nil {
		t.Errorf("Received an error from streaming data: %v\n", err2)
	}

	for i := 0; i < 10; i++ {
		select {
		case msg1 := <-actualMeasurementResult1:
			if msg1.ParseError != nil {
				t.Errorf("Could not parse: %v\n", msg1.ParseError)
			}
			if msg1.Type() != "traceroute" {
				t.Errorf("Streaming 6010 not traceroute measurement but got %v\n", msg1.Type())
			}

		case msg2 := <-actualMeasurementResult2:
			if msg2.ParseError != nil {
				t.Errorf("Could not parse: %v", msg2.ParseError)
			}
			if msg2.Type() != "traceroute" {
				t.Errorf("Streaming 6010 not traceroute measurement but got %v\n", msg2.Type())
			}
		}
	}
}

func TestFileTracerouteData(t *testing.T) {

	actualTraceroute, err := getTracerouteMeasurement("basic_traceroute_testing.json")
	if err != nil {
		t.Errorf("Could not read from basic_traceroute_testing.json. Error: %+v", err)
	}
	expectedLength := 34
	actualLength := len(actualTraceroute)

	if !reflect.DeepEqual(expectedLength, actualLength) {
		t.Errorf("Got %+v want %+v\n", actualLength, expectedLength)
	}

	expectedFirmwareCounts := make(map[int]int)
	expectedFirmwareCounts[4790] = 1
	expectedFirmwareCounts[5020] = 4
	expectedFirmwareCounts[5040] = 10
	expectedFirmwareCounts[5070] = 18
	expectedFirmwareCounts[5080] = 1

	actualFirmwareCounts := make(map[int]int)

	for _, traceroute := range actualTraceroute {
		actualFirmwareCounts[traceroute.Fw()] = actualFirmwareCounts[traceroute.Fw()] + 1
	}

	for fw, count := range expectedFirmwareCounts {
		if actualFirmwareCounts[fw] != count {
			t.Errorf("Did not get correct count for firmware %v: Expected %v but got %v\n", fw, count, actualFirmwareCounts[fw])
		}
	}
}

func TestIPv6FileTracerouteData(t *testing.T) {
	actualTraceroute, err := getTracerouteMeasurement("small_BRootIPv6Traceroute.json")
	if err != nil {
		t.Errorf("Could not read from small_BRootIPv6Traceroute.json. Error: %+v", err)
	}

	expectedSrcAddr := []string{"2a01:4f8:1c17:6262::1", "2a10:3781:2393:1:220:4aff:fec8:2099", "2a04:6480:204:0:1:44ff:fe1d:53b2", "2a02:a465:8ead:1:1:dff:fe08:5119"}
	expectedDstAddr := "2001:500:200::b"
	expectedHopCounts := []int{18, 15, 12, 13}

	//Check for Src Address
	for i, traceroute := range actualTraceroute {
		if traceroute.SrcAddr() != expectedSrcAddr[i] {
			t.Errorf("Incorrect Src Addr for BRootIPv6. Expected %v, but got %v\n", expectedSrcAddr[i], traceroute.SrcAddr())
		}
	}

	//Check for Dst Addr
	for _, traceroute := range actualTraceroute {
		if traceroute.DstAddr() != expectedDstAddr {
			t.Errorf("Incorrect Dst Addr for BRootIPv6. Expected %v, but got %v\n", expectedDstAddr, traceroute.SrcAddr())
		}
	}

	//Check Hop counts
	for i, traceroute := range actualTraceroute {
		if len(traceroute.TracerouteResults()) != expectedHopCounts[i] {
			t.Errorf("Incorrect Number of hops for BRootIPv6. Expected %v, but got %v\n", len(traceroute.TracerouteResults()), expectedHopCounts[i])
		}
	}

}

func getTracerouteMeasurement(fileName string) ([]measurement.Result, error) {
	// Read Atlas results from a file
	a := ripeatlas.Atlaser(ripeatlas.NewFile())
	channel, err := a.MeasurementResults(ripeatlas.Params{"file": fileName})
	if err != nil {
		log.Printf("Could not read from %v. Error: %+v", fileName, err)
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
