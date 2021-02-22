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
	// ErrInvalidResponseStatusCode invalid response status code
	ErrInvalidResponseStatusCode = errors.New("invalid response status code")
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
	param := &Param{URL: url, Headers: make(map[string]string)}
	for _, parameter := range parameters {
		parameter.apply(param)
	}

	response, err := doGet(url, param)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if param.ValidStatusCode != nil && !param.ValidStatusCode[response.StatusCode] {
		param.Log(fmt.Sprintf("invalid response status code: %d", response.StatusCode))
		return nil, ErrInvalidResponseStatusCode
	}

	start := time.Now()
	now := start
	lastProgressAt := start
	var completed, lastCompleted, speed, intervalSeconds int64
	var interval time.Duration
	var read int

	buffer := new(bytes.Buffer)
	temp := make([]byte, 10240)
	for {
		read, err = response.Body.Read(temp)
		if read < 0 {
			break
		}

		_, err1 := buffer.Write(temp[:read])
		if err1 != nil {
			return nil, fmt.Errorf("write buffer failed due to %v", err1)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read response failed due to %v", err)
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
	param := &Param{URL: url, Headers: make(map[string]string)}
	for _, parameter := range parameters {
		parameter.apply(param)
	}

	return doGet(url, param)
}

// doGet send a GET request to server and return response or error
func doGet(url string, param *Param) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, param.URL, nil)
	if err != nil {
		param.Log(fmt.Sprintf("init http request failed due to %v", err))
		return nil, err
	}

	for key, value := range param.Headers {
		request.Header.Set(key, value)
	}

	var response *http.Response
	var message string
	for index := 0; index <= param.Retry; index++ {
		response, err = http.DefaultClient.Do(request)
		if err == nil {
			return response, nil
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
