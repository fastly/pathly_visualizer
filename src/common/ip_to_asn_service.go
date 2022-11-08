package common

import (
	"github.com/jmeggitt/fastly_anycast_experiments.git/asn"
	"log"
	"net/netip"
	"time"
)

// IpToAsnRefreshPeriod is the time between refreshes of the IpToAsn mapping. CAIDA recommends a refresh period of
// between 12 and 24 hours, so we use the lower recommended duration
const IpToAsnRefreshPeriod = 12 * time.Hour

type IpToAsnService struct{}

func (IpToAsnService) Name() string {
	return "IpToAsnService"
}

func (IpToAsnService) Init(state *ApplicationState) (err error) {
	// No locking needed since init is done in a single threaded context
	state.IpToAsn, err = asn.CreateIpToAsn()
	return
}

func (IpToAsnService) Run(state *ApplicationState) error {
	for {
		state.ipToAsnRefreshLock.RLock()
		timeElapsed := time.Since(state.IpToAsn.LastRefresh())
		state.ipToAsnRefreshLock.RUnlock()

		if timeElapsed < IpToAsnRefreshPeriod {
			time.Sleep(IpToAsnRefreshPeriod - timeElapsed)
		} else {
			state.ipToAsnRefreshLock.Lock()
			// Since it will probably succeed on the next attempt it should be close enough even if we encounter an
			// error every so often. Just log errors instead of stopping the service.
			if err := state.IpToAsn.Refresh(); err != nil {
				log.Println("Got error while attempting to refresh IP to ASN:", err)
			}
			state.ipToAsnRefreshLock.Unlock()
		}
	}
}

// GetIpToAsn extends the functionality of ApplicationState by adding a convenient thread-safe way to convert an ip to
// an asn.
func (state *ApplicationState) GetIpToAsn(ip netip.Addr) (asn uint32, present bool) {
	state.ipToAsnRefreshLock.RLock()
	asn, present = state.IpToAsn.Get(ip)
	state.ipToAsnRefreshLock.RUnlock()
	return
}
