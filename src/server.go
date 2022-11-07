package main

import (
	"github.com/joho/godotenv"
	"log"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Err loading .env file: %s\n", err.Error())
		log.Println("Configuration will be loaded from environment variables instead")
	}

	// Anything else that should be set up before main
}

func main() {
	log.Println("hello world")

}
