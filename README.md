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

That progression leads naturally to building resilient high-concurrency
services: retry only when it is likely to help, spread retries over time, and
limit the extra load that retries may create.
