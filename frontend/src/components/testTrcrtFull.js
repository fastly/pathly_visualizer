export const tDataFull = {
    probeIp: "101.15.19.7 / 222.22.22.2",
    nodes: [
        {
            id: {
                ip: "101.15.19.7",
                timeoutsSinceKnown: 0,
            },
            asn: "1234",
            averageRtt: 123.15,
            lastUsed: 1234567.89,
            averagePathLifespan: 123.45,
        },
        {
            id: {
                ip: "101.15.19.7",
                timeoutsSinceKnown: 1,
            },
            asn: "1234",
            averageRtt: 123.15,
            lastUsed: 1234567.89,
            averagePathLifespan: 123.45,
        },
        {
            id: {
                ip: "123.45.67.8",
                timeoutsSinceKnown: 0,
            },
            asn: "4567",
            averageRtt: 111.11,
            lastUsed: 9876543.21,
            averagePathLifespan: 543.21,
        },
        {
            id: {
                ip: "111.11.11.1",
                timeoutsSinceKnown: 0
            },
            asn: "1111",
            averageRtt: 111.11,
            lastUsed: 1111111.11,
            averagePathLifespan: 111.11,
        },
        {
            id: {
                ip: "100.10.10.0",
                timeoutsSinceKnown: 0,
            },
            asn: "1111",
            averageRtt: 111.11,
            lastUsed: 1111111.11,
            averagePathLifespan: 111.11,
        },
        {
            id: {
                ip: "222.22.22.2",
                timeoutsSinceKnown: 0,
            },
            asn: "2222",
            averageRtt: 222.22,
            lastUsed: 2222222.22,
            averagePathLifespan: 222.22,
        },
        {
            id: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 0,
            },
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },
        {
            id: {
                ip: "112.22.11.7",
                timeoutsSinceKnown: 0,
            },
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },
        {
            id: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 1,
            },
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },
        {
            id: {
                ip: "abc.de.fg.h",
                timeoutsSinceKnown: 0,
            },
            asn: "1244",
            averageRtt: 121.01,
            lastUsed: 2244400.01,
            averagePathLifespan: 444.11,
        },

    ],
    edges: [
        {
            start: {
                ip: "101.15.19.7",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "101.15.19.7",
                timeoutsSinceKnown: 1,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "101.15.19.7",
                timeoutsSinceKnown: 1,
            },
            end: {
                ip: "123.45.67.8",
                timeoutsSinceKnown: 0,    
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "123.45.67.8",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "111.11.11.1",
                timeoutsSinceKnown: 0,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "123.45.67.8",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "100.10.10.0",
                timeoutsSinceKnown: 0,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        // ###########
        // IPV6 
        // ###########
        {
            start: {
                ip: "222.22.22.2",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 0,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "112.22.11.7",
                timeoutsSinceKnown: 0,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 0,
            },
            end: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 1,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
        {
            start: {
                ip: "bac.12.22.a",
                timeoutsSinceKnown: 1,
            },
            end: {
                ip: "abc.de.fg.h",
                timeoutsSinceKnown: 0,
            },
            outboundCoverage: 10.0,
            totalTrafficCoverage: 20.0,
            lastUsed: 1111111.11,
        },
    ]
}