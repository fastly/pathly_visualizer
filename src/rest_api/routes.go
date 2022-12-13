package rest_api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmeggitt/fastly_anycast_experiments.git/service"
)

// This function currently holds routes form the REST API frontend experiments. After being cleaned up, this function
// should not use inlined anonymous functions for routing.
func setupRoutes(router *gin.Engine, state *service.ApplicationState) {
	router.LoadHTMLFiles("index.html")
	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	api := router.Group("/api")

	api.GET("/destinations", func(cxt *gin.Context) {
		cxt.JSON(http.StatusOK, []gin.H{{"ipv4": "151.101.0.1", "ipv6": "2a04:4e42::1"}})
	})

	traceroute := api.Group("/traceroute")
	traceroute.POST("/raw", DataRoute{state}.GetTracerouteRaw)
	traceroute.POST("/clean", DataRoute{state}.GetTracerouteClean)
	traceroute.POST("/full", DataRoute{state}.GetTracerouteFull)

	api.POST("/probes", DataRoute{state}.GetProbes)

	router.NoRoute(func(ctx *gin.Context) {
		ctx.AbortWithStatus(http.StatusNotFound)
	})
}

type DataRoute struct {
	*service.ApplicationState
}
