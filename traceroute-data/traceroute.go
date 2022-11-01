package traceroute_data

type Traceroute struct {
	Fw        int    //Firmware version used by the probe
	Af        int    //address family, 4 or 6
	Src_addr  string //source address used by probe
	Dst_addr  string //IP address of the destination
	Dst_name  string // name of the destination
	Timestamp int    //Unix timestamp for start of measurement
	Endtime   int    // Unix timestamp for end of measurement
	Lts       int    //last time synchronised in seconds. If -1 then probe does not know whether it is in sync
	Prb_id    int    //source probe ID
	Proto     string //"UDP", "ICMP", or "TCP"
	Result    []Hop  // List of Hop Structs
	Size      int    //  packet size
}

type Hop struct {
	Hop    int      // hop number
	Error  string   //when an error occurs trying to send a packet. In that case there will not be a result structure
	Result []Result //Array of Result Struct
}

type Result struct {
	//Case: Timeout
	X string //"x" -- "*"
	//Case Reply
	Err  ErrWrapper // error ICMP: "N" (network unreachable,), "H" (destination unreachable), "A" (administratively prohibited), "P" (protocol unreachable), "p" (port unreachable) "h" (string) Unrecognized error codes are represented as integers
	From string     // IPv4 or IPv6 source address in reply
	Late int        // number of packets a reply is late, in this case rtt is not present
	Rtt  float64    // round-trip-time of reply, not present when the response is late
	Size int        //size of reply
	Ttl  int        //time-to-live in reply
}

type ErrWrapper string

func (w *ErrWrapper) UnmarshalJSON(data []byte) (err error) {

	str := string(data)
	*w = ErrWrapper(str)
	return nil
}
