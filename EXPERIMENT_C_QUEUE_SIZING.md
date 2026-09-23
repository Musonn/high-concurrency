# Experiment C — Queue Size Trade-offs

Date: 2026-09-23 (UTC)

Code commit: `f9c7f71ab543b64846d892e894a5af7cf499bf1d`

## Fixed configuration

```text
rate         = 800 req/s
workers      = 10
service-time = 20ms
requests     = 2000
```

Each queue size was run once, sequentially, against the local mock server. The
client uses non-blocking admission: jobs that cannot enter the bounded queue
are rejected immediately. `Accepted rate` is the admission rate during the
producer window, rather than completed-request throughput.

## Results

| Queue size | Actual offered (req/s) | Accepted | Rejected | Reject % | Accepted rate (req/s) | Queue P95 | Total P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 0 | 796.96 | 1140 | 860 | 43.00% | 454.27 | 5.871 µs | 21.236442 ms |
| 10 | 797.61 | 1205 | 795 | 39.75% | 480.56 | 20.345296 ms | 41.594456 ms |
| 25 | 797.44 | 1227 | 773 | 38.65% | 489.23 | 55.737326 ms | 76.879492 ms |
| 40 | 796.19 | 1238 | 762 | 38.10% | 492.84 | 83.745976 ms | 104.878604 ms |
| 100 | 796.20 | 1305 | 695 | 34.75% | 519.52 | 208.677867 ms | 229.560542 ms |

## Observation

Increasing the queue from `0` to `100` admitted 165 additional jobs (8.25% of
all attempts) and reduced rejection by 8.25 percentage points. In exchange,
P95 total latency increased from about `21ms` to `230ms`—more than tenfold.

The measured queue P95 closely follows the simple steady-state estimate:

```text
QueueLatency ≈ QueueSize / ServiceRate
```

| Queue size | Estimated queue wait | Measured Queue P95 |
|---:|---:|---:|
| 10 | 20.8 ms | 20.3 ms |
| 25 | 52.1 ms | 55.7 ms |
| 40 | 83.3 ms | 83.7 ms |
| 100 | 208.3 ms | 208.7 ms |

Queue size is therefore an explicit latency-versus-rejection trade-off. For a
`100ms` P95 total-latency SLO, sizes `0`, `10`, and `25` met the target in this
run; size `40` narrowly missed it at about `105ms`, and size `100` missed it
substantially. Of the configurations tested, `25` was the largest that met the
SLO while accepting more work than the smaller queues.

One additional detail: `queueSize=0` accepted fewer jobs than `queueSize=10`.
A small buffer decouples the producer and workers enough to keep workers busy
despite scheduling variation, without permitting a long queueing delay.

## SLO-based queue sizing

1. Define the total-latency SLO.
2. Measure service latency at the relevant percentile.
3. Subtract service latency from the total-latency budget.
4. Estimate sustainable service rate.
5. Start with `QueueSize ≈ ServiceRate × QueueLatencyBudget`.
6. Leave a safety margin for service-time variation and bursts.
7. Validate the choice with load tests, including bursty traffic.
