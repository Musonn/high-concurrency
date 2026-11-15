# Experiment E — Client Deadline and Server Cancellation

Experiment E shows what happens when a client gives up on a request before the
server has finished doing its work. It compares a server that ignores client
cancellation with one that honors it.

## Fixed configuration

```ini
service-time = 100ms
max-inflight = 10
client-deadline = 30ms
```

The server's theoretical capacity is:

```text
10 / 0.1s = 100 req/s
```

Every client request has a 30ms deadline. Since the server needs 100ms to
complete normal work, an admitted request will outlive the client deadline
unless the server stops when the request context is cancelled.

## E1 — Server ignores cancellation

Keep the server's work implementation as:

```go
time.Sleep(serviceTime)
```

The client times out after 30ms, but the server continues working for the full
100ms. Its in-flight slot remains occupied until that work completes.

Record:

```text
client deadline exceeded
server completed work
server canceled work
```

Expected result: client deadline failures increase while the server still
reports completed work; canceled work should remain zero. This is wasted work
because the client no longer needs the response.

## E2 — Server honors cancellation

Replace unconditional sleeping with context-aware work:

```go
select {
case <-time.After(serviceTime):
	// normal completion

case <-r.Context().Done():
	// client no longer needs the work
	return
}
```

The sequence is now:

```text
client timeout at 30ms
        ↓
request context canceled
        ↓
server stops work
        ↓
slot released
```

Record the same counters as E1

## Comparison

Compare E1 and E2 using the three counters and, if useful, the observed
in-flight occupancy. E2 should show canceled server work and earlier slot
release, making cancellation's capacity benefit directly visible.
