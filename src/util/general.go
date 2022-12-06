package util

import (
	"errors"
	"io"
	"os"
)

const defaultCacheDir = ".cache"
const defaultByteLimit = 512

var ErrMessageTooLong = errors.New("message is too long")

func GetCacheDir() (string, error) {
	cachePath, ok := os.LookupEnv(CacheDirectory)
	if !ok {
		cachePath = defaultCacheDir
	}

	stat, err := os.Stat(cachePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(cachePath, os.ModePerm)
	} else if err == nil && !stat.IsDir() {
		err = errors.New(CacheDirectory + " must point to a directory")
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
	b := make([]byte, 0, min(defaultByteLimit, limit))

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
