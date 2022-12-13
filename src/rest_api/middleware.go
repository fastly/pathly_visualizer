package rest_api

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
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

func handleErrors(ctx *gin.Context) {
	if len(ctx.Errors) == 0 {
		return
	}

	log.Println("Got", len(ctx.Errors), "errors when handling route", ctx.Request.Method, ctx.Request.URL)

	for index, err := range ctx.Errors {
		if err.Type == gin.ErrorTypePrivate {
			log.Printf("\t%d (Server): %v", index, err.Err)
		} else {
			log.Printf("\t%d (Request): %v", index, err.Err)
		}
	}

	lastErr := ctx.Errors.Last()
	if lastErr.IsType(gin.ErrorTypePublic) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": lastErr.Error()})
	}
}
