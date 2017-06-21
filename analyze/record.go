package analyze

import (
	"fmt"

	"github.com/calebcase/sla/uow"

	"github.com/stripe/veneur/tdigest"
)

type Record struct {
	DNS        *tdigest.MergingDigest
	Connection *tdigest.MergingDigest
	TLS        *tdigest.MergingDigest
	Request    *tdigest.MergingDigest
	Delay      *tdigest.MergingDigest
	Response   *tdigest.MergingDigest
	Duration   *tdigest.MergingDigest
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
