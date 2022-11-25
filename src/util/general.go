package util

import (
	"errors"
	"log"
	"os"
	"strings"
)

// True and false variable options are taken from the YAML 1.1 standard for booleans
var trueEnvOptions = []string{"true", "yes", "on", "y"}
var falseEnvOptions = []string{"false", "no", "off", "n"}

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

func GetCacheDir() (string, error) {
	cachePath, ok := os.LookupEnv("CACHE_DIR")
	if !ok {
		cachePath = ".cache"
	}

	stat, err := os.Stat(cachePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(cachePath, os.ModePerm)
	} else if err == nil && !stat.IsDir() {
		err = errors.New("CACHE_DIR must point to a directory")
	}

	return cachePath, err
}

func MapGetOrCreate[K comparable, V any](data map[K]V, key K, init func() V) V {
	if value, ok := data[key]; ok {
		return value
	}

	value := init()
	data[key] = value
	return value
}
