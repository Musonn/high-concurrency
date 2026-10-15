# Little's Law: Experimental Results

Date: 2026-08-31 (UTC)

Code commit: `7a355447bc76fb1383d6f1f30482cf3283c9c79a`

## Results

Each worker count was run once with 100 requests. All 500 requests succeeded.

| Workers | Total time | Throughput (req/s) | Avg | P50 | P95 | P99 |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 1.093230025 s | 91.47 | 10.919422 ms | 10.170603 ms | 15.857939 ms | 19.704253 ms |
| 5 | 228.206351 ms | 438.20 | 11.223590 ms | 9.555110 ms | 20.848157 ms | 35.612564 ms |
| 10 | 119.379714 ms | 837.66 | 11.413482 ms | 8.419224 ms | 38.232703 ms | 38.342568 ms |
| 20 | 109.993306 ms | 909.15 | 20.698884 ms | 10.691815 ms | 60.119434 ms | 60.312415 ms |
| 50 | 94.792117 ms | 1054.94 | 43.694403 ms | 25.813200 ms | 72.476045 ms | 73.685952 ms |

## Little's Law

The experiment confirms Little's Law:

```text
L = lambda * W
```

where:

```text
L = average number of requests in the system
lambda = throughput, in requests per second
W = average time a request spends in the system
```

Using `Throughput × Avg Latency` for each run, with average latency converted from milliseconds to seconds, gives the estimated average concurrency:

| Workers | Throughput × Avg Latency | Estimated average L |
|---:|---:|---:|
| 1 | 91.47 x 0.010919 | 1.00 |
| 5 | 438.20 x 0.011224 | 4.92 |
| 10 | 837.66 x 0.011413 | 9.56 |
| 20 | 909.15 x 0.020699 | 18.81 |
| 50 | 1054.94 x 0.043694 | 46.09 |

## Conclusion

The measured results are consistent with Little's Law. As the number of workers increases, the average number of concurrent requests also increases. At higher concurrency, average latency rises, so throughput improvements begin to slow down.

`Estimated average L` is close to, but not necessarily equal to, the worker count because request completion times vary and this is a finite-batch experiment.
