package main

import (
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"time"
)

var cleanScheduler gocron.Scheduler

func StartExpiredCleanSchedule() {
	var err error
	cleanScheduler, err = gocron.NewScheduler()
	if err != nil {
		panic(err)
	}
	job, err := cleanScheduler.NewJob(gocron.DurationJob(5*time.Minute), gocron.NewTask(cleanTask))
	if err != nil {
		panic(err)
	}
	// each job has a unique id
	fmt.Println(job.ID())
	cleanScheduler.Start()
	// block until you are ready to shut down
	select {
	case <-time.After(time.Minute):
	}
}

func StopScheduler() {
	err := cleanScheduler.Shutdown()
	if err != nil {
		fmt.Println(err)
	}
}
