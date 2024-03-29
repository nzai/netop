package netop

import (
	"net/url"
	"time"
)

// Param download parameters
type Param struct {
	URL              string
	Headers          map[string]string
	Form             url.Values
	Retry            int
	RetryInterval    time.Duration
	LogChannel       chan<- string
	ProgressChannel  chan<- *Progress
	ProgressInterval time.Duration
	ValidStatusCode  map[int]bool
}

// Log send log
func (s Param) Log(log string) {
	if s.LogChannel == nil {
		return
	}

	s.LogChannel <- log
}

// SendProgress send progress
func (s Param) SendProgress(progress *Progress) {
	if s.ProgressChannel == nil {
		return
	}

	s.ProgressChannel <- progress
}

// RequestParam defines download parameters
type RequestParam interface {
	apply(u *Param)
}

type paramFunc func(*Param)

func (f paramFunc) apply(p *Param) {
	f(p)
}

// Refer set request refer
func Refer(refer string) RequestParam {
	return Header("Referer", refer)
}

// Header set request header
func Header(key, value string) RequestParam {
	return paramFunc(func(p *Param) { p.Headers[key] = value })
}

// FormData set formdata
func FormData(key, value string) RequestParam {
	return paramFunc(func(p *Param) { p.Form.Set(key, value) })
}

// Retry set request retry times and interval
func Retry(retry int, interval time.Duration) RequestParam {
	return paramFunc(func(p *Param) {
		p.Retry = retry
		p.RetryInterval = interval
	})
}

// Log set log chan to show operation log
func Log(logChannel chan<- string) RequestParam {
	return paramFunc(func(p *Param) {
		p.LogChannel = logChannel
	})
}

// OnProgress send download progress
func OnProgress(progressChannel chan<- *Progress, interval time.Duration) RequestParam {
	return paramFunc(func(p *Param) {
		p.ProgressChannel = progressChannel
		p.ProgressInterval = interval
	})
}

// ValidStatusCode set valid response status code, otherwise error returned.
func ValidStatusCode(statusCodes ...int) RequestParam {
	return paramFunc(func(p *Param) {
		if len(statusCodes) == 0 {
			return
		}

		if p.ValidStatusCode == nil {
			p.ValidStatusCode = make(map[int]bool, len(statusCodes))
		}

		for _, code := range statusCodes {
			p.ValidStatusCode[code] = true
		}
	})
}
