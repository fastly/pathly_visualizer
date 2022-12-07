package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	// Key that represents this value in the config
	key string
	// Once for loading the value
	once sync.Once
	// Default and loaded values
	defaultValue any
	loadedValue  any
}

func makeConfig(key string, defaultValue any) *Config {
	return &Config{
		key:          key,
		defaultValue: defaultValue,
		// once and loadedValue are zero-initialized
	}
}

func performLoad[T any](config *Config, parseValue func(string) (T, error)) T {
	config.once.Do(func() {
		value, ok := os.LookupEnv(config.key)

		if !ok {
			log.Println("Unable to find", config.key, "in .env. Using fallback value of", config.defaultValue)
			config.loadedValue = config.defaultValue
			return
		}

		parsed, err := parseValue(value)
		if err != nil {
			log.Printf("Found invalid value %q for %s: %s\n", value, config.key, err.Error())
			log.Println("Using fallback value of", config.defaultValue, "for", config.key)
			config.loadedValue = config.defaultValue
			return
		}

		log.Println("Using", config.key, "value of", parsed)
		config.loadedValue = parsed
	})

	return config.loadedValue.(T)
}

// True and false variable options are taken from the YAML 1.1 standard for booleans
var trueEnvOptions = [...]string{"true", "yes", "on", "y"}
var falseEnvOptions = [...]string{"false", "no", "off", "n"}

func (config *Config) GetAsFlag() bool {
	return performLoad(config, func(value string) (bool, error) {
		for _, trueOption := range trueEnvOptions {
			if value == trueOption {
				return true, nil
			}
		}

		// Check false options as well to verify if this environment variable is invalid, so we can log it
		for _, falseOption := range falseEnvOptions {
			if value == falseOption {
				return false, nil
			}
		}

		var validOptions []string
		validOptions = append(validOptions, trueEnvOptions[:]...)
		validOptions = append(validOptions, falseEnvOptions[:]...)

		return false, errors.New(fmt.Sprintf("value does not match one of %v", validOptions))
	})
}

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

func (config *Config) GetString() string {
	return performLoad(config, func(value string) (string, error) {
		return value, nil
	})
}

func (config *Config) GetDuration() time.Duration {
	return performLoad(config, func(value string) (time.Duration, error) {
		period, err := strconv.ParseUint(value, 10, 64)

		return time.Duration(period) * time.Second, err
	})
}

func (config *Config) GetInt() int {
	return performLoad(config, strconv.Atoi)
}

func (config *Config) GetFloat() float64 {
	return performLoad(config, func(value string) (float64, error) {
		return strconv.ParseFloat(value, 64)
	})
}
