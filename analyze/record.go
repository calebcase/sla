package analyze

import (
	"fmt"

	"github.com/calebcase/sla/uow"

	"github.com/gonum/stat"
	"github.com/stripe/veneur/tdigest"
)

type Circular struct {
	Current int
	Data    []float64
}

func NewCircular(size int) *Circular {
	circular := Circular{
		Data: make([]float64, size),
	}

	return &circular
}

func (c *Circular) Add(data float64) {
	c.Data[c.Current] = data
	c.Current += 1

	if c.Current >= len(c.Data) {
		c.Current = 0
	}
}

func (c *Circular) Quantile(quantile float64) float64 {
	data := make([]float64, len(c.Data))
	copy(data, c.Data)
	stat.SortWeighted(data, nil)

	return stat.Quantile(quantile, stat.Empirical, data, nil)
}

type Record struct {
	DNS        *tdigest.MergingDigest
	Connection *tdigest.MergingDigest
	TLS        *tdigest.MergingDigest
	Request    *tdigest.MergingDigest
	Delay      *tdigest.MergingDigest
	Response   *tdigest.MergingDigest
	Duration   *tdigest.MergingDigest

	Trailing *Circular
}

func NewRecord() *Record {
	record := &Record{
		DNS:        tdigest.NewMerging(100, false),
		Connection: tdigest.NewMerging(100, false),
		TLS:        tdigest.NewMerging(100, false),
		Request:    tdigest.NewMerging(100, false),
		Delay:      tdigest.NewMerging(100, false),
		Response:   tdigest.NewMerging(100, false),
		Duration:   tdigest.NewMerging(100, false),

		Trailing: NewCircular(10),
	}

	return record
}

func (r *Record) AddRound(round *uow.Round) {
	r.DNS.Add(round.Timing.DNS.Seconds(), 1.0)
	r.Connection.Add(round.Timing.Connection.Seconds(), 1.0)
	r.TLS.Add(round.Timing.TLS.Seconds(), 1.0)
	r.Request.Add(round.Timing.Request.Seconds(), 1.0)
	r.Delay.Add(round.Timing.Delay.Seconds(), 1.0)
	r.Response.Add(round.Timing.Response.Seconds(), 1.0)
	r.Duration.Add(round.Timing.Duration.Seconds(), 1.0)

	r.Trailing.Add(round.Timing.Duration.Seconds())
}

func (r *Record) Header() []string {
	headers := []string{
		"DNS",
		"Connection",
		"TLS",
		"Request",
		"Delay",
		"Response",
		"Duration",
	}

	return headers
}

func (r *Record) Quantiles(quantile float64) []float64 {
	results := []float64{
		r.DNS.Quantile(quantile),
		r.Connection.Quantile(quantile),
		r.TLS.Quantile(quantile),
		r.Request.Quantile(quantile),
		r.Delay.Quantile(quantile),
		r.Response.Quantile(quantile),
		r.Duration.Quantile(quantile),
	}

	return results
}

func (r *Record) String() string {
	return fmt.Sprintf("%v", r.Quantiles(0.95))
}
