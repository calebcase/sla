package analyze

import (
	"sync"

	"github.com/calebcase/sla/uow"

	log "github.com/inconshreveable/log15"
)

type Worker struct {
	Logger   log.Logger
	Workers  *sync.WaitGroup
	Analysis <-chan uow.Job
	Record   *Record
}

func New(logger log.Logger, workers *sync.WaitGroup, analysis <-chan uow.Job, record *Record) *Worker {
	workers.Add(1)

	worker := Worker{
		Logger:   logger,
		Workers:  workers,
		Analysis: analysis,
		Record:   record,
	}

	return &worker
}

func (w *Worker) Work() {
	for job := range w.Analysis {
		w.Logger.Info("Analyzing!", "job", job)

		for _, round := range job.Rounds {
			w.Record.AddRound(round)
		}

		w.Logger.Debug("Timings", log.Ctx{
			"header": w.Record.Header(),
			"95th":   w.Record.Quantiles(0.95),
		})
	}

	w.Workers.Done()
}
