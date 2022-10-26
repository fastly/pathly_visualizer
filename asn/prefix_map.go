package asn

import "net/netip"

type PrefixMap[T any] struct {
	// An ordered map like a BTreeMap could have been better, but I was unable to find a good generic implementation
	inner map[netip.Prefix]T
	ipv4  prefixBitRange
	ipv6  prefixBitRange
}

func MakePrefixMap[T any]() PrefixMap[T] {
	return PrefixMap[T]{
		make(map[netip.Prefix]T),
		prefixBitRange{32, 0},
		prefixBitRange{128, 0},
	}
}

func (prefixMap *PrefixMap[T]) Length() int {
	return len(prefixMap.inner)
}

func (prefixMap *PrefixMap[T]) Clear() {
	prefixMap.inner = make(map[netip.Prefix]T)
	prefixMap.ipv4 = prefixBitRange{32, 0}
	prefixMap.ipv6 = prefixBitRange{128, 0}
}

func (prefixMap *PrefixMap[T]) Set(prefix netip.Prefix, value T) {
	prefixMap.inner[prefix.Masked()] = value

	if prefix.Addr().Is4() {
		prefixMap.ipv4.add(prefix.Bits())
	} else {
		prefixMap.ipv6.add(prefix.Bits())
	}
}

func (prefixMap *PrefixMap[T]) Get(prefix netip.Prefix) (value T, present bool) {
	value, present = prefixMap.inner[prefix.Masked()]
	return
}

func (prefixMap *PrefixMap[T]) GetAddr(addr netip.Addr) (value T, present bool) {
	bitRange := prefixMap.ipv4

	if addr.Is6() {
		bitRange = prefixMap.ipv6
	}

	for bits := bitRange.max; bits >= bitRange.min && !present; bits-- {
		prefix := netip.PrefixFrom(addr, bits).Masked()
		value, present = prefixMap.inner[prefix]
	}

	return
}

func (prefixMap *PrefixMap[T]) Remove(prefix netip.Prefix) {
	delete(prefixMap.inner, prefix.Masked())
}

type prefixBitRange struct {
	min int
	max int
}

func (bitRange *prefixBitRange) add(value int) {
	if value < bitRange.min {
		bitRange.min = value
	}

	if value > bitRange.max {
		bitRange.max = value
	}
}
