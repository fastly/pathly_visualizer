package util

import (
	"runtime"
	"sync/atomic"
)

// ArcCloser is an Atomic Reference Counter that assists in closing shared in a simple and readable manor once all
// goroutines using those resources have exited. This structure is used by creating with the number of goroutines that
// will be using a resource then having each routine call ArcCloser.Close(closerFunc) when exiting. Only the last
// goroutine to exit will result in closerFunc being called.
type ArcCloser struct {
	arc *uintptr
}

func MakeArcCloser(num uintptr) ArcCloser {
	return ArcCloser{
		arc: &num,
	}
}

func (arcCloser ArcCloser) Close(closer func()) {
	for {
		previous := atomic.LoadUintptr(arcCloser.arc)
		next := previous - 1

		if atomic.CompareAndSwapUintptr(arcCloser.arc, previous, next) {
			if next == 0 {
				closer()
			}

			return
		}
	}
}

// MakeWorkGroupWith starts up a group of goroutines that will process inputs from the input channel. Upon receiving an
// input, the handler function will be called and it will have the option to send any number of outputs to the output
// channel. These goroutines will continue to run until the input channel is closed. At that point the output channel
// will also be closed and the goroutines will exit.
//
// Note: The number of goroutines is the number of CPUs on the system and output channel is a bounded channel that can
// buffer 64 values
func MakeWorkGroupWith[I, O any](buffered int, input <-chan I, handler func(I, chan O)) <-chan O {
	output := make(chan O, buffered)
	closer := MakeArcCloser(uintptr(runtime.NumCPU()))

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				value, ok := <-input
				if !ok {
					break
				}

				handler(value, output)
			}

			closer.Close(func() {
				close(output)
			})
		}()
	}

	return output
}

func MakeWorkGroup[I, O any](buffered int, handler func(I, chan O)) (chan I, <-chan O) {
	inputChannel := make(chan I, buffered)
	return inputChannel, MakeWorkGroupWith(buffered, inputChannel, handler)
}
