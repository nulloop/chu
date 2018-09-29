```
   _|_|_|  _|    _|  _|    _|
 _|        _|    _|  _|    _|
 _|        _|_|_|_|  _|    _|
 _|        _|    _|  _|    _|
   _|_|_|  _|    _|    _|_|   is a new way of working with events
```

```go
func main() {
  provider := nats.NewProvider()

  r := provider.Receiver()

  r.Use(loggerMiddleware)

  r.Route("a.b.c", func (r chu.Receiver) {
    r.Use(parseMiddleware)

    // subscribe to topic "a.b.c.test"
    r.Handle("test", func (msg chu.Message) error {
      ...
    })
  })

  messageID := "1"
  aggregateID := "2"

  // some where in code
  r.Sender().Send(
    nats.NewMessage(
      context.Background(),
      messageID,
      aggregateID,
      "a.b.c.test",
      []byte("hello world"),
    ),
  )
}
```
