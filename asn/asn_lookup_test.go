package asn

import (
	"net/netip"
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
