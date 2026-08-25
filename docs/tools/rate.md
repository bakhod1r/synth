# Real-time pacing

Deliver records over wall-clock time, the way a real event source would — for
streaming tests and consumer back-pressure experiments:

```go
synth.Rate[Event](synth.RateConfig{PerSecond: 5000, Jitter: 0.2}).
    Run(ctx, func(e Event) error { return myProducer.Send(e) })
```

| Field | Meaning |
| --- | --- |
| `PerSecond` | target delivery rate |
| `Jitter` | fraction of that interval to vary by, so arrivals aren't a metronome |

Synth paces the handoff; where the events go is your code's decision. The
handler is your Kafka producer, your HTTP client, your channel — Synth opens no
connection of its own, here or anywhere else.

Cancelling the context stops the run; the handler's error stops it too, and is
returned.
