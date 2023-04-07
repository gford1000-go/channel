[![Go Reference](https://pkg.go.dev/badge/github.com/gford1000-go/channel.svg)](https://pkg.go.dev/github.com/gford1000-go/channel)

channel
========

channel implements a type, `ChanMgr` which supports non-blockable as well as the usual blocking submissions to the `chan` it manages.

In addition, `context.Context` is supported by the `ChanMgr`.  See [Google's blog](https://go.dev/blog/context) for more details.


```go
func processor[T](ctx context.Context, c <-chan T) {
	select {
	case <-ctx.Done():
		return
	case t := <-c:
		// Do something
	}
}

func main() {

	// Create a cancellable context (if required)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the managed change, capacity = 10
	m := NewChanMgr[int](ctx, 10)

	// Start processor, listening to channel
	go processor[int](ctx, m.GetChan())

	// Create work for the processor, with the option to 
	// block or non-block against the chan
	for i := 0; i < 1000; i++ {
		err := m.AddNoBlocking(i)
		if err != nil && err == ErrChanIsFull {
			// can't add to chan at present, do something else
		}
	}

}
```

## How?

The command line is all you need.

```
go get github.com/gford1000-go/connection
```
