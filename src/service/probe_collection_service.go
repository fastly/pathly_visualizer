package service

import "github.com/jmeggitt/fastly_anycast_experiments.git/probe"

type ProbeCollectionService struct {
}

func NewProbeCollectionService() *ProbeCollectionService {
	return new(ProbeCollectionService)
}

func (service *ProbeCollectionService) Name() string {
	return "ProbeCollectionService"
}

func (service *ProbeCollectionService) Init(state *ApplicationState) (err error) {
	state.probeCollection = *probe.NewProbeCollection()

	return nil
}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {
	state.probeCollection.GetProbesFromRipeAtlas()
	return nil
}
