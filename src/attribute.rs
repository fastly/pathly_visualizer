//! BGP Attributes compatible with RFC4760
use num_derive::{FromPrimitive, ToPrimitive};

/// Written based on [IANA Address Family Numbers]. I don't know what most of these mean, but I
/// would like to have them available so they can be printed in a more google-able format if an
/// unexpected address family appears.
///
/// [IANA Address Family Numbers]: https://www.iana.org/assignments/address-family-numbers/address-family-numbers.xhtml#address-family-numbers-2
#[derive(Debug, FromPrimitive, ToPrimitive)]
#[repr(u16)]
pub enum AddressFamilyNumber {
    IPv4 = 1,
    IPv6 = 2,
    NSAP = 3,
    HDLC = 4,
    BNN = 5,
    IEEE802 = 6,
    E163 = 7,
    E164 = 8,
    F69 = 9,
    X121 = 10,
    IPX = 11,
    Appletalk = 12,
    DecnetIV = 13,
    BanyanVines = 14,
    E164WithNSAP = 15,
    DNS = 16,
    DistinguishedName = 17,
    ASNumber = 18,
    XtpViaIPv4 = 19,
    XtpViaIPv6 = 20,
    XtpNative = 21,
    FibreChannelPortName = 22,
    FibreChannelNodeName = 23,
    GWID = 24,
    AFI = 25,
    MplsTpSectionEndpointIdentifier = 26,
    MplsTpLspEndpointIdentifier = 27,
    MplsTpPseudowireEndpointIdentifier = 28,
    MTIPv4 = 29,
    MTIPv6 = 30,
    BgpSfc = 31,

    EigrpCommonServiceFamily = 16384,
    EigrpIPv4 = 16385,
    EigrpIPv6 = 16386,
    LCAF = 16387,
    BgpLs = 16388,
    MAC48bit = 16389,
    MAC64bit = 16390,
    OUI = 16391,
    MAC24 = 16392,
    MAC40 = 16393,
    IPv6_64 = 16394,
    RBridgePortID = 16395,
    TrillNickname = 16396,
    UUID = 16397,
    RoutingPolicyAFI = 16398,
    MplsNamespaces = 16399,
}

/// Subsequent Address Family Identifiers based on
/// https://www.iana.org/assignments/safi-namespace/safi-namespace.xhtml#safi-namespace-2
#[derive(Debug, FromPrimitive, ToPrimitive)]
#[repr(u16)]
pub enum SubsequentAddressFamilyIdentifiers {
    // Reserved = 0,
    NlriUnicastForwarding = 1,
    NlriMulticastForwarding = 2,
    // Reserved = 3,
    NlriMpls = 4,
    McastVpn = 5,
    /// Extremely shortened version of "Network Layer Reachability Information used for Dynamic
    /// Placement of Multi-Segment Pseudowires"
    NlriDynamicPseudowires = 6,
    EncapsulationSAFI = 7,
    McastVpls = 8,
    BgpSfc = 9,
    TunnelSafi = 64,
    VPLS = 65,
    BgpMdtSafi = 66,
    Bgp4over6Safi = 67,
    Bgp6over4Safi = 68,
    VpnDiscoveryInformation = 69,
    BgpEvpns = 70,
    BgpLs = 71,
    BgpLsVpn = 72,
    SrTePolicySafi = 73,
    SdWanCapabilities = 74,
    RoutingPolicySafi = 75,
    ClassfulTransportSafi = 76,
    TunneledTrafficFlowspec = 77,
    McastTree = 78,
    BgpDps = 79,
    BgpLsSpf = 80,
    BgpCar = 83,
    BgpVpnCar = 84,
    BgpMupSafi = 85,
    MplsVpnAddress = 128,
    /// "Multicast for BGP/MPLS IP Virtual Private Networks (VPNs)"
    BgpMulticastOrMplsVpns = 129,
    RouteTargetConstrains = 132,
    FlowSpecificationRules = 133,
    L3VpnFlowSpecificationRules = 134,
    VpnAutoDiscovery = 140,
}
