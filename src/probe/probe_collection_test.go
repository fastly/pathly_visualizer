package probe

import (
	"testing"
)

func TestGetProbeID(t *testing.T) {

	probeCollection := NewProbeCollection()
	probeCollection.GetProbeById(54318)
	probeMap := probeCollection.ProbeMap

	expectedLength := 1

	if expectedLength != len(probeMap) {
		t.Errorf("Incorrect number of probes in map expected %v but got %v", expectedLength, len(probeMap))
	}

}
