package traceroute

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"time"
)

type TimeRange struct {
	Start, End time.Time
}

func (timeRange TimeRange) append(timestamp time.Time) TimeRange {
	if timeRange.Start.After(timestamp) {
		timeRange.Start = timestamp
	}

	if timeRange.End.Before(timestamp) {
		timeRange.End = timestamp
	}

	return timeRange
}

// clipTo returns a new TimeRange at or after the given timestamp. If the previous TimeRange only covered a period
// before the input timestamp, then false will be returned and the TimeRange should be ignored.
func (timeRange TimeRange) clipTo(timestamp time.Time) (TimeRange, bool) {
	if timeRange.End.Before(timestamp) {
		return timeRange, false
	}

	if timeRange.Start.Before(timestamp) {
		timeRange.Start = timestamp
	}

	return timeRange, true
}

type RouteUsageMetrics struct {
	MeasurementRanges map[int]TimeRange
}

func makeRouteUsageMetrics() RouteUsageMetrics {
	return RouteUsageMetrics{
		MeasurementRanges: make(map[int]TimeRange),
	}
}

func (metrics *RouteUsageMetrics) UsesSingleMeasurement(measurement int) bool {
	for id := range metrics.MeasurementRanges {
		if id != measurement {
			return false
		}
	}

	return true
}

func (metrics *RouteUsageMetrics) AppendMeasurement(measurement *measurement.Result) {
	timestamp := time.Unix(int64(measurement.Timestamp()), 0)

	value, ok := metrics.MeasurementRanges[measurement.MsmId()]
	if !ok {
		value = TimeRange{
			Start: timestamp,
			End:   timestamp,
		}
	}

	metrics.MeasurementRanges[measurement.MsmId()] = value.append(timestamp)
}

func (metrics *RouteUsageMetrics) EvictMetricsUpTo(timestamp time.Time) {
	for id, timeRange := range metrics.MeasurementRanges {
		if newRange, ok := timeRange.clipTo(timestamp); ok {
			metrics.MeasurementRanges[id] = newRange
		} else {
			// I had to check, but it is safe to remove items from a map while iterating through it
			delete(metrics.MeasurementRanges, id)
		}
	}
}
