package asn

import (
	"log"
	"net/netip"
)
import "github.com/jmeggitt/nradix"

type PrefixMap[T any] struct {
	ipv4 *nradix.Tree
	ipv6 *nradix.Tree
}

func MakePrefixMap[T any]() PrefixMap[T] {
	return PrefixMap[T]{
		ipv4: nradix.NewTree(0),
		ipv6: nradix.NewTree(0),
	}
}

func (prefixMap *PrefixMap[T]) Clear() {
	prefixMap.ipv4 = nradix.NewTree(0)
	prefixMap.ipv6 = nradix.NewTree(0)
}

func (prefixMap *PrefixMap[T]) forAddressFamily(prefix netip.Addr) *nradix.Tree {
	if prefix.Is4() {
		return prefixMap.ipv4
	} else {
		return prefixMap.ipv6
	}
}

func (prefixMap *PrefixMap[T]) Set(prefix netip.Prefix, value T) {
	tree := prefixMap.forAddressFamily(prefix.Addr())

	if err := tree.SetCIDR(prefix.String(), value); err != nil && prefix.IsValid() {
		// Perform safety check to verify that no errors are created
		log.Panic("SetCIDR returned error on valid prefix (", prefix, "): ", err)
	}
}

func (prefixMap *PrefixMap[T]) Get(prefix netip.Prefix) (value T, present bool) {
	tree := prefixMap.forAddressFamily(prefix.Addr())

	if found, err := tree.FindCIDR(prefix.String()); err != nil && prefix.IsValid() {
		// Perform safety check to verify that no errors are created
		log.Panic("FindCIDR returned error on valid prefix (", prefix, "): ", err)
	} else if found != nil {
		value = found.(T)
		present = true
	}

	return
}

func (prefixMap *PrefixMap[T]) GetAddr(addr netip.Addr) (value T, present bool) {
	tree := prefixMap.forAddressFamily(addr)

	if found, err := tree.FindCIDR(addr.String()); err != nil && addr.IsValid() {
		// Perform safety check to verify that no errors are created
		log.Panic("FindCIDR returned error on valid address (", addr, "): ", err)
	} else if found != nil {
		value = found.(T)
		present = true
	}

	return
}

func (prefixMap *PrefixMap[T]) Remove(prefix netip.Prefix) {
	tree := prefixMap.forAddressFamily(prefix.Addr())

	if err := tree.DeleteCIDR(prefix.String()); err != nil && prefix.IsValid() {
		// Perform safety check to verify that no errors are created
		log.Panic("DeleteCIDR returned error on valid prefix (", prefix, "): ", err)
	}
}

func (prefixMap *PrefixMap[T]) RemoveRange(prefix netip.Prefix) {
	tree := prefixMap.forAddressFamily(prefix.Addr())

	if err := tree.DeleteWholeRangeCIDR(prefix.String()); err != nil && prefix.IsValid() {
		// Perform safety check to verify that no errors are created
		log.Panic("DeleteCIDR returned error on valid prefix (", prefix, "): ", err)
	}
}
