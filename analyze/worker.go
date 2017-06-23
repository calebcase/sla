package analyze

import (
	"sync"

	"github.com/calebcase/sla/uow"

	"github.com/felixge/pidctrl"
	log "github.com/inconshreveable/log15"
)

type Worker struct {
	Logger   log.Logger
	Workers  *sync.WaitGroup
	Analysis <-chan uow.Job
	SLO      float64
	Delay    *float64

	record *Record
	pid    *pidctrl.PIDController
}

func New(logger log.Logger, workers *sync.WaitGroup, analysis <-chan uow.Job, slo float64, delay *float64) *Worker {
	workers.Add(1)

	worker := Worker{
		Logger:   logger,
		Workers:  workers,
		Analysis: analysis,
		SLO:      slo,
		Delay:    delay,

		record: NewRecord(),
		pid:    pidctrl.NewPIDController(2.0, 0.5, 0.3).Set(slo),
	}
	worker.pid.SetOutputLimits(-1.0, 1.0)

	return &worker
}

func (w *Worker) Work() {
	for job := range w.Analysis {
		w.Logger.Info("Analyzing!", "job", job)

		for _, round := range job.Rounds {
			w.record.AddRound(round)
		}

		q95 := w.record.Trailing.Quantile(0.95)
		w.Logger.Debug("Timings", log.Ctx{
			"95th": q95,
		})

		adj := w.pid.Update(q95)

		w.Logger.Debug("Adjust?", log.Ctx{
			"adj": adj,
		})

		*w.Delay -= adj
		if *w.Delay < 0.01 {
			*w.Delay = 0.01
		}
		if *w.Delay > 5.0 {
			*w.Delay = 5.0
		}
	}

	w.Workers.Done()
}
