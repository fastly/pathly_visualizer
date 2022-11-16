package util

import (
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
