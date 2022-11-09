package main

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/rest_api"
	"github.com/jmeggitt/fastly_anycast_experiments.git/service"
	"github.com/joho/godotenv"
	"log"
	"sync"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Err loading .env file: %s\n", err.Error())
		log.Println("Configuration will be loaded from environment variables instead")
	}

	log.Println("Finished environment initialization")
}

func main() {
	// Services should be listed here in order initialization and startup
	services := []service.Service{
		service.IpToAsnService{},
		rest_api.NewRestApiService(),
		// etc...
	}

	state := service.InitApplicationState()

	initServices(state, services)
	waitGroup := startServices(state, services)

	// Wait for all services to exit before closing program
	waitGroup.Wait()
}

func initServices(state *service.ApplicationState, services []service.Service) {
	log.Println("Performing initialization for", len(services), "services")

	for _, serviceToInit := range services {
		log.Println("Initializing service", serviceToInit.Name())

		if err := serviceToInit.Init(state); err != nil {
			// It is safe to emit a fatal panic in this context since the server would not be able to continue if a
			// service failed to start
			log.Fatalf("Failed to initialize service %s: %s\n", serviceToInit.Name(), err.Error())
		}
	}
}

func startServices(state *service.ApplicationState, services []service.Service) *sync.WaitGroup {
	log.Println("Starting", len(services), "services")
	waitGroup := new(sync.WaitGroup)
	waitGroup.Add(len(services))

	for _, serviceToRun := range services {
		// Service is passed to closure as arguments since GoLand warned that direct usage may produce unexpected values
		go func(service service.Service) {
			log.Println("Starting service", service.Name())

			// A service should run until the application is exited so any value returned would be treated as an error
			err := service.Run(state)
			log.Println("Service", service.Name(), "exited prematurely:", err)
			waitGroup.Done()
		}(serviceToRun)
	}

	return waitGroup
}
