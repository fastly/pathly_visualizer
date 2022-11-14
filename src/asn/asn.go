package asn

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const (
	CaidaPrefix2AsnIpv4   = "https://publicdata.caida.org/datasets/routing/routeviews-prefix2as/"
	CaidaPrefix2AsnIpv6   = "https://publicdata.caida.org/datasets/routing/routeviews6-prefix2as/"
	Prefix2AsnCreationLog = "pfx2as-creation.log"
)

type IpToAsn struct {
	asnMap      PrefixMap[uint32]
	lastRefresh time.Time
}

func CreateIpToAsn() (ipToAsn IpToAsn, err error) {
	ipToAsn.asnMap = MakePrefixMap[uint32]()
	err = ipToAsn.Refresh()
	return
}

func (ipToAsn *IpToAsn) LastRefresh() time.Time {
	return ipToAsn.lastRefresh
}

func (ipToAsn *IpToAsn) Refresh() (err error) {
	ipToAsn.lastRefresh = time.Now()

	if err = ipToAsn.refreshFromSource(CaidaPrefix2AsnIpv4); err != nil {
		return
	}

	return ipToAsn.refreshFromSource(CaidaPrefix2AsnIpv6)
}

func (ipToAsn *IpToAsn) refreshFromSource(searchDir string) (err error) {
	var searchUrl string
	if searchUrl, err = latestCaidaData(searchDir); err != nil {
		return
	}

	return ipToAsn.refreshFromUrl(searchUrl)
}

func (ipToAsn *IpToAsn) Get(addr netip.Addr) (asn uint32, present bool) {
	return ipToAsn.asnMap.GetAddr(addr)
}

func (ipToAsn *IpToAsn) refreshFromUrl(url string) (err error) {
	var response *http.Response
	if response, err = http.Get(url); err != nil {
		return err
	}

	defer closeAndLogErrors("Error while closing HTTP response:", response.Body)

	var gzipReader *gzip.Reader
	if gzipReader, err = gzip.NewReader(response.Body); err != nil {
		return err
	}

	scanner := bufio.NewScanner(gzipReader)

	for scanner.Scan() {
		line := scanner.Text()
		prefix, asn, err := parseAsnLine(line)

		if err != nil {
			log.Println("Failed to parse CAIDA asn line")
			log.Printf("Failed line: \"%s\"\n", line)
			return err
		}

		if shouldIncludeAsnPrefix(prefix, asn) {
			// Remove any children from the range we are about to cover
			ipToAsn.asnMap.RemoveRange(prefix)

			// If we are still able to retrieve this prefix, we know it has already been covered by a higher prefix
			if found, ok := ipToAsn.asnMap.Get(prefix); !ok || found != asn {
				ipToAsn.asnMap.Set(prefix, asn)
			}
		}
	}

	return scanner.Err()
}

// shouldIncludeAsnPrefix checks that address meets the following conditions:
//   - The prefix corresponds to a public global unicast address
//   - The prefix is not too specific (greater than a /24 in IPv4 or a /48 in IPv6)
//   - The ASN is in the public globally assigned range
func shouldIncludeAsnPrefix(prefix netip.Prefix, asn uint32) bool {
	return prefix.Addr().IsGlobalUnicast() &&
		!prefix.Addr().Is4In6() &&
		!prefix.Addr().IsPrivate() &&
		!isPrefixTooSpecific(prefix) &&
		isPublicAsn(asn)
}

func isPrefixTooSpecific(prefix netip.Prefix) bool {
	if prefix.Addr().Is4() {
		return prefix.Bits() > 24
	} else {
		return prefix.Bits() > 48
	}
}

type rangeInclusive struct {
	min, max uint32
}

// reservedAsnRanges holds inclusive ranges of ASN values which have been reserved for various uses. These do not
// include unallocated ranges as they may be allocated in the future. These ranges are non-overlapping and are in
// ascending order.
var reservedAsnRanges = []rangeInclusive{
	{min: 0, max: 0},                   // Reserved (RFC7607)
	{min: 23456, max: 23456},           // Reserved for transition in ASN from 16-bit to 32-bit (RFC6793)
	{min: 64512, max: 65534},           // Reserved for private use (RFC6996)
	{min: 65535, max: 65535},           // Reserved (RFC7300)
	{min: 64496, max: 64511},           // Reserved for use in documentation and sample code (RFC5398)
	{min: 65536, max: 65551},           // Reserved for use in documentation and sample code (RFC5398)
	{min: 65552, max: 131071},          // Reserved
	{min: 4200000000, max: 4294967294}, // Reserved for private use (RFC6996)
	{min: 4294967295, max: 4294967295}, // Reserved (RFC7300)
}

func isPublicAsn(asn uint32) bool {
	for _, reservedRange := range reservedAsnRanges {
		if asn < reservedRange.min {
			// All the following ranges will be greater than the ASN, so we can skip them
			break
		}

		if asn >= reservedRange.min && asn <= reservedRange.max {
			return false
		}
	}

	return true
}

// Parses a line to extract info about the range of addresses and the ASN it refers to.
func parseAsnLine(input string) (prefix netip.Prefix, asn uint32, err error) {
	segments := strings.SplitN(input, "\t", 3)

	if len(segments) != 3 {
		err = errors.New("unexpected end of line: Not enough segments to parse")
		return
	}

	var addr netip.Addr
	if addr, err = netip.ParseAddr(segments[0]); err != nil {
		return
	}

	var parsedInt uint64
	if parsedInt, err = strconv.ParseUint(segments[1], 10, 8); err != nil {
		return
	}

	prefix = netip.PrefixFrom(addr, int(parsedInt))

	splitIndex := strings.IndexAny(segments[2], ",_")

	if splitIndex != -1 {
		segments[2] = segments[2][:splitIndex]
	}

	parsedInt, err = strconv.ParseUint(segments[2], 10, 32)
	asn = uint32(parsedInt)

	return
}

func latestCaidaData(searchDir string) (url string, err error) {
	var response *http.Response
	if response, err = http.Get(searchDir + Prefix2AsnCreationLog); err != nil {
		return
	}

	defer closeAndLogErrors("Error while closing HTTP response:", response.Body)

	scanner := bufio.NewScanner(response.Body)

	lastLine := ""
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			lastLine = line
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}

	// Get the latest entry in file
	lastSeparator := strings.LastIndexByte(lastLine, '\t')
	if lastSeparator == -1 {
		err = errors.New("unable to parse most recent pfx2asn file")
		return
	}

	url = searchDir + lastLine[lastSeparator+1:]
	return
}

func closeAndLogErrors(source string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Println(source, err)
	}
}
