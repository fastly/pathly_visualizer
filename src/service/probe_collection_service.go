package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"net/netip"
	"time"
)

// ProbeCollectionRefreshPeriod We want to refresh the probes that we have every 30 minutes
const DefaultProbeCollectionRefreshPeriod = 30 * time.Minute

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
	state.DestinationToProbeMap = make(map[netip.Addr][]*probe.Probe)
	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {

	refreshPeriod := util.GetEnvDuration(util.ProbeCollectionRefreshPeriod, DefaultProbeCollectionRefreshPeriod)

	for {
		//Check how much time has passed since we last updated the probes
		state.ProbeDataLock.RLock()
		timeElapsed := time.Since(service.probeCollection.GetLastRefresh())
		state.ProbeDataLock.RUnlock()

		//If it has been less than 30 minutes then be ready for probe registration
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
		if currProbe.Id == probeID {
			return
		}
	}

	//Get the corresponding probeObj
	probeObj := service.probeCollection.GetProbeFromID(probeID)

	//Return immediately if we could not find probeObj, already logged
	if probeObj == nil {
		return
	}

	//If we did not find the probeObj then this is a new one, and we append it to the current list of probes
	state.DestinationToProbeMap[destination] = append(probesFromAddress, probeObj)
}

func getFromRipeAtlas(service *ProbeCollectionService, state *ApplicationState) {
	//Get the probes from Ripe Atlas
	state.ProbeDataLock.Lock()
	service.probeCollection.GetProbesFromRipeAtlas()
	state.ProbeDataLock.Unlock()
}
