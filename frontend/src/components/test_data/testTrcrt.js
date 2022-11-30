export const tData = {
    probeIp: "101.15.19.7 / 222.22.22.2",
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
        },
        {
            ip: "100.10.10.0",
            asn: "1111",
            averageRtt: 111.11,
            lastUsed: 1111111.11,
            averagePathLifespan: 111.11,
        },
        {
            ip: "222.22.22.2",
            asn: "2222",
            averageRtt: 222.22,
            lastUsed: 2222222.22,
            averagePathLifespan: 222.22,
        },
        {
            ip: "bac.12.22.a",
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },
        {
            ip: "112.22.11.7",
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },
        {
            ip: "abc.de.fg.h",
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },

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
        },
        {
            start: "123.45.67.8",
            end: "100.10.10.0",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: "222.22.22.2",
            end: "bac.12.22.a",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: "bac.12.22.a",
            end: "112.22.11.7",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: "bac.12.22.a",
            end: "abc.de.fg.h",
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        }
    ]
}