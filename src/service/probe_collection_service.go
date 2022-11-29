package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"net/netip"
	"time"
)

// ProbeCollectionRefreshPeriod We want to refresh the probes that we have every 30 minutes
const ProbeCollectionRefreshPeriod = 30 * time.Minute

type ProbeCollectionService struct {
	probeCollection          probe.ProbeCollection
	probeRegistrationChannel chan probe.ProbeRegistration
}

func NewProbeCollectionService() *ProbeCollectionService {
	return new(ProbeCollectionService)
}

func (service *ProbeCollectionService) Name() string {
	return "ProbeCollectionService"
}

func (service *ProbeCollectionService) Init(*ApplicationState) (err error) {
	service.probeCollection = probe.MakeProbeCollection()
	service.probeRegistrationChannel = make(chan probe.ProbeRegistration)
	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {

	for {
		//Check how much time has passed since we last updated the probes
		state.probeCollectionRefreshLock.RLock()
		timeElapsed := time.Since(service.probeCollection.GetLastRefresh())
		state.probeCollectionRefreshLock.RUnlock()

		//If it has been less than 30 minutes then be ready for probe registration
		if timeElapsed < ProbeCollectionRefreshPeriod {
			checkWithinElapsed(service, state, timeElapsed)
		} else {
			getFromRipeAtlas(service, state)
		}
	}
}

func checkWithinElapsed(service *ProbeCollectionService, state *ApplicationState, timeElapsed time.Duration) {
	//Wait on channel or timeout
	select {
	//Wait for traceroute measurement to give probes
	//Ok is always true unless there are no more items within the channel
	case msg, ok := <-service.probeRegistrationChannel:
		if !ok {
			return
		} else {
			addProbeRegistration(service, state, msg.DestinationIP, msg.ProbeID)
		}
		//We continue to get probes from Ripe Atlas
	case <-time.After(ProbeCollectionRefreshPeriod - timeElapsed):
		return
	}
}

func addProbeRegistration(service *ProbeCollectionService, state *ApplicationState, destination netip.Addr, probeID int) {
	//Get the corresponding probeObj
	probeObj := service.probeCollection.GetProbesFromID(probeID)

	//Get the list of probes related to that probeID
	probesFromAddress := state.DestinationToProbeMap[destination]

	//Check if we already have this probeObj
	//This could be annoying should we be sorting them or store as a map
	for _, currProbe := range probesFromAddress {
		if currProbe == probeObj {
			return
		}
	}

	//If we did not find the probeObj then this is a new one, and we append it to the current list of probes
	state.DestinationToProbeMap[destination] = append(probesFromAddress, probeObj)
}

func getFromRipeAtlas(service *ProbeCollectionService, state *ApplicationState) {
	//Get the probes from Ripe Atlas
	state.probeCollectionRefreshLock.Lock()
	service.probeCollection.GetProbesFromRipeAtlas()
	state.probeCollectionRefreshLock.Unlock()
}
