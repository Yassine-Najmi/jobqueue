package main

// retryDispatcher is a single, dedicated goroutine whose only job is
// draining retries and feeding them back into the main jobs queue. It
// never does real work (no Handle calls), so it's always available to
// receive — meaning a worker sending onto retries never blocks waiting
// for a busy worker, only for this always-ready dispatcher.
func retryDispatcher(retries <-chan Job, jobs chan<- Job) {
	for retriedJob := range retries {
		jobs <- retriedJob
	}
}
