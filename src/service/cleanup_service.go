package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/config"
	"github.com/jmeggitt/fastly_anycast_experiments.git/probe"
	"time"
)

type CleanupService struct {
	LastCleanup   time.Time
	CleanupPeriod time.Duration
}

func (CleanupService) Name() string {
	return "CleanupService"
}

func (service CleanupService) Init(*ApplicationState) (err error) {
	//State that the last cleanup is when the program initializes
	service.LastCleanup = time.Now()
	//The Cleanup period is given as an environment variable or default option
	service.CleanupPeriod = config.CleanupPeriod.GetDuration() //This I just realized should be different
	return
}

func (service CleanupService) Run(state *ApplicationState) (err error) {
	for {
		timeElapsed := time.Since(service.LastCleanup)

		//Wait for cleanup until Statistic Period has passed
		if timeElapsed < service.CleanupPeriod {
			<-time.After(service.CleanupPeriod - timeElapsed)
		} else {
			//Cleanup once the statistics period has reached
			//Lock the data
			state.TracerouteDataLock.Lock()
			state.ProbeDataLock.Lock()
			//Evict the old Traceroute Data
			state.TracerouteData.EvictOutdatedData()
			//Evict the old Probe data
			evictDestinationProbeMap(state, service)
			//Unlock the data
			state.TracerouteDataLock.Unlock()
			state.ProbeDataLock.Unlock()
			//Set the new clean up time
			service.LastCleanup = time.Now()
		}
	}
}

func evictDestinationProbeMap(state *ApplicationState, service CleanupService) {
	//Get the oldest time that we are keeping
	oldestAllowed := time.Now().Add(-service.CleanupPeriod)

	//Go through each destination ip
	for destIP, probeList := range state.DestinationToProbeMap {
		//Create a new Probe list with the up-to-date probes
		var newProbeList []*probe.ProbeUsage
		//Look through each probe connected to that destination IP
		for _, probeDest := range probeList {
			//If that probe is before the time we allow, then we remove it from the list
			if !(probeDest.LastUsed.Before(oldestAllowed)) {
				newProbeList = append(newProbeList, probeDest)
			}
		}
		//Add the new probe list to the destination map
		state.DestinationToProbeMap[destIP] = newProbeList
	}
}
