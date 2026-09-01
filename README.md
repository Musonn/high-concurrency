# Load Shedding with a Bounded Queue

This lesson moves the experiment from a public API to a deterministic localhost server. The server spends a fixed `20ms` on every request, so the worker pool's approximate processing capacity is known before the test begins.

You will write the load generator and use it to observe this progression:

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

Use `10` workers throughout the first experiment. If each worker completes one request every `20ms`, one worker can process approximately:

```text
1 second / 20ms = 50 requests/second
```

The entire pool can therefore process approximately:

```text
10 workers × 50 requests/second = 500 requests/second
```

This is an ideal estimate. HTTP and scheduler overhead mean the measured capacity may be slightly below `500 req/s`.

## Your Task

Adapt the root `main.go` into a rate-controlled load-shedding experiment. Keep the existing queue-time and service-time measurements where useful, but replace the public API request with:

```text
GET http://127.0.0.1:8080/work
```

Aim for this command-line interface:

```bash
go run . \
  -workers=10 \
  -queuesize=100 \
  -rate=800 \
  -duration=10s
```

Suggested defaults:

| Flag | Default | Meaning |
|---|---:|---|
| `-workers` | `10` | Number of request workers |
| `-queuesize` | `100` | Maximum number of waiting jobs |
| `-rate` | `400` | Offered requests per second |
| `-duration` | `10s` | Period during which new jobs are generated |
| `-url` | `http://127.0.0.1:8080/work` | Mock endpoint |

Validate that workers, queue size, rate, and duration are positive.

## Implementation Milestones

### 1. Replace the fixed request loop with a rate-controlled producer

The previous experiment created exactly 100 jobs as quickly as possible. This experiment must attempt work at a configured rate for a configured duration.

For the initial version, a `time.Ticker` interval can be calculated from the rate:

```text
interval = 1 second / target rate
```

On every tick, create one job and increment an `offered` counter. Stop generating jobs when the experiment duration expires.

The producer should do very little work so that it can keep up with the requested rate. Record the actual elapsed generation time and calculate the measured offered rate afterward.

### 2. Make admission non-blocking

A normal channel send blocks when the queue is full. That demonstrates backpressure, but it prevents the producer from maintaining the configured offered load.

For this experiment, attempt a non-blocking send:

```go
select {
case jobs <- job:
	// accepted
default:
	// rejected because the queue is full
}
```

Increment `accepted` only when the send succeeds. Increment `rejected` in the `default` branch. A rejected job must never reach a worker.

This admission decision is the load-shedding mechanism.

### 3. Preserve request timing boundaries

Each accepted job should retain its creation time. A worker should measure:

```text
job created
    │
    │ QueueTime
    ▼
worker receives job
    │
    │ ServiceTime
    ▼
HTTP request completes
```

Rejected jobs have no ServiceTime because they never enter the worker pool. If you want to measure rejection latency, record it separately rather than mixing it into request latency.

### 4. Implement a clean shutdown sequence

Use this ownership order:

1. The producer generates jobs for the configured duration.
2. The producer stops and closes `jobs`.
3. Workers finish accepted jobs that remain in the queue.
4. After all workers exit, close `results`.
5. The result collector finishes and prints the report.

Only the goroutine that owns a channel's send lifecycle should close that channel. Do not close `jobs` merely because the queue is full.

### 5. Avoid measurement races

Counters updated from multiple goroutines must be synchronized. Choose one of these approaches:

- Keep producer-owned counters in the producer and return them when it finishes.
- Send events to a single collector goroutine.
- Use `sync/atomic` for simple counters.

Do not read a counter while another goroutine may still be writing it unless synchronization establishes that the writer has finished.

## Required Metrics

Report at least:

- configured rate;
- measured offered rate;
- offered jobs;
- accepted jobs;
- rejected jobs;
- completed requests;
- failed HTTP requests;
- rejection percentage;
- completion throughput during the generation window;
- QueueTime average and P95;
- ServiceTime average and P95;
- TotalTime average and P95;
- queue length when generation stops.

Check these invariants after every run:

```text
offered = accepted + rejected
accepted = completed + failed
```

The second invariant should be checked after the accepted queue has drained. If HTTP failures are included in `completed`, define the counters differently and state that definition in the output.

## Measurement Window and Queue Drain

Stopping the producer does not mean the experiment is fully finished. Accepted jobs may still be waiting in the queue.

Record a snapshot when the generation window ends:

- completed during the window;
- rejected during the window;
- queue length at the end of the window.

Then drain the queue before exiting. Report drain time separately. Do not use generation time plus drain time as the denominator for the offered rate.

This distinction prevents a large queue from appearing to create worker capacity merely because it accepts extra jobs that are completed later.

## Experiment Matrix

Keep the following values fixed:

```ini
workers = 10
queuesize = 100
duration = 10s
service_time = 20ms
```

Change only the offered rate:

```bash
go run . -workers=10 -queuesize=100 -rate=400  -duration=10s
go run . -workers=10 -queuesize=100 -rate=500  -duration=10s
go run . -workers=10 -queuesize=100 -rate=800  -duration=10s
go run . -workers=10 -queuesize=100 -rate=1000 -duration=10s
```

Run each rate at least five times. Localhost removes public-network variability, but the Go scheduler and HTTP stack still introduce small variations.

## Expected Behavior

Treat these as hypotheses to test, not results to copy into the experiment record.

| Offered rate | Expected regime | Expected observation |
|---:|---|---|
| `400 req/s` | Below capacity | Little queueing and almost no rejection |
| `500 req/s` | Near capacity | Sensitive to overhead; queueing may begin |
| `800 req/s` | Overload | Queue fills; sustained rejection follows |
| `1000 req/s` | Heavy overload | Completion throughput remains near capacity while rejection increases |

At `800` or `1000 req/s`, the queue can temporarily accept more work than workers complete. That is burst absorption, not additional processing capacity. Once the bounded queue is full, rejecting excess jobs prevents waiting time and memory use from growing without limit.

## Results Template

Create a separate Markdown file after running the experiment. Record all five runs and summarize them with a table like this:

| Rate | Offered req/s | Completed req/s | Rejected | Rejection % | QueueTime avg | QueueTime P95 | Queue length at stop | Drain time |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 400 | | | | | | | | |
| 500 | | | | | | | | |
| 800 | | | | | | | | |
| 1000 | | | | | | | | |

Do not write the conclusion before inspecting the data. In particular, compare completion throughput with offered load, and separate work completed during the generation window from work completed while draining the queue.
