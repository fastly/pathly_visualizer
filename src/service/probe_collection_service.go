package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"net/netip"
	"time"
)

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

func (service *ProbeCollectionService) Init(state *ApplicationState) (err error) {
	service.probeCollection = probe.MakeProbeCollection()
	service.probeRegistrationChannel = make(chan probe.ProbeRegistration)
	state.DestinationToProbeMap = make(map[netip.Addr][]*probe.ProbeUsage)
	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {
	refreshPeriod := config.ProbeCollectionRefreshPeriod.GetDuration()

	for {
		//Check how much time has passed since we last updated the probes
		state.ProbeDataLock.RLock()
		timeElapsed := time.Since(service.probeCollection.GetLastRefresh())
		state.ProbeDataLock.RUnlock()

		//If it has been less than Refresh Period then be ready for probe registration
		if timeElapsed < refreshPeriod {
			timeLeft := refreshPeriod - timeElapsed
			checkWithinElapsed(service, state, timeLeft)
		} else {
			getFromRipeAtlas(service, state)
		}
	}
}

func checkWithinElapsed(service *ProbeCollectionService, state *ApplicationState, timeLeft time.Duration) {
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
	case <-time.After(timeLeft):
		return
	}
}

func addProbeRegistration(service *ProbeCollectionService, state *ApplicationState, destination netip.Addr, probeID int) {

	//Get the list of probes related to that probeID
	probesFromAddress := state.DestinationToProbeMap[destination]

	//Check if we already have this probe id
	for _, currProbe := range probesFromAddress {
		if currProbe.Probe.Id == probeID {
			currProbe.LastUsed = time.Now()
			return
		}
	}

	//We have not found the probe in the destination to probe map
	//Get the corresponding probeObj
	probeObj := service.probeCollection.GetProbeFromID(probeID)

	//Return immediately if we could not find probeObj within our storage, already logged the missing probe
	if probeObj == nil {
		return
	}

	//Create the object for the destination to probe map
	newProbeDestination := probe.ProbeUsage{
		Probe:    probeObj,
		LastUsed: time.Now(),
	}

	//If we did not find the probeObj then this is a new one, and we append it to the current list of probes
	state.DestinationToProbeMap[destination] = append(probesFromAddress, &newProbeDestination)
}

func getFromRipeAtlas(service *ProbeCollectionService, state *ApplicationState) {
	//Get the probes from Ripe Atlas
	state.ProbeDataLock.Lock()
	service.probeCollection.GetProbesFromRipeAtlas()
	state.ProbeDataLock.Unlock()
}
