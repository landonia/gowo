// Copyright 2016 Landon Wainwright.

// Package gowo provides a very simple but powerful worker pool
package gowo

import (
	"fmt"
	"sync"
	"time"
)

// Handler is the function that will be called by the handlers
type Handler func(work interface{}) (result interface{})

// Job specifies the work required
type Job struct {
	data     interface{}      // The data for the job
	ResultCh chan interface{} // The results will be sent on the queue
}

// Gowo wraps the worker functionality
type Gowo struct {
	workCh     chan Job            // The incoming work queue
	handler    Handler             // The handler for the workers
	workSize   int                 // The amount of work to give to a worker
	bufferSize int                 // The work queue will be initialised to this size
	workers    int                 // The number of workers that will be started
	exit       chan chan time.Time // A signal to shutdown
	mut        sync.Mutex          // Incase of race conditions on startup/shutdown
	running    bool                // True if the pool is up and running
	waiting    bool                // True if the pool is waiting to shutdown
}

// New will create a new worker pool
func New(workers int, handler Handler, workSize, bufferSize int) *Gowo {
	gowo := &Gowo{
		workers:    workers,
		workCh:     make(chan Job),
		handler:    handler,
		workSize:   workSize,
		bufferSize: bufferSize,
		exit:       make(chan chan time.Time),
	}
	return gowo
}

// Send will send the data into the job queue
func (gowo *Gowo) Send(data interface{}) Job {
	job := Job{
		data:     data,
		ResultCh: make(chan interface{}, 1),
	}
	gowo.workCh <- job
	return job
}

// SendSync will synchronously send the job and wait for the result
func (gowo *Gowo) SendSync(data interface{}) interface{} {
	job := gowo.Send(data)
	return <-job.ResultCh
}

// Running will return true if the pool is still active
func (gowo *Gowo) Running() bool {
	gowo.mut.Lock()
	defer gowo.mut.Unlock()
	return gowo.running
}

// Work will start the workers
// An error will be returned if the pool is already executing
func (gowo *Gowo) Work() (*Gowo, error) {
	gowo.mut.Lock()
	defer gowo.mut.Unlock()
	if gowo.running {
		return gowo, fmt.Errorf("Already running")
	}
	gowo.running = true

	// Start the work in a new routine
	go func() {
		work(gowo.workCh, gowo.handler, gowo.workSize, gowo.bufferSize, gowo.workers, gowo.exit)

		// When it gets here it has finished
		gowo.mut.Lock()
		gowo.waiting = false
		gowo.running = false
		gowo.mut.Unlock()
	}()
	return gowo, nil
}

// work will setup the workers, wait for work and start processing
func work(work chan Job, handler Handler, workSize, bufferSize, workers int, exit chan chan time.Time) {
	var shutdown chan time.Time
	total := 0

	// Create the work buffer
	jobs := make([]Job, 0, bufferSize)
	pos := 0

	// this will allow the workers to request work and receive the work on their queue
	requests := make(chan chan []Job, workers)

	// Spin up the workers
	for i := 0; i < workers; i++ {
		go worker(requests, handler)
	}

	// Now listen for work on the incoming channel
	for {
		select {
		case job, ok := <-work:

			// Add it to the current work queue
			if ok {
				jobs = append(jobs, job)
				pos++
			}
		case request := <-requests:
			if shutdown != nil && pos == 0 {

				// Send the nil signal back to shutdown the worker
				request <- nil
				total++

				// Now the signal has been sent to all the workers we can exit
				if total == workers {
					shutdown <- time.Now()
					return
				}
			} else {

				// We need to take the amount of work required for this worker
				if pos > workSize {

					// Take the amount of work configured
					request <- jobs[:workSize]

					// A slice always holds the original array so we need to copy the items
					// over or else the items will not be garbage collected.
					remaining := jobs[workSize:]
					pos = len(remaining)
					jobs = make([]Job, pos, pos+bufferSize)
					copy(jobs, remaining)
				} else {

					// Take all the work and create a new empty buffer
					request <- jobs[:pos]
					jobs = make([]Job, 0, bufferSize)
					pos = 0
				}
			}
		case signal := <-exit:

			// Exit the system, but let the workers finish what they are doing
			// Stop receiving anymore work
			close(work)

			// Mark that we are shutting down
			shutdown = signal
		}
	}
}

// worker will start up a new worker that will request some work and then send it
func worker(request chan chan []Job, handler Handler) {

	// Request some more work
	work := make(chan []Job)
	for {
		request <- work
		select {
		case jobs := <-work:
			if jobs == nil {

				// Drop out signal
				return
			}

			// Work on the items received
			for _, job := range jobs {
				if result := handler(job.data); job.ResultCh != nil && result != nil {
					job.ResultCh <- result
				}
			}
		}
	}
}

// Stop will not accept any more work on the incoming queue
// allow the workers to finish and then shutdown and cleanup.
func (gowo *Gowo) Stop() (chan time.Time, error) {
	gowo.mut.Lock()
	defer gowo.mut.Unlock()
	if !gowo.running {
		return nil, fmt.Errorf("Not running")
	} else if gowo.waiting {
		return nil, fmt.Errorf("Finishing jobs")
	}
	gowo.waiting = true
	exit := make(chan time.Time)
	gowo.exit <- exit
	return exit, nil
}
