# Next Lesson — D4: Retry Budget

D3 showed that exponential backoff with full jitter can improve final success,
but unlimited retries can still create substantial HTTP amplification. D4 adds
a retry budget: retries are permitted only while a rate-limited allowance is
available.

## Goal

Test whether a bounded retry rate retains the resilience benefit of D3 while
reducing unnecessary work during overload.

## Fixed workload and server configuration

```text
offered rates = 500, 600, 800 req/s
workers       = 100
originals     = 2000 per offered rate
service-time  = 20ms
max-inflight  = 10
max retries   = 3
```

Run the local mock server with a fixed service time and a concurrency cap:

```bash
go run ./cmd/mockserver \
  -service-time=20ms \
  -max-inflight=10
```

When all ten service slots are busy, `/work` immediately returns `503 Service
Unavailable` and includes `Retry-After: 1`.

## D4 retry policy

D4 retains D3's retry-delay policy:

```text
base backoff = 40ms
backoff cap  = 200ms
jitter       = full jitter
Retry-After  = ignored
```

For retry number `n`, starting at 1, independently sample:

```text
delay_n ~ Uniform(0, min(200ms, 40ms × 2^(n - 1)))
```

Therefore the three possible retry delays are sampled from `[0, 40ms)`,
`[0, 80ms)`, and `[0, 160ms)`.

Add a retry budget of 10% of the offered original-request rate:

| Offered original rate | Retry allowance |
|---:|---:|
| 500 req/s | 50 retries/s |
| 600 req/s | 60 retries/s |
| 800 req/s | 80 retries/s |

Implement the allowance as a token bucket, shared by all workers for one
offered-rate run. The bucket should replenish at the retry-allowance rate and
hold at most one second of allowance. A retry consumes one token before its
backoff delay begins.

```text
503 received
│
├─ retry token available → consume token → backoff + full jitter → retry
│
└─ no retry token        → final failure
```

A `200`, a non-`503` error, exhaustion of three retries, or no available retry
token is a final outcome. D4 must not parse or wait for `Retry-After`.

## Measurements

Run D4 once at each offered rate: 500, 600, and 800 req/s. Record the standard
table used in D2 and D3, plus retry-budget denials if the implementation adds
that counter.

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|

Definitions:

```text
HTTP amplification = HTTP attempts / original requests
Efficiency         = final success / HTTP attempts
```

## Cross-policy comparison

At 800 req/s, compare the four policies:

| Policy | Failure % | HTTP amplification | Success P95 | Efficiency |
|---|---:|---:|---:|---:|
| D1 No retry |  |  |  |  |
| D2 Immediate retry |  |  |  |  |
| D3 Unlimited backoff + jitter |  |  |  |  |
| D4 Budgeted backoff + jitter |  |  |  |  |

The primary decision metrics are:

1. Failure percentage
2. HTTP amplification
3. Success P95
4. Efficiency

The desired D4 outcome is lower amplification and higher efficiency than D3,
without surrendering all of D3's final-success improvement over D1 and D2.
