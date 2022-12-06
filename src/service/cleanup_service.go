package service

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/util"
	"time"
)

const DefaultCleanupPeriod = 24 * time.Hour

type CleanupService struct {
	LastCleanup   time.Time
	CleanupPeriod time.Duration
}

func (CleanupService) Name() string {
	return "CleanupService"
}

func (service CleanupService) Init(state *ApplicationState) (err error) {
	//State that the last cleanup is when the program initializes
	service.LastCleanup = time.Now()
	//The Cleanup period is given as an environment variable or default option
	service.CleanupPeriod = util.GetEnvDuration(util.StatisticsPeriod, DefaultCleanupPeriod)
	return
}

func (service CleanupService) Run(state *ApplicationState) (err error) {
	for {
		timeElapsed := time.Since(service.LastCleanup)

		//Wait for cleanup until Statistic Period has passed
		if timeElapsed < service.CleanupPeriod {

		} else {
			//Cleanup once the statistics period has reached

		}
	}
	return
}
