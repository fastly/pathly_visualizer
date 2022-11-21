package probe

import (
	"encoding/json"
	"fmt"
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/request"
	"log"
	"net/http"
	"net/netip"
	"runtime"
	"sync"
)

type ProbeCollection struct {
	ProbeMap map[int]*Probe
}

const ProbePage string = "https://atlas.ripe.net/api/v2/probes/?format=json"

func NewProbeCollection() *ProbeCollection {
	var p ProbeCollection
	p.ProbeMap = make(map[int]*Probe)
	return &p
}

func (probeCollection *ProbeCollection) GetProbesFromRipeAtlas() {

	//Create the probe map if already didn't
	if probeCollection.ProbeMap == nil {
		probeCollection.ProbeMap = make(map[int]*Probe)
	}

	//Get the total number of pages
	responseProbe, err := http.Get(ProbePage)
	if err != nil {
		log.Fatalf("http.Get(%s): %s", ProbePage, err.Error())
	}

	var pageCountResponse probeAPIPage
	if err = json.NewDecoder(responseProbe.Body).Decode(&pageCountResponse); err != nil {
		log.Fatalf("Could not get the total number of probes: %v\n", err.Error())
	}
	defer responseProbe.Body.Close()

	totalPages := (pageCountResponse.Count + 99) / 100
	pagesPerCPU := totalPages / runtime.NumCPU()

	//Create a waitgroupint
	var wg sync.WaitGroup

	//Create channel that each routine will send a probe to
	probeChannel := make(chan *Probe, 64)

	// Read Atlas results using REST API
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())

	//Create number of go routines equal to number of CPU cores
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(currentCore int) {
			defer wg.Done()
			//The starting Page for each CPU core
			startingPage := currentCore * pagesPerCPU
			var endingPage int

			//Last page is either to the next set or to the very end
			if currentCore == runtime.NumCPU()-1 {
				endingPage = totalPages + 1
			} else {
				endingPage = (currentCore + 1) * pagesPerCPU
			}

			//Go through each page and make a
			for page := startingPage; page < endingPage; page++ {
				//Add this routine as a waitgroup

				//Connect to the specific page
				probes, err := a.Probes(ripeatlas.Params{
					"page": int64(page),
				})
				if err != nil {
					log.Fatalf("Could not get probes from Ripe Atlas %v", err.Error())
				}

				//Check for each probe on the page
				for probe := range probes {
					//Only worry about correctly parsed and connected probes
					if !isProbeValid(probe) {
						continue
					}

					//Create our own probe object
					probeObj, err := createProbe(probe)

					if err != nil {
						log.Printf("Could not parse the probe id: %v, got error: %v\n", probe.Id(), err)
						continue
					}
					//Send the obj to the channel
					probeChannel <- probeObj
				}
			}
		}(i)
	}

	//Wait until all the routines are done and close the channel
	go func() {
		wg.Wait()
		close(probeChannel)
	}()

	//Add each probe from the channel and add it to our main list
	for p := range probeChannel {
		probeCollection.ProbeMap[p.Id] = p
	}

}

func (probeCollection *ProbeCollection) GetProbesFromID(probeID int) *Probe {

	//If we already store that probe then return it
	if probe, ok := probeCollection.ProbeMap[probeID]; ok {
		return probe
	}
	// Read Atlas results using REST API
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())

	//Connect to the specific page
	probes, err := a.Probes(ripeatlas.Params{
		"pk": fmt.Sprint(probeID),
	})
	if err != nil {
		log.Fatalf("Could not get probes id: %v from Ripe Atlas: %v", probeID, err)
	}

	var probeObj *Probe
	for probe := range probes {
		//Only worry about correctly parsed and connected probes
		if !isProbeValid(probe) {
			continue
		}
		//Create our own probe object
		probeObj, err = createProbe(probe)
		if err != nil {
			log.Printf("Could not parse the probe id: %v, got error: %v\n", probe.Id(), err)
			continue
		}
		//Add it to our storage
		probeCollection.ProbeMap[probeObj.Id] = probeObj
	}
	//Return the probe object
	return probeObj
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
	var probeAddress4, probeAddress6 netip.Addr
	var err error
	//Only parse if we are getting a Ipv4 or Ipv6
	if probe.AddressV4() != "" {
		probeAddress4, err = netip.ParseAddr(probe.AddressV4())
		if err != nil {
			return nil, err
		}
	}

	if probe.AddressV6() != "" {
		probeAddress6, err = netip.ParseAddr(probe.AddressV6())
		if err != nil {
			return nil, err
		}
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
