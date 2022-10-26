package asn

import (
	"encoding/binary"
	"net/netip"
	"strconv"
	"testing"
)

func expectContains(t *testing.T, prefixMap *PrefixMap[string], key netip.Addr, expected string) {
	value, present := prefixMap.GetAddr(key)

	if !present {
		t.Fatal("Failed to find expected key", key)
	}

	if value != expected {
		t.Fatalf("Map value \"%s\" does not match expected \"%s\"", value, expected)
	}
}

func TestPrefixMap_GetAddr(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	// Add prefixes at varying depths
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "a")
	prefixMap.Set(netip.MustParsePrefix("1.2.0.0/16"), "b")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "c")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "d")

	// Add some prefixes that start with the same bytes as the ones above to try to confuse it
	prefixMap.Set(netip.MustParsePrefix("0100::/8"), "e")
	prefixMap.Set(netip.MustParsePrefix("0102::/15"), "f")

	// Try overwriting some prefixes to ensure it handles it correctly
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "g")
	prefixMap.Set(netip.MustParsePrefix("0100::/8"), "h")

	// Test across IPs of different specificity
	expectContains(t, &prefixMap, netip.MustParseAddr("1.23.19.23"), "g")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.123.2"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.0.0"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.22"), "c")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "d")
}

func TestPrefixMapEdgeCases(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	// Test some edge cases
	prefixMap.Set(netip.MustParsePrefix("0.0.0.0/0"), "h")
	prefixMap.Set(netip.MustParsePrefix("1.2.128.0/17"), "i")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "j")
	prefixMap.Set(netip.MustParsePrefix("5.6.7.8/32"), "k")

	expectContains(t, &prefixMap, netip.MustParseAddr("22.1.24.6"), "h")
	expectContains(t, &prefixMap, netip.MustParseAddr("0.0.0.0"), "h")
	expectContains(t, &prefixMap, netip.MustParseAddr("255.255.255.255"), "h")

	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.133.235"), "i")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.128.0"), "i")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.255.255"), "i")

	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "j")
	expectContains(t, &prefixMap, netip.MustParseAddr("5.6.7.8"), "k")
}

func TestPrefixMapBitLenIPv4(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	addrBuffer := [4]byte{0, 0, 0, 0}
	for bitLen := 32; bitLen > 0; bitLen-- {
		mapValue := strconv.FormatInt(int64(bitLen), 10)

		addrBits := uint32(1) << (32 - bitLen)

		binary.BigEndian.PutUint32(addrBuffer[:], addrBits)
		firstAddr := netip.AddrFrom4(addrBuffer)

		binary.BigEndian.PutUint32(addrBuffer[:], addrBits|(addrBits-1))
		lastAddr := netip.AddrFrom4(addrBuffer)

		prefixMap.Set(netip.PrefixFrom(firstAddr, bitLen), mapValue)

		expectContains(t, &prefixMap, firstAddr, mapValue)
		expectContains(t, &prefixMap, lastAddr, mapValue)

		if x, ok := prefixMap.GetAddr(firstAddr.Prev()); ok && x == mapValue {
			t.Fatal("Address", firstAddr.Prev(), "should not be present in map for bit length of", bitLen)
		}

		if x, ok := prefixMap.GetAddr(lastAddr.Next()); ok && x == mapValue {
			t.Fatal("Address", lastAddr.Next(), "should not be present in map for bit length of", bitLen)
		}
	}
}

func TestPrefixMapBitLenIPv6(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	addrBuffer := [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for bitLen := 128; bitLen > 0; bitLen-- {
		mapValue := strconv.FormatInt(int64(bitLen), 10)

		var hi, lo uint64 = 0, 0
		if bitLen <= 64 {
			hi = uint64(1) << (64 - bitLen)
		} else {
			lo = uint64(1) << (128 - bitLen)
		}

		binary.BigEndian.PutUint64(addrBuffer[0:8], hi)
		binary.BigEndian.PutUint64(addrBuffer[8:16], lo)
		firstAddr := netip.AddrFrom16(addrBuffer)

		if bitLen <= 64 {
			hi = hi | (hi - 1)
			lo = ^lo
		} else {
			lo = lo | (lo - 1)
		}

		binary.BigEndian.PutUint64(addrBuffer[0:8], hi)
		binary.BigEndian.PutUint64(addrBuffer[8:16], lo)
		lastAddr := netip.AddrFrom16(addrBuffer)

		prefixMap.Set(netip.PrefixFrom(firstAddr, bitLen), mapValue)

		expectContains(t, &prefixMap, firstAddr, mapValue)
		expectContains(t, &prefixMap, lastAddr, mapValue)

		if x, ok := prefixMap.GetAddr(firstAddr.Prev()); ok && x == mapValue {
			t.Fatal("Address", firstAddr.Prev(), "should not be present in map for bit length of", bitLen)
		}

		if x, ok := prefixMap.GetAddr(lastAddr.Next()); ok && x == mapValue {
			t.Fatal("Address", lastAddr.Next(), "should not be present in map for bit length of", bitLen)
		}
	}
}

func TestIpToAsn(t *testing.T) {
	asnMap, err := CreateIpToAsn()
	if err != nil {
		t.Fatal("Failed to create IpToAsn:", err.Error())
	}

	if asnMap.Length() == 0 {
		t.Fatal("AsnMap does not contain any entries")
	}

	const FastlyAsn = 54113

	knownFastlyIPs := []string{
		// Fastly Anycast Addresses
		"151.101.0.1",
		"2a04:4e42::1",
		// Some other ips from Fastly's other largest prefixes
		"199.232.0.1",
		"2a04:4e41::1",
	}

	correctValues := 0
	for _, ip := range knownFastlyIPs {
		fastlyAddr := netip.MustParseAddr(ip)
		asn, ok := asnMap.Get(fastlyAddr)

		if ok && asn == FastlyAsn {
			correctValues += 1
		} else {
			t.Logf("Failed to identify %s as belonging to Fastly's ASN (%d). Response: { present: %v, ASN: %d}", ip, FastlyAsn, ok, asn)
		}
	}

	if correctValues < len(knownFastlyIPs)*2/3 {
		t.Fatalf("IpToAsn was only able to correctly identify %d/5 IP addresses in Fastly's ASN", correctValues)
	}
}

// Check ASN lookup against all the Root DNS servers (except j-root, because it used a couple ASNs)
func TestIpToAsnWithRootDns(t *testing.T) {
	asnMap, err := CreateIpToAsn()
	if err != nil {
		t.Fatal("Failed to create IpToAsn:", err.Error())
	}

	if asnMap.Length() == 0 {
		t.Fatal("AsnMap does not contain any entries")
	}

	type Pair struct {
		ip  string
		asn uint32
	}

	rootDnsServers := []Pair{
		{"198.41.0.4", 397197},
		{"2001:503:ba3e::2:30", 397197},
		{"199.9.14.201", 394353},
		{"2001:500:200::b", 394353},
		{"192.33.4.12", 2149},
		{"2001:500:2::c", 2149},
		{"199.7.91.13", 10886},
		{"2001:500:2d::d", 10886},
		{"192.203.230.10", 21556},
		{"2001:500:a8::e", 21556},
		{"192.5.5.241", 3557},
		{"2001:500:2f::f", 3557},
		{"192.112.36.4", 5927},
		{"2001:500:12::d0d", 5927},
		{"198.97.190.53", 1508},
		{"2001:500:1::53", 1508},
		{"192.36.148.17", 29216},
		{"2001:7fe::53", 29216},
		{"193.0.14.129", 25152},
		{"2001:7fd::1", 25152},
		{"199.7.83.42", 20144},
		{"2001:500:9f::42", 20144},
		{"202.12.27.33", 7500},
		{"2001:dc3::35", 7500},
	}

	correctValues := 0
	for _, pair := range rootDnsServers {
		fastlyAddr := netip.MustParseAddr(pair.ip)
		asn, ok := asnMap.Get(fastlyAddr)

		if ok && asn == pair.asn {
			correctValues += 1
		} else {
			t.Logf("Failed to identify %s as belonging to Root DNS server (%d). Response: { present: %v, ASN: %d}", pair.ip, pair.asn, ok, asn)
		}
	}

	if correctValues < len(rootDnsServers)*2/3 {
		t.Fatalf("IpToAsn was only able to correctly identify %d/5 IP addresses in Root DNS server list", correctValues)
	}
}
