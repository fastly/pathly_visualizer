package probe

import (
	"encoding/json"
	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/request"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"net/http"
	"net/netip"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type ProbeCollection struct {
	ProbeMap    map[int]*Probe
	LastRefresh time.Time
}

const ProbePage string = "https://atlas.ripe.net/api/v2/probes/?format=json"
const MaxProbeChannelLimit = 64

func MakeProbeCollection() ProbeCollection {
	return ProbeCollection{
		ProbeMap:    make(map[int]*Probe),
		LastRefresh: time.Unix(0, 0),
	}
}

// Struct used for messages between traceroute measurement and probe collection
type ProbeRegistration struct {
	ProbeID       int
	DestinationIP netip.Addr
}

func (probeCollection *ProbeCollection) GetProbesFromRipeAtlas() {

	//Get the total number of pages
	responseProbe, err := http.Get(ProbePage)
	if err != nil {
		log.Printf("Could not connect to probe page http.Get(%s): %s\n", ProbePage, err.Error())
	}
	var pageCountResponse struct {
		Count int
	}

	if err = json.NewDecoder(responseProbe.Body).Decode(&pageCountResponse); err != nil {
		log.Printf("Could not get the total number of probes: %v\n", err.Error())
		return
	}
	defer util.CloseAndLogErrors("Probes from Ripe Atlas", responseProbe.Body)
	//Total number of CPU for distributing work
	numberOfCPUs := runtime.NumCPU()
	totalPages := (pageCountResponse.Count + 99) / 100
	pagesPerCPU := totalPages / numberOfCPUs

	//Create a wait group
	var wg sync.WaitGroup
	var once sync.Once

	//Create channel that each routine will send a probe to
	probeChannel := make(chan Probe, MaxProbeChannelLimit)

	// Read Atlas results using REST API
	//This struct is safe to share across threads and use concurrently
	atlas := ripeatlas.Atlaser(ripeatlas.NewHttp())

	//Add the number of workers to the waitgroup
	wg.Add(numberOfCPUs)
	//Create number of go routines equal to number of CPU cores
	for i := 0; i < numberOfCPUs; i++ {
		//Each CPU core will handle multiple pages
		go func(currentCore int) {
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
				probes, err := atlas.Probes(ripeatlas.Params{
					"page": int64(page),
				})
				if err != nil {
					log.Println("Could not get probes from Ripe Atlas:,", err.Error())
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

			//We are done with this worker
			wg.Done()

			//Wait until all the routines are done and close the channel
			//This will be done by one of the workers
			once.Do(func() {
				wg.Wait()
				close(probeChannel)
				probeCollection.LastRefresh = time.Now()
			})
		}(i)
	}

	//Add each probe from the channel and add it to our main list
	for p := range probeChannel {
		//If we already store that probe then replace the contents
		if probe, ok := probeCollection.ProbeMap[p.Id]; ok {
			*probe = p
		} else {
			//If it is a new probe then add the new pointer
			probeCollection.ProbeMap[p.Id] = &p
		}
	}

}

// Check if we already store the probe.
// If So return it
// If we do not have it in storage then grab the probe from Ripe Atlas
func (probeCollection *ProbeCollection) GetProbeFromID(probeID int) *Probe {

	//If we already store that probe then return it
	if probe, ok := probeCollection.ProbeMap[probeID]; ok {
		return probe
	}
	// Read Atlas results using REST API
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())

	//Connect to the specific page
	probes, err := a.Probes(ripeatlas.Params{
		"pk": strconv.Itoa(probeID),
	})
	if err != nil {
		log.Printf("Could not get probes id: %v from Ripe Atlas: %v\n", probeID, err)
	}

	var probeObj Probe
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
		probeCollection.ProbeMap[probeObj.Id] = &probeObj
		return &probeObj
	}
	//Return the probe object, returns nil if no probe is found
	return &probeObj
}

func (probeCollection *ProbeCollection) GetLastRefresh() time.Time {
	return probeCollection.LastRefresh
}

func isProbeValid(probe *request.Probe) bool {
	//If we can't parse the probes break
	if probe.ParseError != nil && probe.Id() != 0 {
		log.Printf("Error Parsing Probe: %v\n", probe.ParseError)
		return false
	}

	//Only worry about connected probes
	if probe.Status().Id() != 1 {
		return false
	}
	return true
}

func createProbe(probe *request.Probe) (Probe, error) {

	//Get the Addresses
	var probeAddress4, probeAddress6 netip.Addr
	var err error
	//Only parse if we are getting a Ipv4 or Ipv6
	if probe.AddressV4() != "" {
		if probeAddress4, err = netip.ParseAddr(probe.AddressV4()); err != nil {
			return Probe{}, err
		}
	}

	if probe.AddressV6() != "" {
		if probeAddress6, err = netip.ParseAddr(probe.AddressV6()); err != nil {
			return Probe{}, err
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

	return probeObj, nil
}
