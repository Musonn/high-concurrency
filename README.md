# High Concurrency

可以，而且我建议我们就这么学。

## 快速开始

环境要求：Go 1.26 或更高版本。

```bash
go run .
```

当前代码实现了 **Step 0**：从 JSONPlaceholder 获取并解析一篇 post。

最适合第一阶段的是 **JSONPlaceholder + httpbin**：

- **JSONPlaceholder**：真实的 REST/JSON 结构，有 `/posts/1` 到 `/posts/100`，很适合练习批量并发请求。[JSONPlaceholder 官方指南](https://jsonplaceholder.typicode.com/guide/)
- **httpbin**：专门用来测试 HTTP client，可以人为制造状态码、redirect、stream 等情况，非常适合后面练习 timeout、retry 和 error handling。[httpbin](https://httpbin.org/)
- **Open-Meteo**：后期可以做更真实的小项目，例如并发获取几十个城市的天气；目前非商业使用不需要 API key。[Open-Meteo](https://open-meteo.com/en/about)

不过有一个原则：

> **公共 API 用来学习并发，不要拿来真正做压力测试。**

比如 10、20、50 个并发请求可以；不要直接用 `100000 goroutines` 去轰别人的服务器。真正练习 10k/100k concurrency 时，我们会自己在 localhost 启动一个 Go server，或者在本地运行 httpbin。

## 第一课：Concurrent REST API Fetcher

目标：

```text
JSONPlaceholder

/posts/1
/posts/2
/posts/3
...
/posts/100
```

顺序请求的 naive 版本：

```text
request 1  ──────► response
request 2          ──────► response
request 3                  ──────► response
...
```

然后改成：

```text
             jobs
              │
              ▼
        ┌───────────┐
        │  channel  │
        └─────┬─────┘
              │
       ┌──────┼──────┐
       ▼      ▼      ▼
    worker  worker  worker
       │      │      │
       └──────┼──────┘
              ▼
       REST API requests
              │
              ▼
          results
```

第一版要求非常简单：

```text
100 requests
5 workers

统计：
成功数量
失败数量
总耗时
每个 request latency
```

## Step 0：先不要并发

先写一个普通 HTTP request：

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func main() {
	resp, err := http.Get(
		"https://jsonplaceholder.typicode.com/posts/1",
	)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var post Post

	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", post)
}
```

这个 endpoint 是官方文档推荐的基本用法。先确保这个版本能够跑通。

## 开始真正的并发

第一次升级不是：

```go
for i := 1; i <= 100; i++ {
	go fetch(i)
}
```

虽然这个很诱人。😂

而是直接学习更合理的方式：

```go
jobs := make(chan int)

for i := 0; i < 5; i++ {
	go worker(jobs)
}

for id := 1; id <= 100; id++ {
	jobs <- id
}

close(jobs)
```

worker：

```go
func worker(jobs <-chan int) {
	for id := range jobs {
		fetch(id)
	}
}
```

你马上就会碰到第一个问题：

```text
main goroutine
    │
    ├── 创建 workers
    │
    ├── 提交 jobs
    │
    └── main 结束
             ↓
        程序退出 💀

worker 可能还没做完
```

于是自然引出：

```go
sync.WaitGroup
```

这比先讲半小时 WaitGroup 理论有效得多。

## 逐级升级

我们可以把同一个程序一直写下去：

```text
Level 1
Sequential HTTP requests
        ↓
Level 2
goroutine
        ↓
Level 3
WaitGroup
        ↓
Level 4
worker pool
        ↓
Level 5
results channel
        ↓
Level 6
context timeout
        ↓
Level 7
retry
        ↓
Level 8
rate limiter
        ↓
Level 9
backpressure
        ↓
Level 10
benchmark / pprof
```

例如到了 timeout，我们就换成 httpbin：

```text
/delay/*
/status/*
```

或者使用其他可控 endpoint，专门观察：

```text
正常请求
慢请求
500
404
timeout
connection error
```

httpbin 本身就是为这类 HTTP client 测试设计的。

## 最终目标：一个真正的小工具

例如：

```bash
go-rest-bench \
    -url https://jsonplaceholder.typicode.com/posts/{id} \
    -requests 100 \
    -workers 10 \
    -timeout 2s
```

输出：

```text
Requests:       100
Workers:         10

Success:         98
Failed:           2

Total time:     1.84s
Throughput:    54.35 req/s

Latency:
  avg:          142ms
  p50:          121ms
  p95:          298ms
  p99:          417ms
```

到这里，你就已经开始真正接触：

**throughput、latency、concurrency、worker pool、backpressure**。

现在就从上面的 **Step 0** 开始：先把单请求版本跑起来。下一步不要直接看答案，尝试自己写 `fetchPost(id int)`，然后再进行 code review。
