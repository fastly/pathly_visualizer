package common

// ApplicationState holds the state of the server and will be updated based on incoming new data being created and
// distributed via the rest api.
//
// NOTE: This MUST be thread safe! Since this is being shared between multiple goroutines then we need to keep thread
// safety in mind. We will most likely want to use sync.RWMutex due to how the rest api will frequently read, but the
// other services will do a mix of reading and writing. There should be one concurrency structures in here for each
// piece of the state that can be used in isolation.
type ApplicationState struct {
}



