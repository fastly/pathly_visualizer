package rest_api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/service"
	"log"
	"os"
)

type RestApiService struct {
	router *gin.Engine
}

func NewRestApiService() *RestApiService {
	return new(RestApiService)
}

func (service *RestApiService) Name() string {
	return "RestApiService"
}

func (service *RestApiService) Init(state *service.ApplicationState) (err error) {
	service.router = gin.New()

	// Add logging and recovery to gin
	service.router.Use(gin.Logger())
	service.router.Use(gin.Recovery())

	if err := service.router.SetTrustedProxies(nil); err != nil {
		return err
	}

	// Setup middleware
	log.Println("Setting up REST API middleware")
	service.router.Use(allowCORSMiddleware)

	log.Println("Setting up REST API routes")
	setupRoutes(service.router, state)

	service.router.Use(handleErrors)
	return
}

func (service *RestApiService) Run(state *service.ApplicationState) error {
	if os.Getenv(gin.EnvGinMode) == gin.ReleaseMode {
		return service.router.Run(":80")
	} else {
		return service.router.Run(":8080")
	}
}
