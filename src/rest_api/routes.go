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

	// GET request, basic
	api.GET("/get", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"msg": "world"})
	})

	// POST rq that returns the body of the request
	api.POST("/post", func(ctx *gin.Context) {
		buf := make([]byte, 1024)
		num, _ := ctx.Request.Body.Read(buf)
		reqBody := string(buf[0:num])
		ctx.JSON(http.StatusOK, gin.H{"msg": reqBody})
	})

	api.GET("/probes", func(ctx *gin.Context) {
		//ctx.JSON(http.StatusOK, gin.H{"probes": bytes.NewBuffer(jsonStruct)})
	})

	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{})
	})
}
