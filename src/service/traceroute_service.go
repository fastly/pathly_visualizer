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
	logProgress := util.IsEnvFlagSet("LOG_TRACEROUTE_PROGRESS")
	progressCounter := util.MakeProgressCounter(3 * time.Second)

loop:
	for {
		if logProgress {
			progressCounter.Periodic(func(count uint64) {
				log.Println("[Traceroute Progress] Parsed a total of", count, "traceroute messages")
			})
		}

		select {
		case msg, ok := <-channel:
			if !ok {
				break loop
			}

			progressCounter.Increment()
			state.TracerouteDataLock.Lock()
			state.TracerouteData.AppendMeasurement(msg)
			state.TracerouteDataLock.Unlock()
		case <-time.After(3 * time.Second):
			// Continue loop to allow progress counter to run
		}
	}

	log.Println("[Traceroute Progress] Exited after parsing a total of", progressCounter.Count(), "traceroute messages")
}

func (service TracerouteDataService) Run(state *ApplicationState) (err error) {
	var resultChannel <-chan *measurement.Result
	if resultChannel, err = ripe_atlas.CachedGetTraceRouteData(46320619); err != nil {
		return
	}

	service.handleIncomingMessages(state, resultChannel)
	return nil
}
