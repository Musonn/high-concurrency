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

The client runs 2,000 requests at each offered rate—`500`, `600`, and
`800 req/s`. Its default 100 workers let the offered load reach the server;
use `-workers` to override that concurrency.

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

1. Keep the offered rates `500`, `600`, and
   `800 req/s`; 2,000 original requests per rate; 100 client workers; `20ms`
   server service time; and `max-inflight=10`.
2. Retry only `503` responses, in the same worker and without sleeping or
   parsing `Retry-After`.
3. Keep all attempts for an original request under the same request ID. Record
   its final outcome and final latency from initial creation through its final
   attempt.
4. Use this results table. `200` and `503` are response counts across all HTTP
   attempts; final success and final failure are counts of original requests.
   `Efficiency = final success / HTTP attempts`; `Retries / success = retries /
   final success`.

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|

5. Compare D2 with D1. The expected failure mode is that immediate retries
   increase HTTP attempts and `503` responses under overload, while providing
   little or no improvement in final success and increasing final latency.

## D3 — Exponential Backoff with Jitter Plan

D3 keeps D2's workload and server configuration, but replaces immediate retry
with capped exponential backoff and full jitter:

```text
workers      = 100
originals    = 2000 per offered rate
max-inflight = 10
service      = 20ms
max-retries  = 3
base         = 40ms
cap          = 200ms
Retry-After  = omitted
```

For retry number `n`, starting at 1, sample the delay independently as:

```text
delay_n ~ Uniform(0, min(cap, base × 2^(n - 1)))
```

With the D3 constants, the three possible retry delays are sampled from
`[0, 40ms)`, `[0, 80ms)`, and `[0, 160ms)`. The `200ms` cap does not bind with
three retries though.

The client retries only `503` responses. It does not parse or wait for the
server's `Retry-After: 1`; the jittered backoff above is the sole retry delay.
As in D2, a `200`, a non-`503` error, or exhaustion of the third retry is the
final outcome.

Run D3 at `500`, `600`, and `800 req/s` and record the same table used by D2:

`Efficiency = final success / HTTP attempts`; `Retries / success = retries /
final success`.

| Offered (req/s) | Original | HTTP attempts | Retries | Amplification | Efficiency | Retries / success | 200 | 503 | Final failure | Failure % | Success P95 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|

Compare D3 to D2 to test whether spreading retries reduces 503 amplification
while preserving or improving final success. D3's added delay may raise Success
P95; that latency cost should be evaluated alongside the reduction in excess
HTTP work.
