package main

import (
	"math"
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
	// Summary Stats
	record := analyze.NewRecord()

	// Job Control
	requesters := &sync.WaitGroup{}
	reviewers := &sync.WaitGroup{}
	analyzers := &sync.WaitGroup{}

	jobs := make(chan uow.Job, 1)
	retries := make(chan uow.Job, 10)
	results := make(chan uow.Job, 10)
	analysis := make(chan uow.Job, 10)

	// Various kinds of workers.
	for w := 1; w <= 100; w++ {
		logger := log.New("worker", w)
		go request.New(logger, requesters, &http.Client{}, jobs, results).Work()
	}

	{
		logger := log.New("reviewer", "master")
		go review.New(logger, reviewers, results, retries, analysis).Work()
	}

	{
		logger := log.New("analyzer", "master")
		go analyze.New(logger, analyzers, analysis, record).Work()
	}

	// Read in our jobs to be done.
	headers := make(map[string]string)

	// Set our initial delay to 1 second. This will dynamically adjust as
	// we go.
	delay := 500 * time.Millisecond

	// Our Latency SLO in seconds.
	slo := 1.0

	reset := 0

	log.Debug("Adding jobs to the queue.")
	for {
		faster := true
		reset += 1

		select {
		case retry := <-retries:
			faster = false
			jobs <- retry
		default:
			jobs <- uow.Job{Requests: []uow.Request{uow.Request{"GET", "http://localhost:10080", headers, nil}}}
		}

		time.Sleep(delay)

		if reset > 10 {
			record.Truncate(0.95)
			reset = 0
		}

		latency := record.Duration.Quantile(0.95)
		if !math.IsNaN(latency) {
			if latency >= slo {
				faster = false
			}

			if faster == true {
				log.Warn("Faster")
				delay -= 10 * time.Millisecond
				if delay < 0 {
					delay = 1
				}
			} else {
				log.Warn("Slower")
				delay += 10 * time.Millisecond
			}
		}

		log.Debug("Delay", log.Ctx{
			"delay":   delay,
			"latency": latency,
			"rps":     float64(time.Second) / float64(delay),
		})
	}
	log.Debug("Done adding jobs.")

	requesters.Wait()
	reviewers.Wait()
	analyzers.Wait()
}
