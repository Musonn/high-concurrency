# Experiment C: How Should You Size a Queue?

Now we can answer a practical production question:

> Why is `queueSize=100`? Why not 10, 40, or 1,000?

The answer should not be:

```text
100 seems about right.
```

Instead, work backward from a **latency SLO**.

## Size the Queue From the Latency Budget

Suppose the requirement is:

```text
P95 total latency <= 100ms
```

If measured P95 service time is approximately:

```text
21ms
```

then the queue has only the remaining latency budget:

```text
100ms - 21ms = 79ms
```

At an estimated processing capacity of `475 req/s`, the number of jobs that may wait is approximately:

```text
475 req/s * 0.079s = 37.5 jobs
```

A reasonable queue size to try first is therefore:

```text
35-40 jobs
```

not 100.

With a stricter SLO:

```text
P95 total latency <= 50ms
```

the queue budget becomes:

```text
50ms - 21ms = 29ms
```

and the estimated queue size becomes:

```text
475 req/s * 0.029s = 13.8 jobs
```

In that case, the queue can hold only about:

```text
14 jobs
```

This gives a useful rule of thumb:

```text
QueueSize ≈ ServiceRate * QueueLatencyBudget
```

where:

```text
QueueLatencyBudget = TotalLatencySLO - ServiceLatency
```

Use a P95 service-time measurement when sizing a queue for a P95 total-latency SLO. This is a starting estimate, not a guarantee: scheduler overhead, service-time variation, and burstiness still need to be tested.

## Experiment C: Queue Size Trade-offs

Keep the overload scenario fixed:

```ini
rate        = 800/s
workers     = 10
serviceTime = 20ms
requests    = 2000
```

Change only the queue size:

```ini
queueSize = 0
queueSize = 10
queueSize = 25
queueSize = 40
queueSize = 100
```

Start the mock server in one terminal:

```bash
go run ./cmd/mockserver -service-time=20ms
```

Then run the client for each queue size:

```bash
go run . -workers=10 -queuesize=0 -rate=800
go run . -workers=10 -queuesize=10 -rate=800
go run . -workers=10 -queuesize=25 -rate=800
go run . -workers=10 -queuesize=40 -rate=800
go run . -workers=10 -queuesize=100 -rate=800
```

Record the results:

| Queue size | Accepted | Reject % | Queue P95 | Total P95 |
|---:|---:|---:|---:|---:|
| 0 | | | | |
| 10 | | | | |
| 25 | | | | |
| 40 | | | | |
| 100 | | | | |

## Expected Trade-off

```text
smaller queue
    ↓
higher rejection rate
lower latency

larger queue
    ↓
lower rejection rate
higher latency
```

There is no free configuration that avoids rejection, avoids queueing, and still exceeds system capacity. You must choose the trade-off:

> Are you more willing to sacrifice latency, or acceptance rate?

After Experiment C, the next topics are latency SLOs, queue sizing, `429`/`503` responses, and why retries can trigger a retry storm.
