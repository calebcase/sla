package uow

import "time"

type Status int

const (
	TBD = iota
	Done
	Retry
	Fail
)

type Timing struct {
	Start      time.Time
	DNS        time.Duration
	Connection time.Duration
	TLS        time.Duration
	Request    time.Duration
	Delay      time.Duration
	Response   time.Duration
	Stop       time.Time
	Duration   time.Duration
}

type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    *string
}

type Response struct {
	StatusCode    int
	ContentLength int64
}

type Round struct {
	Timing   Timing
	Request  *Request
	Response Response
}

type Job struct {
	Attempts uint64
	Status   Status
	Requests []Request
	Rounds   []*Round
}
