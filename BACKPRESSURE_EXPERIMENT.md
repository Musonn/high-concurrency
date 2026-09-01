# Backpressure Experiment

## Objective

Observe how changing the jobs queue size affects queue time, total execution time, and throughput while keeping the worker count and request count fixed.

## Experiment Setup

- Workers: `10`
- Requests: `100`
- Queue sizes: `0`, `1`, `10`, `50`, `100`
- Repetitions: `5` runs for each queue size
- API: `https://jsonplaceholder.typicode.com/posts/{id}`
- HTTP client timeout: `2s`
- Total runs: `25`

The request count is fixed by `requestCount = 100` in the program.

## Commands

```bash
go run . -workers=10 -queuesize=0
go run . -workers=10 -queuesize=1
go run . -workers=10 -queuesize=10
go run . -workers=10 -queuesize=50
go run . -workers=10 -queuesize=100
```

Each command was run five times. The experiments were executed sequentially in queue-size order rather than in parallel, to avoid the experiments competing with one another for access to the public API.

## Aggregate Results

The following values are the averages across five runs for each queue size. The throughput column also shows the range across the five runs.

| Queue size | Successful | Failed | Average throughput (req/s) | Throughput range (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime | TotalTime P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 0 | 100 | 0 | 564.97 | 302.61–737.12 | 1.79 ms | 4.73 ms | 20.50 ms | 48.36 ms |
| 1 | 100 | 0 | 664.07 | 653.92–677.01 | 2.73 ms | 6.23 ms | 17.05 ms | 44.80 ms |
| 10 | 100 | 0 | 651.96 | 488.78–762.52 | 15.42 ms | 47.21 ms | 30.48 ms | 57.31 ms |
| 50 | 100 | 0 | 501.93 | 291.03–642.11 | 78.53 ms | 118.97 ms | 99.34 ms | 137.70 ms |
| 100 | 100 | 0 | 661.07 | 600.07–706.63 | 80.96 ms | 135.43 ms | 95.56 ms | 145.68 ms |

## Per-Run Results

### Queue size = 0

| Run | Throughput (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime |
|---:|---:|---:|---:|---:|
| 1 | 552.58 | 1.66 ms | 4.74 ms | 19.11 ms |
| 2 | 302.61 | 2.98 ms | 9.15 ms | 34.58 ms |
| 3 | 737.12 | 1.25 ms | 2.96 ms | 14.34 ms |
| 4 | 685.08 | 1.35 ms | 3.00 ms | 15.50 ms |
| 5 | 547.44 | 1.70 ms | 3.79 ms | 18.98 ms |

### Queue size = 1

| Run | Throughput (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime |
|---:|---:|---:|---:|---:|
| 1 | 664.66 | 2.76 ms | 5.50 ms | 17.27 ms |
| 2 | 664.33 | 2.70 ms | 6.54 ms | 16.88 ms |
| 3 | 660.43 | 2.60 ms | 7.47 ms | 16.28 ms |
| 4 | 677.01 | 2.76 ms | 5.37 ms | 17.22 ms |
| 5 | 653.92 | 2.84 ms | 6.25 ms | 17.61 ms |

### Queue size = 10

| Run | Throughput (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime |
|---:|---:|---:|---:|---:|
| 1 | 626.88 | 14.80 ms | 41.48 ms | 29.56 ms |
| 2 | 758.67 | 12.89 ms | 40.93 ms | 25.60 ms |
| 3 | 622.93 | 15.58 ms | 54.46 ms | 30.85 ms |
| 4 | 762.52 | 12.99 ms | 38.86 ms | 25.60 ms |
| 5 | 488.78 | 20.85 ms | 60.32 ms | 40.77 ms |

### Queue size = 50

| Run | Throughput (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime |
|---:|---:|---:|---:|---:|
| 1 | 291.03 | 128.40 ms | 199.67 ms | 161.80 ms |
| 2 | 635.19 | 52.48 ms | 82.76 ms | 66.90 ms |
| 3 | 490.07 | 72.49 ms | 99.88 ms | 92.25 ms |
| 4 | 642.11 | 62.38 ms | 103.17 ms | 77.51 ms |
| 5 | 451.24 | 76.88 ms | 109.37 ms | 98.23 ms |

### Queue size = 100

| Run | Throughput (req/s) | Average QueueTime | QueueTime P95 | Average TotalTime |
|---:|---:|---:|---:|---:|
| 1 | 693.87 | 73.02 ms | 123.04 ms | 86.62 ms |
| 2 | 633.52 | 90.09 ms | 142.86 ms | 105.14 ms |
| 3 | 671.28 | 80.00 ms | 134.52 ms | 94.52 ms |
| 4 | 600.07 | 84.47 ms | 151.13 ms | 100.75 ms |
| 5 | 706.63 | 77.21 ms | 125.58 ms | 90.76 ms |

## Recorded Observations

- QueueTime increased noticeably when the queue size changed from `0/1` to `10`.
- With queue sizes of `50` and `100`, average QueueTime was approximately `80 ms`, and P95 exceeded `100 ms`.
- Throughput did not increase monotonically with queue size.

## Conclusion

Queue absorbs bursts; it does not create capacity.

                    incoming work
                         │
                         ▼
                ┌─────────────────┐
                │ bounded queue   │
                └────────┬────────┘
                         │
                   worker pool
                         │
                         ▼
                    downstream

When arrival <= capacity, queue_time ≈ 0, latency is low; when arrival > capacity, queue absorbs burst, consuming memory and increasing latency. A bounded queue limits memory usage, but once it becomes full, backpressure must occur: the producer blocks, or new requests are rejected or dropped. An unbounded queue can continue accepting work, but queued requests consume more memory and may eventually cause resource exhaustion.

## Code Commit

The code and metrics implementation used for this experiment was committed in:

- Commit: `7c42cb8`
- Message: `Add queue backpressure metrics`
