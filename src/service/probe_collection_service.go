package service

import (
	"errors"
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"time"
)

// ProbeCollectionRefreshPeriod We want to refresh the probes that we have every 30 minutes
const ProbeCollectionRefreshPeriod = 30 * time.Minute

type ProbeCollectionService struct {
	allProbes probe.ProbeCollection
}

func NewProbeCollectionService() *ProbeCollectionService {
	return new(ProbeCollectionService)
}

func (service *ProbeCollectionService) Name() string {
	return "ProbeCollectionService"
}

func (service *ProbeCollectionService) Init(state *ApplicationState) (err error) {
	service.allProbes = probe.MakeProbeCollection()
	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {

	for {
		//Check how much time has passed since we last updated the probes
		state.probeCollectionRefreshLock.RLock()
		timeElapsed := time.Since(service.allProbes.GetLastRefresh())
		state.probeCollectionRefreshLock.RUnlock()

		//If it has been less than 30 minutes then be ready for probe registration
		if timeElapsed < ProbeCollectionRefreshPeriod {
			if err := checkWithinElapsed(service, state, timeElapsed); err != nil {
				return err
			}
		} else {
			getFromRipeAtlas(service, state)
		}
	}
}

func checkWithinElapsed(service *ProbeCollectionService, state *ApplicationState, timeElapsed time.Duration) error {
	//Wait on channel or timeout
	select {
	//Wait for traceroute measurement to give probes
	case msg, ok := <-state.probeRegistrationChannel:
		if !ok {
			return errors.New("could not receive probe from traceroute measurement")
		} else {
			service.allProbes.AddProbeDestination(msg.DestinationIP, msg.ProbeID)
		}

		//We continue to get probes from Ripe Atlas
	case <-time.After(ProbeCollectionRefreshPeriod - timeElapsed):
		return nil
	}
	return nil
}

func getFromRipeAtlas(service *ProbeCollectionService, state *ApplicationState) {
	//Else Get the probes from Ripe Atlas
	state.probeCollectionRefreshLock.Lock()
	service.allProbes.GetProbesFromRipeAtlas()
	state.probeCollectionRefreshLock.Unlock()
}
