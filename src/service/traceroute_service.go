package service

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/jmeggitt/fastly_anycast_experiments.git/ripe_atlas"
	"github.com/jmeggitt/fastly_anycast_experiments.git/traceroute"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"time"
)

type TracerouteDataService struct{}

func (TracerouteDataService) Name() string {
	return "TracerouteDataService"
}

func (TracerouteDataService) Init(state *ApplicationState) (err error) {
	state.TracerouteData = traceroute.MakeTracerouteData()
	return
}

func (TracerouteDataService) handleIncomingMessages(state *ApplicationState, channel <-chan *measurement.Result) {
	logProgress := util.IsEnvFlagSet(util.LogTracerouteProgress)
	// The progress counter is a debugging tool which will periodically call the Periodic function with the number of
	// times that it has been invoked. This helps show that the program is receiving messages and is not stuck in an
	// invalid state.
	progressCounter := util.MakeProgressCounter(3 * time.Second)

loop:
	for {
		// If the logging the progress has been enabled, then we should periodically print how many messages we have
		// received. This is done using the progressCounter.
		if logProgress {
			progressCounter.Periodic(func(count uint64) {
				log.Println("[Traceroute Progress] Parsed a total of", count, "traceroute messages")
			})
		}

		select {
		case msg, ok := <-channel:
			// If channel is closed and there are no more messages to receive, break loop
			if !ok {
				break loop
			}

			//Check if the measurement actually exists
			if msg == nil {
				log.Println("Measurement was nil?")
				continue
			}

			state.BootstrapCollectProbeInfo(msg)

			// Increment the progress counter so it knows how many messages have been received when calling the periodic
			// function.
			progressCounter.Increment()

			// Since we mutate the shared traceroute state we need to ensure exclusive access to the traceroute state.
			// Unlike other systems where data is swapped out, traceroute data is regularly mutated in place leading to
			// a higher risk of undefined behavior from concurrent reading/writing.
			state.TracerouteDataLock.Lock()
			state.TracerouteData.AppendMeasurement(msg)
			state.TracerouteDataLock.Unlock()
		case <-time.After(3 * time.Second):
			// We could potentially be waiting for longer than the progress counter interval to receive a message. This
			// timeout simply breaks us out of waiting so the progress counter can call the periodic function.
		}
	}

	// If we reach the end of the traceroute data, log how many messages we encountered for debugging purposes. This is
	// always enabled since we want to know if an error causes no messages to be received.
	log.Println("[Traceroute Progress] Exited after parsing a total of", progressCounter.Count(), "traceroute messages")
}

func (service TracerouteDataService) Run(state *ApplicationState) (err error) {
	var resultChannel <-chan *measurement.Result
	if resultChannel, err = ripe_atlas.CachedGetTraceRouteData(46320619); err != nil {
		return
	}

	service.handleIncomingMessages(state, resultChannel)
	if resultChannel, err = ripe_atlas.CachedGetTraceRouteData(46320708); err != nil {
		return
	}

	service.handleIncomingMessages(state, resultChannel)
	return nil
}
