package util

import (
	"log"
	"time"
)

// MovingStatistic is a statistic that reflects a moving window in time. It does not need to be 100% accurate to the
// provided period length so long as all MovingStatistic of that same period length reflect the same period when given
// the same upper bound and the actual period length is at last the duration of the specified period length.
type MovingStatistic interface {
	// IncrementUpperBound shifts up the observed region of this moving statistic. The given timestamp must be greater
	// than or equal to the previous upper bound.
	IncrementUpperBound(timestamp time.Time)
	// Append adds a new value to the moving statistic. The timestamp should coincide with the current observed period.
	// By extension, the timestamp should always be less than or equal to the current upper bound
	Append(value float64, timestamp time.Time)
}

// MovingSummation is a moving statistic that performs the summation of the observed values
type MovingSummation interface {
	MovingStatistic
	Sum() float64
}

const binCount int = 100

type binnedMovingSummation struct {
	alignment time.Time
	binPeriod time.Duration
	bins      [binCount + 1]float64
}

func (binnedSummation *binnedMovingSummation) binFor(timestamp time.Time) int {
	binLatest := binnedSummation.alignment.Add(binnedSummation.binPeriod)
	return int(binLatest.Sub(timestamp).Nanoseconds() / binnedSummation.binPeriod.Nanoseconds())
}

func (binnedSummation *binnedMovingSummation) shiftBins(shift int) {
	// Adjust shift to maximum value if too large
	if shift > binCount+1 {
		shift = binCount + 1
	}

	// Shift bins over by the specified shift amount
	copy(binnedSummation.bins[shift:], binnedSummation.bins[:])

	// Zero new bins at beginning of group
	for index := 0; index < shift; index++ {
		binnedSummation.bins[index] = 0.0
	}
}

func (binnedSummation *binnedMovingSummation) IncrementUpperBound(timestamp time.Time) {
	offset := timestamp.Sub(binnedSummation.alignment)
	shift := int(offset.Nanoseconds() / binnedSummation.binPeriod.Nanoseconds())

	if shift > 0 {
		binnedSummation.shiftBins(shift)
		binnedSummation.alignment = binnedSummation.alignment.Add(time.Duration(shift) * binnedSummation.binPeriod)
	}
}

func (binnedSummation *binnedMovingSummation) Append(value float64, timestamp time.Time) {
	targetBin := binnedSummation.binFor(timestamp)
	if targetBin < 0 {
		log.Fatalln("Unable to add to moving statistic at time after upper bound")
	}

	if targetBin < binCount+1 {
		binnedSummation.bins[targetBin] += value
	}
}

func (binnedSummation *binnedMovingSummation) Sum() (res float64) {
	for _, value := range binnedSummation.bins {
		res += value
	}
	return
}

func MakeMovingSummation(period time.Duration) MovingSummation {
	return &binnedMovingSummation{
		alignment: time.Unix(0, 0),
		binPeriod: time.Duration(period.Nanoseconds()/int64(binCount)) * time.Nanosecond,
	}
}

type MovingAverage interface {
	MovingStatistic
	Average() float64
}

type movingAverageImpl struct {
	sum   MovingSummation
	count MovingSummation
}

func (avg *movingAverageImpl) IncrementUpperBound(timestamp time.Time) {
	avg.sum.IncrementUpperBound(timestamp)
	avg.count.IncrementUpperBound(timestamp)
}

func (avg *movingAverageImpl) Append(value float64, timestamp time.Time) {
	avg.sum.Append(value, timestamp)
	avg.count.Append(1, timestamp)
}

func (avg *movingAverageImpl) Average() float64 {
	return avg.sum.Sum() / avg.count.Sum()
}

func MakeMovingAverage(period time.Duration) MovingAverage {
	return &movingAverageImpl{
		sum:   MakeMovingSummation(period),
		count: MakeMovingSummation(period),
	}
}
