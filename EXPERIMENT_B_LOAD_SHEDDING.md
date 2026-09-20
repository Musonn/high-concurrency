# Experiment B — Load Shedding

Code commit: `3a9d9470d1c7b0f4177f9250c470ad34cee36c65`

## Fixed configuration

```text
workers      = 10
queuesize    = 100
requests     = 2000
service-time = 20ms
```

Relative to Experiment A, only the admission policy changes. The producer attempts a non-blocking send and rejects a job immediately when the bounded queue cannot accept it:

```go
select {
case jobs <- job:
	accepted++
default:
	rejected++
}
```

Each target rate was run once, sequentially, against the local mock server started with `-service-time=20ms`. The nominal worker-pool capacity is `10 / 20ms = 500 req/s`; HTTP and scheduler overhead make the practical completion capacity slightly lower.

`Actual offered rate` is the 2,000 admission attempts divided by the producer's elapsed time. `Accepted rate` is accepted jobs divided by that same producer window. It is an admission rate, not completion throughput: accepted jobs may still be waiting or executing when generation stops. Queue and total P95 include accepted jobs only; rejected jobs never enter the worker pool.

## Results

| Offered rate (req/s) | Actual offered (req/s) | Accepted | Rejected | Reject % | Accepted rate (req/s) | Queue P95 | Total P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 300 | 299.98 | 2000 | 0 | 0.00% | 299.98 | 5.971 µs | 21.257160 ms |
| 400 | 399.76 | 2000 | 0 | 0.00% | 399.76 | 5.981 µs | 21.268319 ms |
| 500 | 499.69 | 2000 | 0 | 0.00% | 499.69 | 165.737806 ms | 186.614401 ms |
| 600 | 599.27 | 1699 | 301 | 15.05% | 509.08 | 207.712034 ms | 228.683477 ms |
| 800 | 794.04 | 1298 | 702 | 35.10% | 515.33 | 215.742718 ms | 238.242309 ms |

## Observation

At `300` and `400 req/s`, every attempted job was accepted and Queue P95 was about `6 µs`. At `500 req/s`, all jobs were still admitted in this finite run, but Queue P95 rose to about `166 ms`; the bounded queue temporarily absorbed the accumulating work.

At `600` and `800 req/s`, the queue filled and the non-blocking admission policy rejected excess work. Rejection rose from `15.05%` to `35.10%` as the offered rate increased. Queue P95 and Total P95 remained bounded by the 100-job queue, at approximately `208–216 ms` and `229–238 ms` respectively.

Experiment A blocks the producer once the queue is full, while Experiment B continues its ticker-driven admission attempts and rejects work that cannot enter the queue. The higher accepted rates at `600` and `800 req/s` do not imply higher completion capacity: they include jobs admitted into the queue during the generation window and completed while draining afterward.

## Conclusion

Latency stops degrading indefinitely and is bounded by the queue capacity. However, this is not free lunch. A larger queue increases latency, while a smaller queue increases rejection rate. 