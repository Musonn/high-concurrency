# Load Shedding with a Bounded Queue

This lesson moves the experiment from a public API to a deterministic localhost server. The server spends a fixed `20ms` on every request, so the worker pool's approximate processing capacity is known before the test begins.

The load generator demonstrates this progression:

```text
offered load
     ↓
bounded queue
     ↓
worker capacity reached
     ↓
queue fills
     ↓
new work is rejected
```

The goal is to distinguish three related mechanisms:

- A queue absorbs a temporary burst.
- Backpressure slows or blocks the producer.
- Load shedding rejects excess work to protect the system.

## Mock Server

The mock server is already implemented in `cmd/mockserver`. Start it in one terminal:

```bash
go run ./cmd/mockserver
```

It listens on `127.0.0.1:8080` and exposes:

- `GET /work`: waits `20ms`, then returns `200 OK`.
- `GET /healthz`: returns `204 No Content` immediately.

You can verify it with:

```bash
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/work
```

The defaults can be changed when needed:

```bash
go run ./cmd/mockserver -addr=127.0.0.1:8081 -service-time=50ms
```

Keep the default `20ms` service time for this lesson.

## Theoretical Capacity

Use `10` workers throughout Experiment B. If each worker completes one request every `20ms`, one worker can process approximately:

```text
1 second / 20ms = 50 requests/second
```

The entire pool can therefore process approximately:

```text
10 workers × 50 requests/second = 500 requests/second
```

This is an ideal estimate. HTTP and scheduler overhead mean the measured capacity may be slightly below `500 req/s`.

## Experiment B: Load Shedding

Relative to Experiment A, the admission policy changes from a blocking channel send to a non-blocking send. Keep the worker count, queue size, service time, request count, and ticker-based producer fixed.

```ini
workers     = 10
queueSize   = 100
serviceTime = 20ms
requests    = 2000
```

The client sends HTTP requests to `http://127.0.0.1:8080/work` with a `2s` timeout. The endpoint and request count are fixed in `main.go`; there are no `-url`, `-duration`, or `-requests` flags.

## Client Configuration

```bash
go run . -workers=10 -queuesize=100 -rate=400
```

| Flag | Default | Meaning |
|---|---:|---|
| `-workers` | `10` | Number of request workers |
| `-queuesize` | `100` | Maximum number of waiting jobs |
| `-rate` | `400` | Target admission attempts per second |

The current client does not validate flag values. For this experiment, use the fixed configuration and rates below. Invalid values such as a zero rate or negative queue size can panic; a nonpositive worker count provides no processing capacity.

Changing the mock server's address also requires changing the client's hardcoded endpoint.

## Producer and Admission Policy

The producer creates one job per received ticker event, using an interval of `time.Second / rate`, until it has attempted exactly 2,000 jobs. Ticker scheduling can affect the actual arrival rate; the configured rate is a target.

Admission is non-blocking:

```go
select {
case jobs <- job:
    accepted++
default:
    rejected++
}
```

A successful send accepts the job. If the queue cannot accept it immediately, the producer rejects it and continues. Rejected jobs never reach a worker or make an HTTP request.

Experiment A's blocking `jobs <- job` slows the producer when the queue fills. Experiment B rejects excess work while continuing to attempt admission on ticker events.

## Timing and Shutdown

Each accepted job retains its creation time:

- Queue time runs from job creation until a worker receives it.
- Total time runs from job creation until the HTTP attempt finishes, including queue time.

After all 2,000 admission attempts, the producer closes `jobs`. Workers drain accepted jobs, then `results` closes after all workers exit. The collector finishes before printing the report.

Admission counters belong to the producer and are returned through a channel. The collector owns the latency samples, avoiding unsynchronized shared counter access.

## Output Metrics

The client prints only these eight metrics:

| Metric | Definition |
|---|---|
| Offered rate | Configured `-rate` target in requests/sec; not a measured arrival rate |
| Actual offered rate | All 2,000 admission attempts divided by the producer's elapsed time |
| Accepted | Jobs successfully sent to the worker queue |
| Rejected | Jobs rejected immediately at admission |
| Reject % | `rejected / 2000 × 100` |
| Accepted rate | Accepted jobs divided by the producer's elapsed time |
| Queue P95 | 95th percentile of queue time for accepted jobs |
| Total P95 | 95th percentile of total time for accepted jobs |

Actual offered rate and accepted rate use the generation window, from before the first ticker wait through the final admission attempt. Rejected requests are excluded from both latency metrics. Every accepted job contributes one latency sample, including if its HTTP attempt fails.

Percentiles use the nearest-rank method; empty samples return zero. Durations are printed with Go duration units, such as `µs` or `ms`.

Admission accounting guarantees `accepted + rejected = 2000`. After drain, every accepted job has one latency sample. The client does not print separate success/failure counts or perform explicit accounting assertions.

## Experiment Matrix

Start the mock server in one terminal:

```bash
go run ./cmd/mockserver -service-time=20ms
```

In another terminal, run these commands sequentially when ready to collect results:

```bash
go run . -workers=10 -queuesize=100 -rate=300
go run . -workers=10 -queuesize=100 -rate=400
go run . -workers=10 -queuesize=100 -rate=500
go run . -workers=10 -queuesize=100 -rate=600
go run . -workers=10 -queuesize=100 -rate=800
```

Each invocation attempts 2,000 jobs and then drains accepted work. Record the eight output metrics in this table:

| Offered rate (req/s) | Actual offered (req/s) | Accepted | Rejected | Reject % | Accepted rate (req/s) | Queue P95 | Total P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 300 | | | | | | | |
| 400 | | | | | | | |
| 500 | | | | | | | |
| 600 | | | | | | | |
| 800 | | | | | | | |

## Expected Behavior

These are hypotheses to test, not recorded results.

| Offered rate (req/s) | Expected behavior |
|---:|---|
| 300 | Little queueing and little or no rejection |
| 400 | Little queueing and little or no rejection |
| 500 | Near ideal capacity; overhead may cause queue buildup |
| 600 | Overload; the queue may fill and trigger rejection |
| 800 | Heavier overload; more rejection while throughput stays near worker capacity |

The 100-job queue can temporarily absorb excess arrivals. With only 2,000 attempts, a run near capacity may finish admission without rejection despite accumulating queue time. Zero rejection does not prove that the offered rate is sustainable indefinitely.

Load shedding does not increase processing capacity or remove waiting time for accepted jobs. It rejects excess work once the bounded queue fills. Compare the completed throughput, rejection percentage, and latency percentiles together.
