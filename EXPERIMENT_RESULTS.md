# Worker count experiment

Date: 2026-08-29 (UTC)

- Endpoint: `https://jsonplaceholder.typicode.com/posts/{id}`
- Requests per run: 100
- Runs per worker count: 10
- Client timeout: 2 seconds
- Failed requests: 0 / 5,000
- Duration: the program's internal `Total time`

## Summary

| Workers | Min | Max | Average |
|---:|---:|---:|---:|
| 1 | 780.971 ms | 966.441 ms | 869.301 ms |
| 5 | 175.361 ms | 225.189 ms | 200.807 ms |
| 10 | 110.219 ms | 212.856 ms | 133.301 ms |
| 20 | 94.246 ms | 122.432 ms | 107.527 ms |
| 50 | 107.100 ms | 157.410 ms | 135.086 ms |

Throughput

1000 ┤                 ● 20
 900 ┤
 800 ┤          ● 10           ● 50
 700 ┤
 600 ┤
 500 ┤     ● 5
 400 ┤
 300 ┤
 200 ┤
 100 ┤ ● 1
     └──────────────────────────────
       1    5    10    20    50
                Workers

## Raw durations

All values are milliseconds.

| Workers | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | Run 6 | Run 7 | Run 8 | Run 9 | Run 10 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 966.441 | 856.686 | 930.463 | 903.176 | 946.989 | 780.971 | 815.930 | 799.018 | 820.304 | 873.036 |
| 5 | 212.081 | 225.189 | 209.613 | 188.358 | 210.097 | 204.738 | 175.361 | 184.769 | 216.347 | 181.517 |
| 10 | 125.379 | 129.922 | 135.394 | 113.256 | 128.743 | 123.937 | 133.671 | 110.219 | 212.856 | 119.634 |
| 20 | 94.246 | 112.787 | 122.432 | 106.588 | 118.868 | 97.995 | 96.246 | 101.410 | 112.918 | 111.777 |
| 50 | 143.328 | 146.610 | 123.805 | 146.642 | 107.100 | 128.060 | 123.397 | 121.987 | 157.410 | 152.516 |


## Takeaway

Too much concurrency can reduce throughput.
