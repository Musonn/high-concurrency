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
