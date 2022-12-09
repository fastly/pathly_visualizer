package probe

import (
	"net/netip"
	"time"
)

type Probe struct {
	Id          int        //Unique identifer for the probe. Used as the key in ProbeCollection
	Ipv4        netip.Addr //Net address for IPv4 would be nil if the Probe is IPv6
	Ipv6        netip.Addr //Net address for IPv6 would be nil if the Probe is IPv4
	CountryCode string     //Two character code that relates to a country
	Asn4        uint32     //ASN if the probe is IPv4, is nil if IPv6
	Asn6        uint32     //ASN if the probe is IPv6, is nil if IPv4
	Type        string     //Type of the GeoJson format will mostly be a "Point"
	Coordinates []float64  //Coordinates from the GeoJson. Will be [Longitude, Latitude]
	// Both Type and Coordinates come together to form part of a GeoJson
}

type ProbeUsage struct {
	Probe    *Probe
	LastUsed time.Time
}
