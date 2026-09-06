package main

import (
	"context"
	"sync"
)

// retryDispatcher is a single, dedicated goroutine whose only job is
// draining retries and feeding them back into the main jobs queue. It
// never does real work (no Handle calls), so it's always available to
// receive — meaning a worker sending onto retries never blocks waiting
// for a busy worker, only for this always-ready dispatcher.
func retryDispatcher(ctx context.Context, retries <-chan Job, jobs chan<- Job, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case retriedJob := <-retries:
			select {
			case jobs <- retriedJob:
			case <-ctx.Done():
				return
			}
		}
	}
}
