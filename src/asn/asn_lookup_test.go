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

func expectDoesNotContain(t *testing.T, prefixMap *PrefixMap[string], key netip.Addr) {
	if value, present := prefixMap.GetAddr(key); present {
		t.Fatalf("Expected key %s to not be present, but found \"%s\"", key.String(), value)
	}
}

func TestPrefixMap_GetAddr(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	// Add prefixes at varying depths
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "a")
	prefixMap.Set(netip.MustParsePrefix("1.2.0.0/16"), "b")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "c")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "d")

	// Test across IPs of different specificity
	expectContains(t, &prefixMap, netip.MustParseAddr("1.23.19.23"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.123.2"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.0.0"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.22"), "c")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "d")
}

func TestPrefixMapOverwrite(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	// Insert IPv4 and IPv6 prefixes into map
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "a")
	prefixMap.Set(netip.MustParsePrefix("1.2.0.0/16"), "b")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "c")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "d")

	prefixMap.Set(netip.MustParsePrefix("0100::/8"), "e")
	prefixMap.Set(netip.MustParsePrefix("0102::/16"), "f")
	prefixMap.Set(netip.MustParsePrefix("0102:0300::/24"), "g")
	prefixMap.Set(netip.MustParsePrefix("0102:0304::/32"), "h")

	// Re-insert the same prefixes in arbitrary order
	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "k")
	prefixMap.Set(netip.MustParsePrefix("0102:0304::/32"), "p")
	prefixMap.Set(netip.MustParsePrefix("1.2.0.0/16"), "j")
	prefixMap.Set(netip.MustParsePrefix("0100::/8"), "m")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "l")
	prefixMap.Set(netip.MustParsePrefix("0102:0300::/24"), "o")
	prefixMap.Set(netip.MustParsePrefix("0102::/16"), "n")
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "i")

	// Check if new values are retrieved when picking an address from each prefix
	expectContains(t, &prefixMap, netip.MustParseAddr("1.34.234.12"), "i")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.0.32"), "j")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.0"), "k")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "l")

	expectContains(t, &prefixMap, netip.MustParseAddr("0100::"), "m")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102::"), "n")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0300::"), "o")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0304::"), "p")
}

func TestPrefixMapOverlap(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "a")
	prefixMap.Set(netip.MustParsePrefix("1.2.0.0/16"), "b")

	prefixMap.Set(netip.MustParsePrefix("0100::/8"), "e")
	prefixMap.Set(netip.MustParsePrefix("0102::/16"), "f")
	prefixMap.Set(netip.MustParsePrefix("0102:0300::/24"), "g")
	prefixMap.Set(netip.MustParsePrefix("0102:0304::/32"), "h")

	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "c")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "d")

	expectContains(t, &prefixMap, netip.MustParseAddr("1.0.0.0"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.0.0"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.0"), "c")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "d")

	expectContains(t, &prefixMap, netip.MustParseAddr("0100::"), "e")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102::"), "f")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0300::"), "g")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0304::"), "h")
}

func TestPrefixMapOverlapHigherSpecificity(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	// Alternate between IPv4 and IPv6 with more specific versions of similar prefixes
	prefixMap.Set(netip.MustParsePrefix("1.0.0.0/8"), "a")
	prefixMap.Set(netip.MustParsePrefix("0102::/16"), "b")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.0/24"), "c")
	prefixMap.Set(netip.MustParsePrefix("0102:0304::/30"), "d")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.5/32"), "e")

	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("0100::"))
	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("01F0::"))
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.0.0"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.128.0"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102::"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0300::"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.0"), "c")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0304::"), "d")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0305::"), "d")
	expectContains(t, &prefixMap, netip.MustParseAddr("0102:0305:1234::"), "d")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.4"), "c")
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.3.5"), "e")

}

func TestPrefixMapSingleIP(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	prefixMap.Set(netip.MustParsePrefix("0.0.0.0/32"), "a")
	prefixMap.Set(netip.MustParsePrefix("135.202.25.19/32"), "b")
	prefixMap.Set(netip.MustParsePrefix("255.255.255.255/32"), "c")

	prefixMap.Set(netip.MustParsePrefix("::/128"), "d")
	prefixMap.Set(netip.MustParsePrefix("1f32:1234::abcd/128"), "e")
	prefixMap.Set(netip.MustParsePrefix("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"), "f")

	expectContains(t, &prefixMap, netip.MustParseAddr("0.0.0.0"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("135.202.25.19"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("255.255.255.255"), "c")

	expectContains(t, &prefixMap, netip.MustParseAddr("::"), "d")
	expectContains(t, &prefixMap, netip.MustParseAddr("1f32:1234::abcd"), "e")
	expectContains(t, &prefixMap, netip.MustParseAddr("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"), "f")
}

func TestPrefixMapLargestPrefix(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	prefixMap.Set(netip.MustParsePrefix("0.0.0.0/0"), "a")
	prefixMap.Set(netip.MustParsePrefix("::/0"), "b")

	expectContains(t, &prefixMap, netip.MustParseAddr("0.0.0.0"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("135.202.25.19"), "a")
	expectContains(t, &prefixMap, netip.MustParseAddr("255.255.255.255"), "a")

	expectContains(t, &prefixMap, netip.MustParseAddr("::"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("1f32:1234::abcd"), "b")
	expectContains(t, &prefixMap, netip.MustParseAddr("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"), "b")
}

func TestPrefixMap_RemoveRange_Single(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/24"), "a")
	prefixMap.Set(netip.MustParsePrefix("5.6.7.8/32"), "b")
	prefixMap.Set(netip.MustParsePrefix("1f32:1234::abcd/48"), "c")
	prefixMap.Set(netip.MustParsePrefix("2f32:3434::1234/128"), "d")

	prefixMap.RemoveRange(netip.MustParsePrefix("1.2.3.4/24"))
	prefixMap.RemoveRange(netip.MustParsePrefix("5.6.7.8/32"))
	prefixMap.RemoveRange(netip.MustParsePrefix("1f32:1234::abcd/48"))
	prefixMap.RemoveRange(netip.MustParsePrefix("2f32:3434::1234/128"))

	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("1.2.3.4"))
	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("5.6.7.8"))
	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("1f32:1234::abcd"))
	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("2f32:3434::1234"))
}

func TestPrefixMap_RemoveRange_Children(t *testing.T) {
	prefixMap := MakePrefixMap[string]()

	prefixMap.Set(netip.MustParsePrefix("1.2.3.4/32"), "a")
	prefixMap.Set(netip.MustParsePrefix("1.2.3.5/32"), "b")
	prefixMap.Set(netip.MustParsePrefix("1.2.4.0/24"), "c")

	prefixMap.RemoveRange(netip.MustParsePrefix("1.2.3.0/24"))

	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("1.2.3.4"))
	expectDoesNotContain(t, &prefixMap, netip.MustParseAddr("1.2.3.5"))
	expectContains(t, &prefixMap, netip.MustParseAddr("1.2.4.5"), "c")
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
			hi = uint64(101) << (64 - bitLen)
		} else {
			lo = uint64(101) << (128 - bitLen)
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
