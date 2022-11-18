package probe

import (
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/request"
	"log"
	"net/netip"
	"sync"
)

type ProbeCollection struct {
	ProbeMap map[int]*Probe
}

const ProbePages int = 455

func NewProbeCollection() *ProbeCollection {
	var p ProbeCollection
	p.ProbeMap = make(map[int]*Probe)
	return &p
}

func (probeCollection *ProbeCollection) GetProbesFromRipeAtlas() {

	probeCollection.ProbeMap = make(map[int]*Probe)

	//Create a waitgroupint
	var wg sync.WaitGroup

	//Create channel
	probeChannel := make(chan *Probe)

	// Read Atlas results using REST API
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())

	//Get all the probes on each page

	for page := 1; page < ProbePages; page++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()
			//Connect to the specific page
			probes, err := a.Probes(ripeatlas.Params{
				"page": int64(pageNum),
			})
			if err != nil {
				log.Fatalf(err.Error())
			}

			for probe := range probes {

				//Only worry about correctly parsed and connected probes
				if !isProbeValid(probe) {
					continue
				}

				probeObj, err := createProbe(probe)

				if err != nil {
					log.Printf("Could not parse the probe id: %v, got error: %v\n", probe.Id(), err)
					continue
				}

				probeChannel <- probeObj
			}
		}(page)
	}
	go func() {
		wg.Wait()
		close(probeChannel)
	}()

	for p := range probeChannel {
		probeCollection.ProbeMap[p.Id] = p
	}

}

func isProbeValid(probe *request.Probe) bool {
	//If we can't parse the probes break
	if probe.ParseError != nil {
		log.Printf("Error Parsing Probe: %v\n", probe.ParseError)
		return false
	}

	//Only worry about connected probes
	if probe.Status().Id() != 1 {
		return false
	}
	return true
}

func createProbe(probe *request.Probe) (*Probe, error) {

	//Get the Addresses
	probeAddress4, err := netip.ParseAddr(probe.AddressV4())
	if err != nil {
		return nil, err
	}

	probeAddress6, err := netip.ParseAddr(probe.AddressV6())
	if err != nil {
		return nil, err
	}

	//Create our own probe
	var probeObj = Probe{
		Id:          probe.Id(),
		Ipv4:        probeAddress4,
		Ipv6:        probeAddress6,
		CountryCode: probe.CountryCode(),
		Asn4:        uint32(probe.AsnV4()),
		Asn6:        uint32(probe.AsnV6()),
		Type:        probe.Geometry().Type(),
		Coordinates: probe.Geometry().Coordinates(),
	}

	return &probeObj, nil
}
