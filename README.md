# Next Lesson — Retries and Retry Storms

Load shedding introduces a real client-side question:

```text
Server: 503 or 429
Client: The request failed, so I will retry immediately.
```

That instinct can undo the protection that load shedding provides. If an
`800 req/s` workload has already saturated a service that can sustain roughly
`480 req/s`, then about `320 req/s` may be rejected. If every rejected request
is retried immediately, the offered load can grow like this:

```text
800 original requests/s
+
320 retries/s
+
retries of retries
+
...
```

In other words, clients can turn a mechanism intended to protect an overloaded
service into a more severe overload of their own making: a **retry storm**.

The next experiment will extend the existing mock server and client to create,
observe, and control a retry storm. It will introduce:

- retries
- exponential backoff
- jitter
- retry budgets

The mock server now supplies the overload signal. Run it with a fixed service
time and a concurrency cap; once all ten slots are busy, `/work` immediately
returns `503` and includes `Retry-After: 1`.

```bash
go run ./cmd/mockserver \
  -service-time=20ms \
  -max-inflight=10
```

Set `-max-inflight=0` to disable overload rejection and retain the earlier,
unbounded mock-server behavior.

With the server running, run the client in a separate terminal:

```bash
go run .
```

The client runs 2,000 requests at each offered rate—`300`, `400`, `500`,
`600`, and `800 req/s`—and reports original requests, total HTTP attempts,
`200` and `503` responses, final failures, retries, and P95 final latency for
each rate. Its default 100 workers let the offered load reach the server; use
`-workers` to override that concurrency.

That progression leads naturally to building resilient high-concurrency
services: retry only when it is likely to help, spread retries over time, and
limit the extra load that retries may create.
