package config

import (
	"time"
)

var (
	// StatisticsPeriod refers to the duration statistics are stored/collected for on measurements
	StatisticsPeriod = makeConfig("STATISTICS_PERIOD", 14*24*time.Hour)

	// LogTracerouteProgress enables logging the progress of traceroute ingestion
	LogTracerouteProgress = makeConfig("LOG_TRACEROUTE_PROGRESS", false)

	// CacheDirectory and CacheStoreDuration control where and how long cached files traceroute data are stored
	CacheDirectory     = makeConfig("CACHE_DIR", ".cache")
	CacheStoreDuration = makeConfig("CACHE_DURATION", 12*time.Hour)

	MinCleanEdgeWeight = makeConfig("MIN_CLEAN_EDGE_WEIGHT", 0.1)

	ProbeCollectionRefreshPeriod = makeConfig("PROBE_COLLECTION_REFRESH_PERIOD", 24*time.Hour)

	RequestByteLimit = makeConfig("REQUEST_BYTE_LIMIT", 4096)

	DebugMeasurementList = makeConfig("ATLAS_DEBUG_MEASUREMENTS", []int{47072659, 47072660})

	// CleanupPeriod refers to how often we clean up our data
	CleanupPeriod = makeConfig("CLEANUP_PERIOD", 24*time.Hour)
)
