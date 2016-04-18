# gowo

A simple but effective worker pool.

## Overview

I keep coming across the issue where I need to hand off work, but because there
is the possibility of it blocking, you end up launching go routines just to push
an item to the channel. Gowo will always take what is on the channel and add
the item to a work queue. The work queue buffer is 0 by default but is configurable
to initialise it to a sensible length if required. By default there is one worker,
which for most tasks is enough. If the task is time or IO bound and lends itself
well to being asynchronous, you can add as many workers as is necessary. The workers
will request work when they have finished their current block. By default, everything
on the queue will be sent to the next requesting worker but you can configure this
to ensure the work is more evenly spread.

## Maturity

Complete but requires the tests to be truly finished (coming soon)

## Installation

simply run `go get github.com/landonia/gowo`

## Use as Library

I have created the library to be as simple as possible, but even with this it
can be very flexible and allows you to customise to suit your requirements.
Ideally, you will write your application using the asynchronous method, but
if this cannot be achieved with an existing pattern you can use the `pool.SendSAync()`
method which will block until it receives the result.

This is a simple example that shows how it can be used within an application.

```go
	package main

	import (
  		"github.com/landonia/goat"
  	)

  	func main() {
			// Allocate the available CPU cores
			noWorkers := runtime.NumCPU()
		  runtime.GOMAXPROCS(noWorkers)

		 // Create a new worker system
		 pool, _ := gowo.New(
			 noWorkers,
			 func(work interface{}) (result interface{}) {
			 	return fmt.Sprintf("work.%v", work)
			 },
			 10,  // 10 items can be given to a worker in batch
			 100, // The bufer will be initialised with a capacity of 100
		 ).Work()

		 // synchronously wait for all the tasks
		 result := pool.SendSync("Work")

		 // Asynchronous call will return a job with a result channel
		 job := pool.Send("Work")
		 result = <-job.ResultCh

		 // Shutdown will not allow any new jobs but will wait for all existing
		 // jobs to finish.
		 exit, err := pool.Stop()
		 <-exit
  	}
```

## Example

simply run `go run github/landonia/gowo/cmd/example.go`

## About

golf was written by [Landon Wainwright](http://www.landotube.com) | [GitHub](https://github.com/landonia).

Follow me on [Twitter @landoman](http://www.twitter.com/landoman)!
