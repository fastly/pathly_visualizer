package asn

import (
	"bufio"
	"compress/gzip"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

type Example uint32

type IpToAsn struct {
	asnMap PrefixMap[uint32]
}

func CreateIpToAsn() (ipToAsn IpToAsn, err error) {
	ipToAsn.asnMap = MakePrefixMap[uint32]()
	err = ipToAsn.Refresh()
	return
}

func (ipToAsn *IpToAsn) Refresh() (err error) {
	var searchUrl string
	searchUrl, err = latestCaicdaData("https://publicdata.caida.org/datasets/routing/routeviews-prefix2as/")
	if err != nil {
		return
	}

	err = ipToAsn.refreshFromUrl(searchUrl)
	if err != nil {
		return
	}

	searchUrl, err = latestCaicdaData("https://publicdata.caida.org/datasets/routing/routeviews6-prefix2as/")
	if err != nil {
		return
	}

	err = ipToAsn.refreshFromUrl(searchUrl)
	return
}

func (ipToAsn *IpToAsn) Get(addr netip.Addr) (asn uint32, present bool) {
	return ipToAsn.asnMap.GetAddr(addr)
}

func (ipToAsn *IpToAsn) Length() int {
	return ipToAsn.asnMap.Length()
}

func (ipToAsn *IpToAsn) refreshFromUrl(url string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}

	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
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

		ipToAsn.asnMap.Set(prefix, asn)
	}

	return scanner.Err()
}

// Parses a line to extract info about the range of addresses and the ASN it refers to.
//
// Note: If an ip range is not routed (determined by checking if asn is 0), no other information will be parsed.
func parseAsnLine(input string) (prefix netip.Prefix, asn uint32, err error) {
	segments := strings.SplitN(input, "\t", 3)

	if len(segments) != 3 {
		err = errors.New("unexpected end of line: Not enough segments to parse")
		return
	}

	var addr netip.Addr
	addr, err = netip.ParseAddr(segments[0])
	if err != nil {
		return
	}

	var parsedInt uint64
	parsedInt, err = strconv.ParseUint(segments[1], 10, 8)
	if err != nil {
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

func latestCaicdaData(searchDir string) (url string, err error) {
	response, httpErr := http.Get(searchDir + "pfx2as-creation.log")
	if httpErr != nil {
		err = httpErr
		return
	}

	scanner := bufio.NewScanner(response.Body)

	lastLine := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lastLine = line
		}
	}

	err = scanner.Err()
	if err != nil {
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
