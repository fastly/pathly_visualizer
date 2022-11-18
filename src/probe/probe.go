package probe

import (
	"net/netip"
)

type Probe struct {
	Id          int
	Ipv4        netip.Addr
	Ipv6        netip.Addr
	CountryCode string
	Asn4        uint32
	Asn6        uint32
	Type        string
	Coordinates []float64
}
