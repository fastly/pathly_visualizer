package service

import (
	"errors"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"github.com/jmeggitt/fastly_anycast_experiments.git/ripe_atlas"
	"github.com/jmeggitt/fastly_anycast_experiments.git/traceroute"
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"log"
	"net/netip"
	"sync"
	"time"
)

type TracerouteDataService struct{}

func (TracerouteDataService) Name() string {
	return "TracerouteDataService"
}

func (TracerouteDataService) Init(state *ApplicationState) (err error) {
	state.TracerouteData = traceroute.MakeTracerouteData()
	state.StoredMeasurements = MakeMeasurementTracker()
	return
}

func (TracerouteDataService) handleIncomingMessages(state *ApplicationState, channel <-chan *measurement.Result) {
	logProgress := config.LogTracerouteProgress.GetAsFlag()
	// The progress counter is a debugging tool which will periodically call the Periodic function with the number of
	// times that it has been invoked. This helps show that the program is receiving messages and is not stuck in an
	// invalid state.
	progressCounter := util.MakeProgressCounter(3 * time.Second)

	var info *MeasurementCollectionInfo

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

			if info == nil {
				info = state.StoredMeasurements.getOrCreateMeasurement(msg.MsmId())
			}

			info.Lock.Lock()
			info.UpdateLatestMeasurement(msg)
			info.Lock.Unlock()

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

func handleRetrieveHistory(state *ApplicationState, info *MeasurementCollectionInfo) {
	channel, closer, err := ripe_atlas.GetLatestTraceRouteData(info.Id)
	defer info.SetCollectingHistory(false)
	defer log.Println("Finished collecting history on measurement", info.Id)

	if err != nil {
		log.Println("Encountered error when trying to fetch measurement history:", err)
		return
	}

	defer util.CloseAndLogErrors("Failed to close request for measurement results", closer)

	for {
		msg, ok := <-channel

		// If channel is closed and there are no more messages to receive, break loop
		if !ok {
			break
		}

		// Check if the measurement actually exists
		if msg == nil {
			log.Println("Encountered nil message while retrieving history")
			continue
		}

		info.Lock.Lock()
		info.UpdateLatestMeasurement(msg)
		info.Lock.Unlock()

		state.TracerouteDataLock.Lock()
		state.TracerouteData.AppendMeasurement(msg)
		state.TracerouteDataLock.Unlock()
	}
}

func handleLiveCollection(state *ApplicationState, info *MeasurementCollectionInfo) {
	channel, err := ripe_atlas.GetStreamingTraceRouteData(info.Id)
	defer info.SetPerformingLiveCollection(false)
	defer log.Println("Exiting live collection goroutine for measurement", info.Id)

	if err != nil {
		log.Println("Encountered error when trying to fetch measurement history:", err)
		return
	}

loop:
	for {
		select {
		case msg, ok := <-channel:
			// If channel is closed and there are no more messages to receive, break loop
			if !ok {
				break loop
			}

			// Check if the measurement actually exists
			if msg == nil {
				log.Println("Encountered nil message while retrieving history")
				continue loop
			}

			state.TracerouteDataLock.Lock()
			state.TracerouteData.AppendMeasurement(msg)
			state.TracerouteDataLock.Unlock()

			info.Lock.Lock()
			info.UpdateLatestMeasurement(msg)

			if info.RequestStopLiveCollection {
				info.RequestStopLiveCollection = false
				info.Lock.Unlock()
				return
			}

			info.Lock.Unlock()
		}
	}
}

func (service TracerouteDataService) Run(state *ApplicationState) (err error) {
	var resultChannel <-chan *measurement.Result

	for _, id := range config.DebugMeasurementList.GetIntList() {
		log.Println("Loading debug measurement ID", id)
		if resultChannel, err = ripe_atlas.CachedGetTraceRouteData(id); err != nil {
			return
		}

		service.handleIncomingMessages(state, resultChannel)
	}
	log.Println("Finished adding debug measurements")

	for {
		action, ok := <-state.StoredMeasurements.requestChannel
		if !ok {
			break
		}

		service.handleAction(state, action)
	}

	return errors.New("traceroute action channel closed unexpectedly")
}

func (service TracerouteDataService) handleAction(state *ApplicationState, action CollectionMessage) {
	info := state.StoredMeasurements.getOrCreateMeasurement(action.target)
	info.Lock.Lock()
	defer info.Lock.Unlock()

	switch action.action {
	case CollectHistory:
		if info.CollectingHistory {
			return
		}

		log.Println("Collecting history on measurement", info.Id)
		info.CollectingHistory = true
		go handleRetrieveHistory(state, info)
	case StartLiveCollection:
		info.RequestStopLiveCollection = false
		if info.PerformingLiveCollection {
			return
		}

		log.Println("Starting live collection on measurement", info.Id)
		info.PerformingLiveCollection = true
		go handleLiveCollection(state, info)
	case StopLiveCollection:
		log.Println("Requesting to stop live collection on measurement", info.Id)
		info.RequestStopLiveCollection = info.PerformingLiveCollection
	}
}

var (
	ErrMeasurementDoesNotExist = errors.New("specified measurement ID does not exist")
	ErrMeasurementAlreadyInUse = errors.New("specified measurement ID is already being collected")
	ErrNotUsingLiveCollection  = errors.New("specified measurement ID not being used for live collection")
)

type actionType = int

type MeasurementTracker struct {
	TrackedMeasurements sync.Map
	requestChannel      chan CollectionMessage
}

func MakeMeasurementTracker() MeasurementTracker {
	return MeasurementTracker{
		requestChannel: make(chan CollectionMessage, 64),
	}
}

func (tracker *MeasurementTracker) sendMessage(measurement int, action actionType) {
	tracker.requestChannel <- CollectionMessage{
		action: action,
		target: measurement,
	}
}

func (tracker *MeasurementTracker) getOrCreateMeasurement(measurement int) *MeasurementCollectionInfo {
	if value, ok := tracker.TrackedMeasurements.Load(measurement); ok {
		return value.(*MeasurementCollectionInfo)
	}

	// Create new data and add it to the map
	newData := &MeasurementCollectionInfo{
		Id:                        measurement,
		PerformingLiveCollection:  false,
		CollectingHistory:         false,
		RequestStopLiveCollection: false,
		LatestData:                time.Unix(0, 0),
		OldestData:                time.Unix(0, 0),
	}

	value, _ := tracker.TrackedMeasurements.LoadOrStore(measurement, newData)
	return value.(*MeasurementCollectionInfo)
}

const (
	CollectHistory      actionType = 0
	StartLiveCollection            = 1
	StopLiveCollection             = 2
)

type CollectionMessage struct {
	action actionType
	target int
}

type MeasurementCollectionInfo struct {
	Id                        int
	DestinationIp             netip.Addr
	PerformingLiveCollection  bool
	CollectingHistory         bool
	RequestStopLiveCollection bool
	LatestData                time.Time
	OldestData                time.Time

	Lock sync.Mutex
}

func (info *MeasurementCollectionInfo) UpdateLatestMeasurementTimestamp(timestamp time.Time) {
	if info.OldestData.After(timestamp) || info.OldestData == time.Unix(0, 0) {
		info.OldestData = timestamp
	}

	if info.LatestData.Before(timestamp) {
		info.LatestData = timestamp
	}
}

func (info *MeasurementCollectionInfo) UpdateLatestMeasurement(msg *measurement.Result) {
	timestamp := time.Unix(int64(msg.Timestamp()), 0)
	if info.OldestData.After(timestamp) || info.OldestData == time.Unix(0, 0) {
		info.OldestData = timestamp
	}

	if info.LatestData.Before(timestamp) {
		info.LatestData = timestamp
	}

	if !info.DestinationIp.IsValid() {
		if ip, err := netip.ParseAddr(msg.DstAddr()); err == nil {
			info.DestinationIp = ip
		}
	}
}

func (info *MeasurementCollectionInfo) SetCollectingHistory(value bool) {
	info.Lock.Lock()
	info.CollectingHistory = value
	info.Lock.Unlock()
}

func (info *MeasurementCollectionInfo) SetPerformingLiveCollection(value bool) {
	info.Lock.Lock()
	info.PerformingLiveCollection = value
	info.Lock.Unlock()
}

func (state *ApplicationState) CollectMeasurementHistory(measurement int) error {
	collectionInfo := state.StoredMeasurements.getOrCreateMeasurement(measurement)
	collectionInfo.Lock.Lock()
	defer collectionInfo.Lock.Unlock()

	// Check if collection was already performed so we can notify the requester
	if collectionInfo.CollectingHistory {
		return ErrMeasurementAlreadyInUse
	}

	state.StoredMeasurements.requestChannel <- CollectionMessage{
		action: CollectHistory,
		target: measurement,
	}

	return nil
}

func (state *ApplicationState) EnableLiveMeasurementCollection(measurement int) error {
	collectionInfo := state.StoredMeasurements.getOrCreateMeasurement(measurement)
	collectionInfo.Lock.Lock()
	defer collectionInfo.Lock.Unlock()

	// Check if collection was already performed so we can notify the requester
	if collectionInfo.PerformingLiveCollection {
		return ErrMeasurementAlreadyInUse
	}

	state.StoredMeasurements.requestChannel <- CollectionMessage{
		action: StartLiveCollection,
		target: measurement,
	}

	return nil
}

func (state *ApplicationState) DisableLiveMeasurementCollection(measurement int) error {
	collectionInfo := state.StoredMeasurements.getOrCreateMeasurement(measurement)
	collectionInfo.Lock.Lock()
	defer collectionInfo.Lock.Unlock()

	// Check if collection was already performed so we can notify the requester
	if !collectionInfo.PerformingLiveCollection {
		return ErrNotUsingLiveCollection
	}

	state.StoredMeasurements.requestChannel <- CollectionMessage{
		action: StopLiveCollection,
		target: measurement,
	}

	return nil
}

func (state *ApplicationState) DropMeasurementData(measurement int) error {
	if _, ok := state.StoredMeasurements.TrackedMeasurements.Load(measurement); !ok {
		return ErrMeasurementDoesNotExist
	}

	state.TracerouteDataLock.Lock()
	defer state.TracerouteDataLock.Unlock()

	state.TracerouteData.DropMeasurementData(measurement)
	return nil
}
