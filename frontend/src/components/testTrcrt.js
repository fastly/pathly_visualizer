export const tData = {
    probeIp: "101.15.19.7",
    nodes: [
        {
            ip: "101.15.19.7",
            asn: "1234",
            averageRtt: 123.15,
            lastUsed: 1234567.89,
            averagePathLifespan: 123.45,
        },
        {
            ip: "123.45.67.8",
            asn: "4567",
            averageRtt: 111.11,
            lastUsed: 9876543.21,
            averagePathLifespan: 543.21,
        },
        {
            ip: "111.11.11.1",
            asn: "1111",
            averageRtt: 111.11,
            lastUsed: 1111111.11,
            averagePathLifespan: 111.11,
        }
    ],
    edges: [
        {
            start: "101.15.19.7",
            end: "123.45.67.8",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: "123.45.67.8",
            end: "111.11.11.1",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        }
    ]
}