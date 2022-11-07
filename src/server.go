package main

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/common"
	"github.com/joho/godotenv"
	"log"
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
	services := []common.Service{
		// Services should be listed here in order initialization and startup
	}

	state := common.InitApplicationState()

	initServices(state, services)
	startServices(state, services)

}

func initServices(state *common.ApplicationState, services []common.Service) {
	log.Println("Performing initialization for", len(services), "services")

	for _, service := range services {
		log.Println("Initializing service", service.Name())

		if err := service.Init(state); err != nil {
			// It is safe to emit a fatal panic in this context since the server would not be able to continue if a
			// service failed to start
			log.Fatalf("Failed to initialize service %s: %s\n", service.Name(), err.Error())
		}
	}
}

func startServices(state *common.ApplicationState, services []common.Service) {
	log.Println("Starting", len(services), "services")

	for _, service := range services {
		// Service is passed to closure as arguments since GoLand warned that direct usage may produce unexpected values
		go func(service common.Service) {
			log.Println("Starting service", service.Name())

			// A service should run until the application is exited so any value returned would be treated as an error
			err := service.Run(state)
			log.Println("Service", service.Name(), "exited prematurely:", err)
		}(service)
	}
}

