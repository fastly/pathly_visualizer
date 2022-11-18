package rest_api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/service"
	"net/http"
)

// This function currently holds routes form the REST API frontend experiments. After being cleaned up, this function
// should not use inlined anonymous functions for routing.
func setupRoutes(router *gin.Engine, state *service.ApplicationState) {
	router.LoadHTMLFiles("index.html")
	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	api := router.Group("/api")

	traceroute := api.Group("/traceroute")
	traceroute.POST("/full", DataRoute{state}.GetTracerouteFull)

	router.NoRoute(func(ctx *gin.Context) {
		ctx.AbortWithStatus(http.StatusNotFound)
	})
}

type DataRoute struct {
	*service.ApplicationState
}
