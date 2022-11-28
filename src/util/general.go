package util

import (
	"errors"
	"os"
)

const defaultCacheDir = ".cache"

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
