package service

type CleanupService struct{}

func (CleanupService) Name() string {
	return "CleanupService"
}

func (CleanupService) Init(state *ApplicationState) (err error) {
	return
}

func (service CleanupService) Run(state *ApplicationState) (err error) {
	return
}
