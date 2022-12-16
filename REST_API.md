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
`POST /api/probes`

```js
const PostBody = {
    destinationIp: string,
    filterAsns: null | list[int],
    filterPrefix: null | string,
}

const Response = [
    {
        "id": int,
        "ipv4": string,
        "ipv6": string,
        "countryCode": string,
        "asn4": uint32,
        "asn6": uint32,
        "type": string,
        "coordinates": 
        [
            float64, //Longitude
            float64  //Latitude
        ],
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
            "asn": uint32,
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
            "asn": uint32, // Optional
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

## Measurement Tracking
### Start Tracking Measurement
`POST /api/measurement/start`
```js
const Request = {
    atlasMeasurementId: int,
    loadHistory: boolean,
    startLiveCollection: boolean,
}
```
The `loadHistory` field determines if the server will attempt to fetch historical data for the previous measurement period prior to doing live collection.

### Stop Tracking Measurement
`POST /api/measurement/stop`
```js
const Request = {
    atlasMeasurementId: int,
    dropStoredData: boolean,
}
```
### List Measurement
`GET /api/measurement/list`
```js
const Response = [
    {
        atlasMeasurementId: int,
        measurementPeriodStart: UnixTimestamp,
        measurementPeriodStop: UnixTimestamp,
        isLoadingHistory: boolean,
        usesLiveCollection: boolean,
    }
]
```
