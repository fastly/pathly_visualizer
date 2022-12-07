package util

import (
	"errors"
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"io"
	"os"
)

var ErrMessageTooLong = errors.New("message is too long")

func GetCacheDir() (string, error) {
	cachePath := config.CacheDirectory.GetString()

	stat, err := os.Stat(cachePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(cachePath, os.ModePerm)
	} else if err == nil && !stat.IsDir() {
		err = errors.New("cache path is not directory")
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

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func ReadAtMost(r io.Reader, limit int) ([]byte, error) {
	// Initialize buffer to at most the size of a single page of memory for efficiency while reading. On most systems
	// this is 4096, 16384, or 65536.
	b := make([]byte, 0, min(os.Getpagesize(), limit))

	for {
		if len(b) >= limit {
			return b, ErrMessageTooLong
		}
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):min(limit, cap(b))])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}
