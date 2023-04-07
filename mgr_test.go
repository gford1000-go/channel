package channel

import (
	"context"
	"testing"
	"time"
)

func TestChanMgr(t *testing.T) {

	m := NewChanMgr[int](context.Background(), 10)

	for i := 0; i < 10; i++ {
		err := m.Add(i)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	c := m.GetChan()

	for i := 0; i < 10; i++ {
		o := <-c
		if i != o {
			t.Fatalf("mismatch: expected %d, got %d", i, o)
		}
	}
}

func TestChanMgrAvoidBlock(t *testing.T) {

	m := NewChanMgr[int](context.Background(), 1)

	m.Add(1)                  // Should be ok
	err := m.AddNoBlocking(2) // Should return error

	if err == nil || err != ErrChanIsFull {
		t.Fatal("unexpected")
	}
}

func TestChanMgrBlocking(t *testing.T) {

	m := NewChanMgr[int](context.Background(), 1)
	done := make(chan bool, 1)

	go func() {
		m.Add(1) // Should be ok
		m.Add(2) // Should block
		done <- true
	}()

	time.Sleep(5 * time.Millisecond)

	received1 := false
	doneFlag := false
	for !doneFlag {
		select {
		case o := <-m.GetChan():
			if o == 1 {
				received1 = true
			}
		case doneFlag = <-done:
			if !received1 {
				t.Fatal("unexpected order")
			}
		}
	}
}

func TestContext(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	m := NewChanMgr[int](ctx, 1)

	m.Add(1)

	cancel()

	<-m.GetChan()

	err := m.Add(2) // Should generate error as context is cancelled

	if err == nil || err != ErrInvalidChanMgr {
		t.Fatal("unexpected")
	}
}

func TestContext2(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	m := NewChanMgr[int](ctx, 1)

	startedConsumer := make(chan bool, 1)

	go func() {
		// Consuming goroutine
		startedConsumer <- true

		for {
			select {
			case <-ctx.Done():
				return

			case <-m.GetChan():
				continue
			}
		}

	}()

	<-startedConsumer

	// Submitting goroutines
	routines := 200
	submissions := 100
	invalidError := make(chan bool, routines*submissions)
	for i := 0; i < routines; i++ {
		go func() {
			for i := 0; i < submissions; i++ {
				err := m.Add(i)
				if err != nil && err != ErrInvalidChanMgr {
					invalidError <- true
				}
			}
		}()
	}

	// Allow activity to spin up
	<-time.After(200 * time.Millisecond)

	// Cancel context - this should result in graceful exit
	// with no deadlocks
	cancel()

	// Allow time to pass
	<-time.After(100 * time.Millisecond)

	// Look for any unexpected behaviour
	if len(invalidError) > 0 {
		t.Fatal("unexpected")
	}
}

func TestContext3(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	m := NewChanMgr[int](ctx, 1)

	// Immediate context cancellation - this will still generate
	// a race condition such that submissions might succeed
	// for a short while, hence not checking when err == nil
	cancel()

	err := m.Add(1)
	if err != nil && err != ErrInvalidChanMgr {
		t.Fatalf("unexpected %v", err)
	}

	err = m.AddNoBlocking(1)
	if err != nil && err != ErrInvalidChanMgr {
		t.Fatalf("unexpected %v", err)
	}

}
