package netop

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
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
	response, err := Get(url, parameters...)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status code: %d", response.StatusCode)
	}

	return ioutil.ReadAll(response.Body)
}

// Get send a GET request to server and return response or error
func Get(url string, parameters ...RequestParam) (*http.Response, error) {
	param := &Param{URL: url}
	for _, parameter := range parameters {
		parameter.apply(param)
	}

	request, err := http.NewRequest("GET", param.URL, nil)
	if err != nil {
		if param.logChannel != nil {
			param.logChannel <- fmt.Sprintf("init http request failed due to %v", err)
		}
		return nil, err
	}

	if param.Refer != "" {
		request.Header.Set("Referer", param.Refer)
	}

	var response *http.Response
	for index := 0; index <= param.Retry; index++ {
		response, err = http.DefaultClient.Do(request)
		if err == nil {
			// return response on status code 200, or give up retry on status code 404
			switch response.StatusCode {
			case http.StatusOK, http.StatusNotFound:
				return response, nil
			default:
				err = fmt.Errorf("response status code: %d", response.StatusCode)
			}
		}

		if param.Retry > index {
			if param.logChannel != nil {
				param.logChannel <- fmt.Sprintf("request failed due to %v, retry in %s(remain %d times)",
					err, param.RetryInterval.String(), param.Retry-index)
			}
			time.Sleep(param.RetryInterval)
		}
	}

	if param.logChannel != nil {
		param.logChannel <- fmt.Sprintf("request failed due to %v, retied %d times)", err, param.Retry)
	}

	return response, err
}

// Param download parameters
type Param struct {
	URL           string
	Refer         string
	Retry         int
	RetryInterval time.Duration
	logChannel    chan<- string
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
		p.logChannel = logChannel
	})
}
