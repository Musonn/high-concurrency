# 下一课我建议进入 **Backpressure**

现在你的 producer：
```bash
for id := 1; id <= 100; id++ {
    jobs <- id
}
```

刚好因为 `jobs` 是 unbuffered，所以已经**无意中实现了一种 backpressure**。

下一步我们可以把它改成：
```yaml
jobs := make(chan int, 10)
```

再到：
```yaml
jobs := make(chan int, 1000)
```

然后研究一个非常实际的问题：

> **为什么 queue 越大不一定越好？如果请求进来的速度是 2000 req/s，但 worker 只能处理 1000 req/s，会发生什么？**

这就是从 **Go concurrency** 正式进入 **high-concurrency system design** 的下一步。

## 下一步：测量 QueueTime 和 ServiceTime

我建议现在做一个很小但非常关键的升级。

不要急着上 HTTP server。

先让 `Job` 带上它的 arrival time。

现在：

```go
jobs chan int
```

改成：

```go
type Job struct {
    ID        int
    CreatedAt time.Time
}
```

然后：

```go
jobs := make(chan Job, *queueSize)
```

producer：

```go
jobs <- Job{
    ID:        id,
    CreatedAt: time.Now(),
}
```

worker 收到：

```go
func worker(
    client *http.Client,
    jobs <-chan Job,
    results chan<- Result,
) {
    for job := range jobs {
        // ...
    }
}
```

然后 Result 可以开始记录两个指标：

```go
type Result struct {
    ID int

    QueueTime   time.Duration
    ServiceTime time.Duration
    TotalTime   time.Duration

    Err error
}
```

worker 拿到 job 的瞬间：

```go
queueTime := time.Since(job.CreatedAt)
```

然后：

```text
Job created
    │
    │ QueueTime
    ▼
worker gets Job
    │
    │ ServiceTime
    ▼
complete
```

最后：

```text
TotalTime ≈ QueueTime + ServiceTime
```
