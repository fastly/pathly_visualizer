package service

import (
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/jmeggitt/fastly_anycast_experiments.git/ripe_atlas"
	"github.com/jmeggitt/fastly_anycast_experiments.git/traceroute"
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

func (TracerouteDataService) Run(state *ApplicationState) (err error) {
	var resultChannel <-chan *measurement.Result
	if resultChannel, err = ripe_atlas.GetStreamingTraceRouteData(46320619); err != nil {
		return
	}

	var received uint64 = 0
	var receivedNil uint64 = 0

	var waitTime time.Duration = 0
	var pushTime time.Duration = 0

	const TimeoutDuration = 1 * time.Second

	startTime := time.Now()

	lastLog := time.Now()

loop:
	for {
		if time.Since(lastLog) > time.Second {
			lastLog = time.Now()
			log.Println("[Progress] Received", received, "traceroute messages with", receivedNil, "messages nil")
		}

		waitStartTime := time.Now()
		select {
		case msg := <-resultChannel:
			waitTime += time.Since(waitStartTime)
			received += 1
			if msg == nil {
				receivedNil += 1
				break
			} else if received == 1 {
				log.Println("Received first message after", time.Since(startTime))
			}
			state.tracerouteDataLock.Lock()
			waitStartTime = time.Now()
			state.TracerouteData.AppendMeasurement(msg)
			pushTime += time.Since(waitStartTime)
			state.tracerouteDataLock.Unlock()
		case <-time.After(TimeoutDuration):
			if received > 10000 {
				break loop
			}
		}
	}

	log.Println("Received a total of", received, "messaged and finished after", time.Since(startTime)-TimeoutDuration)
	log.Println("\tTotal nil messages:", receivedNil)
	log.Println("\tTime spent waiting on channel:", waitTime)
	log.Println("\tTime spent pushing to data:", pushTime)

	log.Fatal("Exiting...")
	return nil
}
