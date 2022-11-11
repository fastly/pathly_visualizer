package test_traceroute_data

import (
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"log"
	"reflect"
	"testing"
)

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
