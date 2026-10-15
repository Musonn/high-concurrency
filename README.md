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
`600`, and `800 req/s`. Its default 100 workers let the offered load reach the
server; use `-workers` to override that concurrency.

## D2 — Immediate-Retry Plan

D2 intentionally demonstrates the unsafe retry policy that can create a retry
storm. It uses the same mock-server and client-load configuration as D1, with
this policy for each original request:

```text
max retries  = 3
on 503       = retry immediately
Retry-After  = ignored
backoff      = none
jitter       = none
```

`max retries = 3` means an original request makes one initial attempt plus at
most three retry attempts, for a maximum of four HTTP attempts. A `200` ends
the request successfully. A fourth `503`, or a non-retryable error, ends it as
a final failure.

Implementation and measurement plan:

1. Keep the D1 configuration: offered rates `300`, `400`, `500`, `600`, and
   `800 req/s`; 2,000 original requests per rate; 100 client workers; `20ms`
   server service time; and `max-inflight=10`.
2. Retry only `503` responses, in the same worker and without sleeping or
   parsing `Retry-After`.
3. Keep all attempts for an original request under the same request ID. Record
   its final outcome and final latency from initial creation through its final
   attempt.
4. Use this results table. `200` and `503` are response counts across all HTTP
   attempts; final success and final failure are counts of original requests.

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | 200 | 503 | Final success | Final failure | Success % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|

5. Compare D2 with D1. The expected failure mode is that immediate retries
   increase HTTP attempts and `503` responses under overload, while providing
   little or no improvement in final success and increasing final latency.