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
		!prefix.Addr().IsPrivate() &&
		!isPrefixTooSpecific(prefix) &&
		!isPrivateAsn(asn)
}

func isPrefixTooSpecific(prefix netip.Prefix) bool {
	if prefix.Addr().Is4() {
		return prefix.Bits() > 24
	} else if prefix.Addr().Is4In6() {
		// IPv4-mapped IPv6 should not appear in this data, but check it regardless
		return prefix.Bits() > 120
	} else {
		return prefix.Bits() > 48
	}
}

func isPrivateAsn(asn uint32) bool {
	return asn >= 64512 && asn <= 65534
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
