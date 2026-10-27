# Experiment D3 — Exponential Backoff with Full Jitter

Date: 2026-10-27 (UTC)

Code baseline commit: `999b669550e25af96409da4933a5bfb65eca5f8e` (with the D3 retry-policy change in the working tree)

## Fixed configuration

```text
offered rates = 500, 600, 800 req/s
workers       = 100
originals     = 2000 per offered rate
service-time  = 20ms
max-inflight  = 10
max retries   = 3
base backoff  = 40ms
backoff cap   = 200ms
jitter        = full jitter
Retry-After   = ignored
```

Each offered rate was run once, sequentially, against the local mock server.
The server accepts at most 10 concurrent requests. When all slots are busy, it
immediately returns `503 Service Unavailable` with `Retry-After: 1`.

A client worker retries only a `503`. For retry number `n`, it independently
samples the delay from `Uniform(0, min(200ms, 40ms * 2^(n-1)))`. Thus, the
three possible retry delays are sampled from `[0, 40ms)`, `[0, 80ms)`, and
`[0, 160ms)`. The server's `Retry-After` header is not parsed or waited for.

Each original can make one initial attempt plus up to three retries, for a
maximum of four HTTP attempts. A `200`, a non-`503` error, or a fourth `503`
is the final outcome.

## Results

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 500 | 2000 | 3502 | 1502 | 1.75× | 53.68% | 0.80 | 1880 | 1622 | 120 | 6.00% | 125.313697 ms |
| 600 | 2000 | 4651 | 2651 | 2.33× | 34.59% | 1.65 | 1609 | 3042 | 391 | 19.55% | 182.038094 ms |
| 800 | 2000 | 5622 | 3622 | 2.81× | 22.22% | 2.90 | 1249 | 4373 | 751 | 37.55% | 194.827814 ms |

The corresponding actual offered rates were `493.14`, `598.24`, and `795.64`
req/s.

No transport errors or response statuses other than `200` and `503` occurred,
so `HTTP attempts = 200 responses + 503 responses` for every row.

## Cross-policy comparison at 800 req/s

| Policy | Attempts | Failure | Failure % | Success P95 |
|---|---:|---:|---:|---:|
| D1 No retry | 2,000 | 821 | 41.05% | ~21ms |
| D2 Immediate | 5,099 | 784 | 39.20% | ~22ms |
| D3 Backoff+jitter | 5,622 | 751 | 37.55% | ~195ms |

Backoff with jitter reduces the final-failure percentage, improving system
resilience. The trade-off is substantially more HTTP attempts and much higher
success latency, so this configuration is not a good operating point.

## Observation

Full-jitter backoff modestly improved final success compared with the D2
immediate-retry run: final failures fell from `137` to `120` at 500 req/s,
from `396` to `391` at 600 req/s, and from `784` to `751` at 800 req/s. The
cost was substantially higher success latency: Success P95 rose from roughly
`21–22ms` in D2 to `125–195ms` here, because successful requests may include
one or more backoff delays.

This single D3 run did not reduce excess HTTP work versus D2. HTTP attempts
were `3502`, `4651`, and `5622`, compared with D2's `2887`, `4098`, and
`5099`; likewise, each D3 row produced more `503` responses. The jittered
delays spread individual retries, but their added residence time can keep the
fixed worker pool occupied and let original work accumulate. Under this exact
configuration and one-run sample, that trade-off yielded only small success
gains while increasing attempt amplification.
