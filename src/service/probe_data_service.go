package service

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"net/netip"
)

// ProbeInfo is missing the country code and location since this stub does not have the functionality
type ProbeInfo struct {
	Ipv4 netip.Addr `json:"ipv4"`
	Ipv6 netip.Addr `json:"ipv6"`
	Asn4 uint32     `json:"asn4"`
	Asn6 uint32     `json:"asn6"`
}

func (state *ApplicationState) BootstrapCollectProbeInfo(result *measurement.Result) {
	state.ProbeDataLock.Lock()
	defer state.ProbeDataLock.Unlock()

	if state.ProbeData == nil {
		state.ProbeData = make(map[int]*ProbeInfo)
	}

	// Do a sanity check to make sure we didn't get bad data.
	if result.PrbId() == 0 {
		return
	}

	var probeInfo *ProbeInfo
	if info, ok := state.ProbeData[result.PrbId()]; ok {
		probeInfo = info
	} else {
		probeInfo = &ProbeInfo{}
		state.ProbeData[result.PrbId()] = probeInfo
	}

	var srcAddr netip.Addr
	if addr, err := netip.ParseAddr(result.SrcAddr()); err == nil {
		srcAddr = addr
	} else if addr, err := netip.ParseAddr(result.From()); err == nil {
		srcAddr = addr
	} else {
		return
	}

	if srcAddr.Is4() {
		probeInfo.Ipv4 = srcAddr

		if asn, ok := state.GetIpToAsn(srcAddr); ok {
			probeInfo.Asn4 = asn
		}
	} else {
		probeInfo.Ipv6 = srcAddr

		if asn, ok := state.GetIpToAsn(srcAddr); ok {
			probeInfo.Asn6 = asn
		}
	}
}
