# Experiment D4 — Retry Budget

Date: 2026-11-15 (UTC)

Code commit: `f986452` (Implement D4 retry budget)

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
retry budget  = 10% of offered original rate, shared per run
bucket cap    = one second of retry allowance
```

The retry allowance is 50, 60, and 80 retries per second at the three offered
rates. The shared token bucket starts full, refills continuously at that rate,
and has capacity equal to one second of allowance. Before each otherwise
eligible retry, a worker consumes one token; if none is available, that `503`
is a final failure. A consumed token is followed by D3's independently sampled
full-jitter delay: `[0, 40ms)`, `[0, 80ms)`, or `[0, 160ms)`.

Each rate was run once against the local mock server. No transport errors or
response statuses other than `200` and `503` occurred.

## Results

| Offered (req/s) | Original | HTTP attempts | Retries | Budget denials | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 500 | 2000 | 2247 | 247 | 174 | 1.12× | 81.13% | 0.14 | 1823 | 424 | 177 | 8.85% | 41.857909 ms |
| 600 | 2000 | 2258 | 258 | 449 | 1.13× | 68.42% | 0.17 | 1545 | 713 | 455 | 22.75% | 43.360387 ms |
| 800 | 2000 | 2278 | 278 | 804 | 1.14× | 52.15% | 0.23 | 1188 | 1090 | 812 | 40.60% | 42.145776 ms |

The corresponding actual offered rates were `500.03`, `599.72`, and `799.13`
req/s. HTTP amplification is `HTTP attempts / original requests`; efficiency
is `final success / HTTP attempts`.

## Cross-policy comparison at 800 req/s

| Policy | Failure % | HTTP amplification | Success P95 | Efficiency |
|---|---:|---:|---:|---:|
| D1 No retry | 41.05% | 1.00× | 21.234634 ms | 58.95% |
| D2 Immediate retry | 39.20% | 2.55× | 22.165998 ms | 23.85% |
| D3 Unlimited backoff + jitter | 37.55% | 2.81× | 194.827814 ms | 22.22% |
| D4 Budgeted backoff + jitter | 40.60% | 1.14× | 42.145776 ms | 52.15% |

## Observation

At 800 req/s, D4 reduced amplification substantially versus D3 (1.14× versus
2.81×) and restored efficiency to 52.15% (from 22.22%). It also retained a
small final-success improvement over D1: failure fell from 41.05% to 40.60%.
In this single run, however, it did not retain D3's improvement over D2:
D4's 40.60% failure rate was higher than D2's 39.20%. The 10% budget therefore
provides a strong bound on excess work, but this particular allowance is too
restrictive to achieve every desired resilience comparison at 800 req/s.
