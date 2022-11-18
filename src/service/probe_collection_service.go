package service

type ProbeCollectionService struct {
}

func NewProbeCollectionService() *ProbeCollectionService {
	return new(ProbeCollectionService)
}

func (service *ProbeCollectionService) Name() string {
	return "ProbeCollectionService"
}

func (service *ProbeCollectionService) Init(state *ApplicationState) (err error) {

}

func (service *ProbeCollectionService) Run(state *ApplicationState) error {

}
