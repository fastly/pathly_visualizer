package util

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// configKey is just a placeholder type to prevent improper usage of keys

const (
	// StatisticsPeriod refers to the duration statistics are stored/collected for on measurements
	StatisticsPeriod = "STATISTICS_PERIOD"

	// LogTracerouteProgress enables logging the progress of traceroute ingestion
	LogTracerouteProgress = "LOG_TRACEROUTE_PROGRESS"

	// CacheDirectory and CacheStoreDuration control where and how long cached files traceroute data are stored
	CacheDirectory     = "CACHE_DIR"
	CacheStoreDuration = "CACHE_DURATION"
)

// True and false variable options are taken from the YAML 1.1 standard for booleans
var trueEnvOptions = [...]string{"true", "yes", "on", "y"}
var falseEnvOptions = [...]string{"false", "no", "off", "n"}

func IsEnvFlagSet(key string) bool {
	value, ok := os.LookupEnv(key)
	cleaned := strings.ToLower(strings.TrimSpace(value))

	if !ok {
		return false
	}

	for _, trueOption := range trueEnvOptions {
		if cleaned == trueOption {
			return true
		}
	}

	// Check false options as well to verify if this environment variable is invalid, so we can log it
	for _, falseOption := range falseEnvOptions {
		if cleaned == falseOption {
			return false
		}
	}

	log.Printf("Unrecognized value for environment variable %q (expected true/false): %q\n", key, value)
	return false
}

func GetEnvDuration(key string, fallBack time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)

	if !ok {
		log.Println("Unable to find", key, "in .env. Using fallback value of", fallBack)
		return fallBack
	}

	period, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		log.Printf("Expected unsigned int value for %s, but found %q. Using fallback value of %v\n", key, value, fallBack)
		return fallBack
	}

	log.Println("Using", key, "of", period, "seconds")
	return time.Duration(period) * time.Second
}
