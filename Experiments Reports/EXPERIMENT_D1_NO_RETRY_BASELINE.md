# Experiment D1 — No-Retry Baseline

Date: 2026-10-15 (UTC)

Code commit: `1c2e8d33315948bf8397d3492f4ae786e8aff4ab`

## Fixed configuration

```text
offered rates = 300, 400, 500, 600, 800 req/s
workers       = 100
requests      = 2000 per offered rate
service-time  = 20ms
max-inflight  = 10
retries       = disabled
```

Each offered rate was run once, sequentially, against the local mock server.
The server accepts at most 10 concurrent requests. Once all slots are busy it
immediately returns `503 Service Unavailable` with `Retry-After: 1`.

The client buffers the original requests locally so all 2,000 requests receive
an initial HTTP attempt. It does not retry `503` responses in this baseline.
`P95 final latency` measures the time from original-request creation to the
completion of its final attempt. With retries disabled, that is the one and
only attempt.

## Results

| Offered rate (req/s) | Actual offered (req/s) | Original requests | HTTP attempts | 200 responses | 503 responses | Final failures | Failure % | Retries | P95 final latency |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 300 | 299.67 | 2000 | 2000 | 2000 | 0 | 0 | 0.00% | 0 | 21.301498 ms |
| 400 | 399.87 | 2000 | 2000 | 2000 | 0 | 0| 0.00% | 0 | 21.276195 ms |
| 500 | 499.12 | 2000 | 2000 | 1817 | 183 | 183 | 9.15% | 0 | 21.281400 ms |
| 600 | 598.66 | 2000 | 2000 | 1540 | 460 | 460 | 23.00% | 0 | 21.258580 ms |
| 800 | 796.93 | 2000 | 2000 | 1179 | 821 | 821 | 41.05% | 0 | 21.234634 ms |

## Observation

At `300` and `400 req/s`, every original request completed successfully. The
P95 final latency stayed at about `21.3ms`, which is the configured `20ms`
service time plus local HTTP and scheduler overhead.

At approximately `500 req/s`, the client reached the service's nominal
capacity of `10 / 20ms = 500 req/s`. The concurrency limit admits requests
only while a slot is free, so scheduling variation already produced 183
immediate `503` responses (9.15%) at the 500 req/s target.

Above capacity, the server shed progressively more work: final failures rose
to 23.00% at `600 req/s` and 41.05% at `800 req/s`. Since the client does not
retry, every `503` is also a final failure, and total HTTP attempts remain
exactly equal to the 2,000 original requests at every rate.

P95 final latency remains essentially flat while failure rises. This is the
intended effect of server-side load shedding: overloaded requests fail quickly
instead of accumulating in a queue and increasing latency for admitted work.
