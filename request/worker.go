package request

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	"github.com/calebcase/sla/uow"

	log "github.com/inconshreveable/log15"
)

type Worker struct {
	Logger  log.Logger
	Workers *sync.WaitGroup
	Client  *http.Client
	Jobs    <-chan uow.Job
	Results chan<- uow.Job
	Delay   time.Duration
}

func New(logger log.Logger, workers *sync.WaitGroup, client *http.Client, jobs <-chan uow.Job, results chan<- uow.Job) *Worker {
	workers.Add(1)

	worker := Worker{
		Logger:  logger,
		Workers: workers,
		Client:  client,
		Jobs:    jobs,
		Results: results,
	}

	return &worker
}

func (w *Worker) Work() {
	for job := range w.Jobs {
		for _, request := range job.Requests {
			w.Logger.Info("Making a request!", "request", request)
			job.Rounds = nil

			round, err := w.request(&request)
			if err != nil {
				w.Logger.Error("Error making request.", "err", err)
				break
			}
			job.Rounds = append(job.Rounds, round)

			if round.Response.StatusCode < 200 || 300 <= round.Response.StatusCode {
				w.Logger.Error("Bad status code.", "code", round.Response.StatusCode)
				job.Status = uow.Retry
				break
			}
		}

		if job.Status == uow.TBD {
			job.Status = uow.Done
		}

		w.Results <- job
	}

	w.Workers.Done()
}

func (w *Worker) request(request *uow.Request) (*uow.Round, error) {
	// Initialize an empty round.
	round := &uow.Round{Request: request}

	// Create basic request.
	var body_reader io.Reader
	if request.Body != nil {
		body_reader = strings.NewReader(*request.Body)
	}
	req, err := http.NewRequest(request.Method, request.URL, body_reader)
	if err != nil {
		return nil, err
	}

	// Add headers.
	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	// Prepare request tracing.
	var dns_start time.Time
	var connection_start time.Time
	var tls_start time.Time
	var request_start time.Time
	var delay_start time.Time
	var response_start time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dns_start = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			round.Timing.DNS = time.Now().Sub(dns_start)
		},
		GetConn: func(hostPort string) {
			connection_start = time.Now()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			round.Timing.Connection = time.Now().Sub(connection_start)
			request_start = time.Now()
		},
		TLSHandshakeStart: func() {
			tls_start = time.Now()
		},
		TLSHandshakeDone: func(tlsInfo tls.ConnectionState, err error) {
			round.Timing.TLS = time.Now().Sub(tls_start)
		},
		WroteRequest: func(wroteInfo httptrace.WroteRequestInfo) {
			round.Timing.Request = time.Now().Sub(request_start)
			delay_start = time.Now()
		},
		GotFirstResponseByte: func() {
			round.Timing.Delay = time.Now().Sub(delay_start)
			response_start = time.Now()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Actually make the request.
	round.Timing.Start = time.Now()
	res, err := w.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Now drain the body away. This includes the time it takes to read the
	// body in the total timing. It seems reasonable that this should be
	// included since most clients would likely not only read it, but do
	// something with it before moving on to the next request.
	io.Copy(ioutil.Discard, res.Body)

	round.Timing.Stop = time.Now()
	round.Timing.Response = round.Timing.Stop.Sub(response_start)
	round.Timing.Duration = round.Timing.Stop.Sub(round.Timing.Start)

	round.Response.StatusCode = res.StatusCode
	round.Response.ContentLength = res.ContentLength

	return round, nil
}
