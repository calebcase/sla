package review

import (
	"sync"

	"github.com/calebcase/sla/uow"

	log "github.com/inconshreveable/log15"
)

type Worker struct {
	Logger   log.Logger
	Workers  *sync.WaitGroup
	Results  <-chan uow.Job
	Retries  chan<- uow.Job
	Analysis chan<- uow.Job
}

func New(logger log.Logger, workers *sync.WaitGroup, results <-chan uow.Job, retries chan<- uow.Job, analysis chan<- uow.Job) *Worker {
	workers.Add(1)

	worker := Worker{
		Logger:   logger,
		Workers:  workers,
		Results:  results,
		Retries:  retries,
		Analysis: analysis,
	}

	return &worker
}

func (w *Worker) Work() {
	for job := range w.Results {
		job.Attempts += 1

		// Fail jobs that are set to retry, but exceed our retry max.
		if job.Status == uow.Retry {
			if job.Attempts > 3 {
				job.Status = uow.Fail
			}
		}

		switch job.Status {
		case uow.TBD:
			w.Retries <- job
		case uow.Done:
			w.Logger.Debug("Sending for analysis!", "job", job)
			w.Analysis <- job
		case uow.Retry:
			w.Logger.Info("Retrying!", "job", job)
			w.Retries <- job
		case uow.Fail:
			w.Logger.Warn("Failed!", "job", job)
		default:
			w.Logger.Error("Job status unknown! Dropping on the floor.", "job", job)
		}
	}

	w.Workers.Done()
}
