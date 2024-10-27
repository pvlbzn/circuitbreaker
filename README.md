# Circuit Breaker

[Circuit Breaker by Martin Fowler](https://martinfowler.com/bliki/CircuitBreaker.html).


## States

Three states:
- `closed`
- `open`
- `half-open`

### Closed

Healthy operating state where all requests are allowed. When requests start to fail consecutively CB shifts its state to `open`.

### Open

All requests are blocked for a pre-defined time period, error returned instead of calling the failing service. After reaching recovery period CD transitions into `half-open` state.

### Half-Open

Limited number of request allowed to see if service recovered. If requests start to return success CB transitions to `closed`, healthy, state. If requests still return errors state transitions to `open`.

