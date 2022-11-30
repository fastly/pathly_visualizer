package probe

import (
	"testing"
)

func TestGetProbes(t *testing.T) {

	probeCollection := MakeProbeCollection()
	probeCollection.GetProbesFromRipeAtlas()

	if probeCollection.ProbeMap == nil {
		t.Errorf("Didn't create Probe Map")
	}

}

func TestGetProbesID(t *testing.T) {

	probeCollection := MakeProbeCollection()
	probeCollection.GetProbeFromID(1004942)

	if probeCollection.ProbeMap == nil {
		t.Errorf("Didn't create Probe Map")
	}

}
