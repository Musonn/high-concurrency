# Experiment A — Backpressure

## Fixed configuration

```text
workers   = 10
queuesize = 100
requests  = 2000
service-time = 20ms
```

The producer uses blocking admission:

```go
jobs <- job
```

Each target rate was run once. The mock server was started with a fixed `20ms` service time.
Ideal pool capacity is `10 / 20ms = 500 req/s`. For this table, estimated pool capacity is `μ ≈ 10 / 20.9ms ≈ 478.47 req/s`, using a representative measured average service time. Here, `λ` is the target incoming rate and `ρ = λ / μ` is the target load relative to estimated capacity; values above 1 indicate overload, not worker utilization above 100%.

## Results

| Target rate (req/s) | λ target incoming (req/s) | ρ = λ/μ (μ≈478.47) | Actual throughput (req/s) | Service Avg | Queue Avg | Queue P95 | Total Avg | Total P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 300 | 300 | 0.627 | 299.03 | 20.724582 ms | 2.609 µs | 6.692 µs | 20.728676 ms | 21.290871 ms |
| 400 | 400 | 0.836 | 398.30 | 20.689412 ms | 3.906 µs | 5.821 µs | 20.694695 ms | 21.283036 ms |
| 500 | 500 | 1.045 | 474.92 | 20.905569 ms | 90.502644 ms | 167.978654 ms | 111.409286 ms | 188.81341 ms |
| 600 | 600 | 1.254 | 475.53 | 20.917101 ms | 182.434676 ms | 215.076359 ms | 203.352862 ms | 236.08999 ms |
| 800 | 800 | 1.672 | 472.91 | 21.048750 ms | 195.943858 ms | 218.966112 ms | 216.993594 ms | 240.067606 ms |

## Observation

At `300` and `400 req/s`, the worker pool keeps up with the producer and queue time remains negligible. At `500 req/s` and above, throughput reaches the practical capacity of approximately `478 req/s`, while queue time and total time increase substantially.

At `600` and `800 req/s`, the bounded queue fills and the blocking `jobs <- job` send applies backpressure. The producer is therefore slowed down instead of increasing the completed throughput beyond worker capacity.
