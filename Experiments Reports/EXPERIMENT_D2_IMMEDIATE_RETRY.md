# Experiment D2 — Immediate Retry

Date: 2026-10-15 (UTC)

Code commit: `5b586fa7996875bf5d25da75cd8ae8b1de3b4218`

## Fixed configuration

```text
offered rates = 500, 600, 800 req/s
workers       = 100
originals     = 2000 per offered rate
service-time  = 20ms
max-inflight  = 10
max retries   = 3
backoff       = none
jitter        = none
Retry-After   = ignored
```

Each offered rate was run once, sequentially, against the local mock server.
The server accepts at most 10 concurrent requests. When all slots are busy, it
immediately returns `503 Service Unavailable` with `Retry-After: 1`.

A client worker retries a `503` immediately. Each original request can
make one initial attempt plus up to three retries, for a maximum of four HTTP
attempts. A `200`, a non-`503` error, or a fourth `503` is the final outcome.

## Results

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 500 | 2000 | 2887 | 887 | 1.44× | 64.53% | 0.48 | 1863 | 1024 | 137 | 6.85% | 21.470134 ms |
| 600 | 2000 | 4098 | 2098 | 2.05× | 39.14% | 1.31 | 1604 | 2494 | 396 | 19.80% | 21.414828 ms |
| 800 | 2000 | 5099 | 3099 | 2.55× | 23.85% | 2.55 | 1216 | 3883 | 784 | 39.20% | 22.165998 ms |

The corresponding actual offered rates were `499.64`, `599.67`, and `793.27
req/s`.

No transport errors or response statuses other than `200` and `503` occurred,
so `HTTP attempts = 200 responses + 503 responses` for every row.

## Observation

At and above `500 req/s`, immediate retries amplified offered load. At `800
req/s`, 2,000 originals created 5,099 HTTP attempts: 3,099 extra requests and
a 2.55× amplification factor. Those attempts generated 3,883 `503`
responses—far more than the 784 original requests that ultimately failed.

This demonstrates why `503 responses` is not `final failure`. For example, at
`600 req/s`, 2,494 attempts received `503`, but only 396 original requests
ended in failure; other requests saw one or more `503` responses and later
succeeded when a service slot became free.

Compared with D1's single no-retry run, D2 improved final success modestly at
the overloaded rates, but required substantially more HTTP work. At `800req/s`, D1 has failure rate of 41% while D2 39%, with far less HTTP requests (2000 vs 5099).

Success P95 remained near the configured 20ms service time because overload is
rejected immediately and successful retries can happen as soon as a slot opens.
It should not be interpreted as latency for failed requests; the metric includes
successful original requests only.
