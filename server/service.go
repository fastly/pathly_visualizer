package main

import "strings"

// Testing using service functions from other files
// works when running directory based configuration in goland
// (in short, need both files running to be functional, can't just run server.go)
func postRQService(rqBody string) bool {
	rqSplit := strings.Split(rqBody, "=")
	if rqSplit[1] == "testing" {
		return true
	} else {
		return false
	}
}
