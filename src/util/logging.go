package util

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"
)

func CloseAndLogErrors(source string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Println(source, err)
	}
}

// ProgressCounter is a helper for periodically logging the progress that has been made doing a task. It is completely
// thread safe so pointers can be shared between multiple worker threads without any issues.
type ProgressCounter struct {
	count         uint64
	lastTriggered uint64
	startTime     time.Time
	period        time.Duration
}

func (counter *ProgressCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}

func (counter *ProgressCounter) Increment() {
	atomic.AddUint64(&counter.count, 1)
}

func (counter *ProgressCounter) Periodic(callback func(uint64)) {
	prevActivate := atomic.LoadUint64(&counter.lastTriggered)
	prevActivateTime := counter.startTime.Add(time.Duration(prevActivate))

	if prevActivateTime.Add(counter.period).Before(time.Now()) {
		desiredEnd := time.Now().Add(counter.period).Sub(counter.startTime)

		if atomic.CompareAndSwapUint64(&counter.lastTriggered, prevActivate, uint64(desiredEnd.Nanoseconds())) {
			callback(atomic.LoadUint64(&counter.count))
		}
	}
}

func MakeProgressCounter(period time.Duration) ProgressCounter {
	return ProgressCounter{
		count:         0,
		lastTriggered: 0,
		startTime:     time.Now(),
		period:        period,
	}
}

type Benchmark[T any] struct {
	name         string
	count        uint64
	totalElapsed uint64
}

func MakeBenchmark[T any](name string) Benchmark[T] {
	return Benchmark[T]{
		name:         name,
		count:        0,
		totalElapsed: 0,
	}
}

func (benchmark *Benchmark[T]) Do(action func() T) T {
	startTime := time.Now()
	result := action()
	elapsed := time.Since(startTime)

	atomic.AddUint64(&benchmark.count, 1)
	atomic.AddUint64(&benchmark.totalElapsed, uint64(elapsed.Nanoseconds()))

	return result
}

func (benchmark *Benchmark[T]) String() string {
	count := atomic.LoadUint64(&benchmark.count)
	totalElapsed := atomic.LoadUint64(&benchmark.totalElapsed)

	if count == 0 {
		return fmt.Sprintf("%s [Count: 0, Net: --, OP/S: --]", benchmark.name)
	}

	totalTime := time.Duration(totalElapsed) * time.Nanosecond
	ops := time.Duration(totalElapsed/count) * time.Nanosecond

	return fmt.Sprintf("%s [Count: %d, Net: %v, Sec/OP: %v]", benchmark.name, count, totalTime, ops)
}
