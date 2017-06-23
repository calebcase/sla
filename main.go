package main

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/calebcase/sla/analyze"
	"github.com/calebcase/sla/request"
	"github.com/calebcase/sla/review"
	"github.com/calebcase/sla/uow"

	log "github.com/inconshreveable/log15"
)

func cannot(err error) {
	if err != nil {
		log.Crit("Fatal error.", "err", err)
		os.Exit(1)
	}
}

func main() {
	// Job Control
	requesters := &sync.WaitGroup{}
	reviewers := &sync.WaitGroup{}
	analyzers := &sync.WaitGroup{}

	jobs := make(chan uow.Job, 1)
	retries := make(chan uow.Job, 10)
	results := make(chan uow.Job, 10)
	analysis := make(chan uow.Job, 10)

	delay := 1.0
	slo := 0.250

	// Various kinds of workers.
	for w := 1; w <= 10; w++ {
		logger := log.New("worker", w)
		go request.New(logger, requesters, &http.Client{}, jobs, results).Work()
	}

	{
		logger := log.New("reviewer", "master")
		go review.New(logger, reviewers, results, retries, analysis).Work()
	}

	{
		logger := log.New("analyzer", "master")
		go analyze.New(logger, analyzers, analysis, slo, &delay).Work()
	}

	// Read in our jobs to be done.
	headers := make(map[string]string)

	log.Debug("Adding jobs to the queue.")
	for {
		select {
		case retry := <-retries:
			jobs <- retry
		default:
			jobs <- uow.Job{Requests: []uow.Request{uow.Request{"GET", "http://localhost:10080", headers, nil}}}
		}

		time.Sleep(time.Duration(delay * 1000000000.0))

		log.Debug("Delay", log.Ctx{
			"delay": delay,
		})
	}
	log.Debug("Done adding jobs.")

	requesters.Wait()
	reviewers.Wait()
	analyzers.Wait()
}
