package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"time"
)

// ProbeCollectionRefreshPeriod We want to refresh the probes that we have every 30 minutes
const ProbeCollectionRefreshPeriod = 30 * time.Minute

type ProbeCollectionService struct {
}

func NewProbeCollectionService() *ProbeCollectionService {
	return new(ProbeCollectionService)
}

func (service *ProbeCollectionService) Name() string {
	return "ProbeCollectionService"
}

func (service *ProbeCollectionService) Init(state *ApplicationState) (err error) {
	state.ProbeCollection = probe.MakeProbeCollection()
	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {

	for {
		//Check how much time has passed since we last updated the probes
		state.probeCollectionRefreshLock.RLock()
		timeElapsed := time.Since(state.IpToAsn.LastRefresh())
		state.ipToAsnRefreshLock.RUnlock()

		//If it has been less than 30 minutes then wait until we need to get it
		if timeElapsed < ProbeCollectionRefreshPeriod {
			time.Sleep(ProbeCollectionRefreshPeriod - timeElapsed)
		} else {
			//Else Get the probes from Ripe Atlas
			state.probeCollectionRefreshLock.Lock()
			state.ProbeCollection.GetProbesFromRipeAtlas()
			state.probeCollectionRefreshLock.Unlock()
		}
	}
}
