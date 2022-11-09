package rest_api

import (
	"github.com/gin-gonic/gin"
)

// TODO: Look into replacing this middleware with https://github.com/gin-contrib/cors
// allowCORSMiddleware prevents CORS issues with frontend
func allowCORSMiddleware(ctx *gin.Context) {
	// allows access from wildcard origin --> need to update later to only allow from specified URL
	headerMap := ctx.Writer.Header()
	headerMap.Set("Access-Control-Allow-Origin", "*")
	headerMap.Set("Access-Control-Allow-Credentials", "true")
	headerMap.Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	headerMap.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

	if ctx.Request.Method == "OPTIONS" {
		ctx.AbortWithStatus(204)
		return
	}

	ctx.Next()
}
