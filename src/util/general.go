package util

import (
	"errors"
	"io"
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

type ndjsonConverter struct {
	consumedHeader bool
	consumingComma bool
	inStr          bool
	escaped        bool
	bracketSurplus int
	writer         io.Writer
}

func NewNdjsonConverter(writer io.Writer) io.Writer {
	return &ndjsonConverter{
		consumedHeader: false,
		consumingComma: false,
		inStr:          false,
		escaped:        false,
		bracketSurplus: 0,
		writer:         writer,
	}
}

func (converter *ndjsonConverter) consumeComma(p []byte) int {
	for index, b := range p {
		if b == ',' {
			converter.consumingComma = false
			return index + 1
		}
	}

	return len(p)
}

func (converter *ndjsonConverter) countToObjectEnd(p []byte) (int, bool) {
	// This should be fine for UTF-8 too since we only perform special operations on ascii bytes
	for index, b := range p {
		if converter.inStr {
			if b == '\\' {
				converter.escaped = !converter.escaped
			} else if b == '"' && !converter.escaped {
				converter.inStr = false
			}

			continue
		}

		switch b {
		case '"':
			converter.inStr = true
		case '{':
			converter.bracketSurplus += 1
		case ']':
			converter.bracketSurplus -= 1
		case '}':
			converter.bracketSurplus -= 1
			// Case where we just finished a block
			if converter.bracketSurplus == 1 {
				converter.consumingComma = true
				return index + 1, true
			}
		case '[':
			converter.bracketSurplus += 1

			// Special case where we are not yet into the array
			if converter.bracketSurplus == 1 {
				converter.consumedHeader = true
				return index + 1, false
			}
		}
	}

	return len(p), converter.consumedHeader
}

func (converter *ndjsonConverter) Write(p []byte) (progress int, err error) {
	for progress < len(p) {
		if converter.consumingComma {
			progress += converter.consumeComma(p[progress:])
			continue
		}

		bytesRead, ok := converter.countToObjectEnd(p[progress:])

		if converter.bracketSurplus >= 1 && ok {
			var n int
			if n, err = converter.writer.Write(p[progress : progress+bytesRead]); err != nil {
				progress += n
				return
			}

			if converter.bracketSurplus == 1 {
				if _, err := converter.writer.Write([]byte{'\n'}); err != nil {
					return progress, err
				}
			}
		}

		progress += bytesRead
	}

	return progress, nil
}
