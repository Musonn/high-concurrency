# 下一步：测量每个 HTTP Request 的 Latency

现在不要再只测 Total Time 了，我们开始测：

> **每一个 HTTP request 的 latency。**

例如：

```go
type Result struct {
	ID       int
	Duration time.Duration
	Err      error
}
```

然后：

```text
worker
  │
  ├── fetch post 1 ──→ Result{duration: 12ms}
  │
  ├── fetch post 7 ──→ Result{duration: 53ms}
  │
  └── fetch post 12 ─→ Result{duration: 15ms}
                         │
                         ▼
                    results channel
                         │
                         ▼
                       main
```

所以函数设计从：

```go
func fetch(id int)
```

变成类似：

```go
func fetch(id int) Result
```

worker 从：

```go
func worker(jobs <-chan int)
```

变成：

```go
func worker(
	jobs <-chan int,
	results chan<- Result,
)
```

注意这里特意写：

```go
jobs <-chan int
```

意思是：

> worker 只能 receive jobs

而：

```go
results chan<- Result
```

意思是：

> worker 只能 send results

这个就是 Go channel 很漂亮的一点。

---

最终要让程序打印：

```text
Workers: 20
Requests: 100
Success: 100
Failed: 0

Total: 107ms
Throughput: 934 req/s

Latency:
min:  7ms
avg: 18ms
p50: 15ms
p95: 31ms
p99: 62ms
max: 71ms
```

这里会出现一个初学者经常困惑的问题：

```text
Total time = 107ms

但

average request latency = 18ms
```

为什么：

```text
100 × 18ms ≠ 107ms
```

答案就是：

> **因为请求是 concurrent 的。**

而理解这个关系之后，我们就可以正式引入 **Little's Law**：

```text
Concurrency ≈ Throughput × Latency
```

这会把目前写的 worker pool 和真正的高并发系统性能理论连起来。

所以下一步自己尝试加入 **`results chan Result`**。先只统计每个 request 的 duration，不用急着算 p50/p95/p99；把收集 latency 的版本写出来后，再进行 code review。
