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
	Record   *Record
	SLO      float64
	Delay    *float64

	pid *pidctrl.PIDController
}

func New(logger log.Logger, workers *sync.WaitGroup, analysis <-chan uow.Job, record *Record, slo float64, delay *float64) *Worker {
	workers.Add(1)

	worker := Worker{
		Logger:   logger,
		Workers:  workers,
		Analysis: analysis,
		Record:   record,
		SLO:      slo,
		Delay:    delay,
		pid:      pidctrl.NewPIDController(10.0, 0.5, 0.3).Set(slo),
	}
	worker.pid.SetOutputLimits(-1.0, 1.0)

	return &worker
}

func (w *Worker) Work() {
	i := 10

	for job := range w.Analysis {
		w.Logger.Info("Analyzing!", "job", job)

		i += 1
		if i > 10 {
			w.Record = NewRecord()
			i = 0
		}

		for _, round := range job.Rounds {
			w.Record.AddRound(round)
		}

		w.Logger.Debug("Timings", log.Ctx{
			"header": w.Record.Header(),
			"95th":   w.Record.Quantiles(0.95),
		})

		q95 := w.Record.Duration.Quantile(0.95)
		adj := w.pid.Update(q95)

		w.Logger.Debug("Adjust?", log.Ctx{
			"95th": q95,
			"adj":  adj,
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
