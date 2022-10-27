package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"strings"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %s\n", err.Error())
		log.Println("Configuration will be loaded from environment variables instead")
	}

	// Anything else that should be set up before main
}

func main() {
	router := gin.Default()

	//prevents CORS issues with frontend
	router.Use(func(ctx *gin.Context) {
		//allows access from wildcard origin --> need to update later to only allow from specified URL
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	})

	api := router.Group("/api")
	{
		//GET rq, basic
		api.GET("/get", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"msg": "world"})
		})

		// POST rq that returns the body of the request
		api.POST("/post", func(ctx *gin.Context) {
			buf := make([]byte, 1024)
			num, _ := ctx.Request.Body.Read(buf)
			reqBody := string(buf[0:num])
			ctx.JSON(200, gin.H{"msg": postRQService(reqBody)})
		})
	}

	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{})
	})

	router.Run(":8080")
}

// used to test changing response in separate function
func postRQ(rqBody string) bool {
	rqSplit := strings.Split(rqBody, "=")
	if rqSplit[1] == "testing" {
		return true
	} else {
		return false
	}
}
