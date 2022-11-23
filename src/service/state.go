package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/asn"
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"sync"
)

// ApplicationState holds the state of the server and will be updated based on incoming new data being created and
// distributed via the rest api.
//
// NOTE: This MUST be thread safe! Since this is being shared between multiple goroutines then we need to keep thread
// safety in mind. We will most likely want to use sync.RWMutex due to how the rest api will frequently read, but the
// other services will do a mix of reading and writing. There should be one concurrency structure in here for each
// piece of the state that can be used in isolation.
type ApplicationState struct {
	IpToAsn                    asn.IpToAsn
	ipToAsnRefreshLock         sync.RWMutex
	ProbeCollection            probe.ProbeCollection
	probeCollectionRefreshLock sync.RWMutex
	// etc...
}

// InitApplicationState created the initial state to use upon the start of the application. This function is
// responsible for doing the initial setup for any data that is not managed by a service
func InitApplicationState() *ApplicationState {
	// Zero initialize for now
	return new(ApplicationState)
}

type Service interface {
	// Name provides the name of the service. This is only used for logging.
	Name() string

	// Init sets up the state of this service
	Init(state *ApplicationState) error

	// Run begins execution of the service. This is assumed to consume the entire thread and should only exit upon
	// encountering a fatal error.
	Run(state *ApplicationState) error
}
