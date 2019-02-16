package netop

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	// ErrNotFound not found
	ErrNotFound = errors.New("not found")
)

// GetString send a GET request to server and return response string or error
func GetString(url string, parameters ...RequestParam) (string, error) {
	buffer, err := GetBytes(url, parameters...)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

// GetBytes send a GET request to server and return response buffer or error
func GetBytes(url string, parameters ...RequestParam) ([]byte, error) {
	buffer, err := GetBuffer(url, parameters...)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// GetBuffer send a GET request to server and return response buffer or error
func GetBuffer(url string, parameters ...RequestParam) (*bytes.Buffer, error) {
	param := &Param{URL: url}
	for _, parameter := range parameters {
		parameter.apply(param)
	}

	response, err := doGet(url, param)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status code: %d", response.StatusCode)
	}

	buffer := new(bytes.Buffer)

	start := time.Now()
	now := start
	lastProgressAt := start
	var completed, lastCompleted, speed, intervalSeconds int64
	var interval time.Duration
	var read int
	temp := make([]byte, 10240)
	for {
		read, err = response.Body.Read(temp)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read response failed due to %v", err)
		}

		_, err = buffer.Write(temp[:read])
		if err != nil {
			return nil, fmt.Errorf("write buffer failed due to %v", err)
		}

		if param.ProgressInterval >= 0 {
			now = time.Now()
			interval = now.Sub(lastProgressAt)
			if interval < param.ProgressInterval {
				continue
			}

			intervalSeconds = int64(interval.Seconds())
			if intervalSeconds == 0 {
				continue
			}

			completed = int64(buffer.Len())
			speed = (completed - lastCompleted) / intervalSeconds
			if speed == 0 {
				continue
			}

			param.ProgressChannel <- &Progress{
				Total:     response.ContentLength,
				Completed: completed,
				Speed:     speed,
				Elapsed:   now.Sub(start),
				Remain:    time.Second * time.Duration((response.ContentLength-completed)/speed),
			}
			lastProgressAt = now
			lastCompleted = completed
		}
	}

	return buffer, nil
}

// Get send a GET request to server and return response or error
func Get(url string, parameters ...RequestParam) (*http.Response, error) {
	param := &Param{URL: url}
	for _, parameter := range parameters {
		parameter.apply(param)
	}

	return doGet(url, param)
}

// doGet send a GET request to server and return response or error
func doGet(url string, param *Param) (*http.Response, error) {
	request, err := http.NewRequest("GET", param.URL, nil)
	if err != nil {
		param.Log(fmt.Sprintf("init http request failed due to %v", err))
		return nil, err
	}

	if param.Refer != "" {
		request.Header.Set("Referer", param.Refer)
	}

	var response *http.Response
	var message string
	for index := 0; index <= param.Retry; index++ {
		response, err = http.DefaultClient.Do(request)
		if err == nil {
			// return response on status code 200, or give up retry on status code 404
			switch response.StatusCode {
			case http.StatusOK, http.StatusNotFound:
				return response, nil
			}
		}

		if param.Retry > index {
			if err != nil {
				message = err.Error()
			} else {
				message = fmt.Sprintf("response status code %d", response.StatusCode)
			}

			param.Log(fmt.Sprintf("request failed due to %s, retry in %s(remain %d times)",
				message, param.RetryInterval.String(), param.Retry-index))
			time.Sleep(param.RetryInterval)
		}
	}

	param.Log(fmt.Sprintf("request failed due to %v, retied %d times)", err, param.Retry))

	return response, err
}

// Progress download progress
type Progress struct {
	Total     int64
	Completed int64
	Speed     int64
	Elapsed   time.Duration
	Remain    time.Duration
}

// Param download parameters
type Param struct {
	URL              string
	Refer            string
	Retry            int
	RetryInterval    time.Duration
	LogChannel       chan<- string
	ProgressChannel  chan<- *Progress
	ProgressInterval time.Duration
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
	return paramFunc(func(p *Param) { p.Refer = refer })
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
