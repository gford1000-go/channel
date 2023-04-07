package channel

import (
	"context"
	"errors"
)

type payload[T any] struct {
	blockable bool
	t         T
	c         chan error
}

func submitPayload[T any](t T, blockable bool, c chan *payload[T]) error {
	p := &payload[T]{
		blockable: blockable,
		t:         t,
		c:         make(chan error, 1),
	}

	c <- p
	err := <-p.c
	return err
}

// ChanMgr extends normal chan behaviour to allow
// full chans to be detected so that the submitting
// goroutine can take alternative actions can be taken
// if required, rather than simply blocking
type ChanMgr[T any] struct {
	c <-chan T
	l chan *payload[T]
}

// GetChan returns the chan so that items can be removed
func (c *ChanMgr[T]) GetChan() <-chan T {
	return c.c
}

// Add will just add the item to the managed queue without
// checks for the chan being full.  This is analogous to
// normal use of chan and will cause blocking if the chan is full.
func (c *ChanMgr[T]) Add(t T) (e error) {
	defer func() {
		if r := recover(); r != nil {
			e = ErrInvalidChanMgr
		}
	}()

	return submitPayload(t, true, c.l)
}

// AddNoBlocking will check for the chan being full, and returns an
// error if it is, or nil if the item was added without blocking.
func (c *ChanMgr[T]) AddNoBlocking(t T) (e error) {
	defer func() {
		if r := recover(); r != nil {
			e = ErrInvalidChanMgr
		}
	}()

	return submitPayload(t, false, c.l)
}

var ErrChanIsFull = errors.New("chan is full")
var ErrInvalidChanMgr = errors.New("ChanMgr is invalid")

// NewChanMgr returns an instance of ChanMgr that is managing a
// channel of the requested capacity.
// If the context completes then the ChanMgr becomes invalid.
func NewChanMgr[T any](ctx context.Context, capacity int) *ChanMgr[T] {

	c := make(chan T, capacity)

	m := &ChanMgr[T]{
		c: c,
		l: make(chan *payload[T], 1),
	}

	go func() {

		exiting := false

		for {
			if exiting {

				for len(m.l) > 0 {
					p := <-m.l
					p.c <- ErrInvalidChanMgr
				}

				close(m.l)

				return

			} else {

				select {
				case <-ctx.Done():
					// Prevent further items being added,
					// as they can't be processed since this
					// goroutine will end
					exiting = true

				case p := <-m.l:
					if !p.blockable {
						// Since only this goroutine has access to
						// add to this chan, this test is valid
						// in concurrent scenarios (i.e. no race condition)
						if len(c) == cap(c) {
							p.c <- ErrChanIsFull
							continue
						}
					}

					c <- p.t
					p.c <- nil
				}

			}
		}
	}()

	return m
}
