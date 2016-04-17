package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/landonia/gowo"
)

func main() {
	noWorkers := runtime.NumCPU()
	runtime.GOMAXPROCS(noWorkers)

	// Create a new worker system
	pool, _ := gowo.New(
		noWorkers,
		func(work interface{}) (result interface{}) {
			return fmt.Sprintf("work.%v", work)
		},
		10,
		100,
	).Work()
	start := time.Now()
	var results []interface{}

	// synchronously wait for all the tasks
	for i := 0; i < 10000; i++ {
		results = append(results, pool.SendSync(i))
	}
	log.Println(fmt.Sprintf("Sync: %d in %s", len(results), time.Now().Sub(start)))

	// Now asynchronously push some work onto the queue
	// This is not getting the best performance doing it in a sync sort of way,
	// but it demonstrates the performance differences
	start = time.Now()
	var jobs []gowo.Job
	for i := 10000; i < 20000; i++ {
		jobs = append(jobs, pool.Send(i))
	}
	if exit, err := pool.Stop(); err == nil {
		<-exit
		log.Println("Finished work")
	} else {
		log.Println(fmt.Sprintf("Error: %s", err.Error()))
	}
	results = make([]interface{}, 0)
	for _, job := range jobs {
		results = append(results, <-job.ResultCh)
	}
	log.Println(fmt.Sprintf("Async: %d in %s", len(jobs), time.Now().Sub(start)))
}
