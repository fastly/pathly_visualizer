# REST API Route
This file documents the REST API routes supported by our application, what they do, and what form the data is in.

All responses and POST requests must be encoded as JSON.

### Get Destinations
`GET /api/destinations`

```js
const Response = [
    {
        "ipv4": string,
        "ipv6": string,
    },
    // etc.
]
```

### Get probes
`GET /api/probes`

```js
const Response = [
    {
        "id": int,
        "ipv4": string,
        "ipv6": string,
        "countryCode": string,
        "asn4": uint32,
        "asn6": uint32,
        "location": GeoJSON,
    },
    // etc.
]
```

[GeoJson](https://geojson.org/)

### Raw Traceroute
`POST /api/traceroute/download`

```js
// POST body
const POSTBody = {
    "probeId": int,
    "destinationIp": string,
}
```

Response is included as an attachment. This attachment will be a json file in the ripeatlas format.

### Traceroute Data
`POST /api/traceroute/clean`

```js
const POSTBody = {
    "probeId": int,
    "destinationIp": string,
}
const Response = {
    "probeIp": string,
    "nodes": [
        {
            "ip": string,
            "asn": int,
            "averageRtt": float,
            "lastUsed": UnixTimestamp,
            "averagePathLifespan": float, // in seconds
        }, // etc...
    ],
    "edges": [
        {
            // start and end are the node ips
            "start": string,
            "end": string,
            "outboundCoverage": float,
            "totalTrafficCoverage": float,
            "lastUsed": UnixTimestamp,
        }
    ]
}
```
unix timestamps are int64s stored in seconds

### Traceroute Data Full
`POST /api/traceroute/full`

```js
const POSTBody = {
    "probeId": int,
    "destinationIp": string,
}

const NodeId = {
    "ip": string,
    "timeoutsSinceKnown": int, // zero on known node
}

const Response = {
    "probeIp": NodeId,
    "nodes": [
        {
            "id": NodeId,
            "asn": int, // Optional
            "averageRtt": float,
            "lastUsed": UnixTimestamp,
            "averagePathLifespan": float, // in seconds
        }, // etc...
    ],
    "edges": [
        {
            // start and end are the node ips
            "start": NodeId,
            "end": NodeId,
            "outboundCoverage": float,
            "totalTrafficCoverage": float,
            "lastUsed": UnixTimestamp,
        }
    ]
}
```

