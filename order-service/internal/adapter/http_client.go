package adapter

import (
	"bytes"
	"io"
	"net/http"
	"order-service/config"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type HttpClient interface {
	CallURL(method, url string, header map[string]string, rawData []byte) (*http.Response, error)
}

type Options struct {
	timeout int
	http    *http.Client
	logger  echo.Logger
}

type loggingTransport struct {
	logger echo.Logger
}

func NewHttpClient(cfg *config.Config) HttpClient {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	transport := &loggingTransport{
		logger: e.Logger,
	}

	client := &http.Client{
		Timeout:   time.Duration(cfg.App.ServerTimeOut) * time.Second,
		Transport: transport,
	}

	return &Options{
		timeout: cfg.App.ServerTimeOut,
		http:    client,
		logger:  e.Logger,
	}
}

func (o *Options) CallURL(method, url string, header map[string]string, rawData []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, url, bytes.NewBuffer(rawData))
	if err != nil {
		o.logger.Errorj(log.JSON{
			"message": "[CallURL - 1] Failed To Prepare Request Client HTTP",
			"error":   err.Error(),
		})
		return nil, err
	}

	if len(header) > 0 {
		for key, value := range header {
			request.Header.Set(key, value)
		}
	}

	response, err := o.http.Do(request)
	if err != nil {
		o.logger.Errorj(log.JSON{
			"message": "[CallURL - 2] Failed To Do Request Client HTTP",
			"error":   err.Error(),
		})
		return nil, err
	}

	return response, nil
}

func (l *loggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	l.logger.Infof("Making request to: %s %s", request.Method, request.URL)
	l.logger.Infof("Request Headers: %+v", request.Header)

	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	l.logger.Infof("Request Body: %s", requestBody)

	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		l.logger.Infof("Request Failed: %v", err)
		return nil, err
	}

	l.logger.Infof("Received response with status: %s", response.Status)
	l.logger.Infof("Response Headers: %+v", response.Header)

	responseBody, err := io.ReadAll(response.Body)
	if err == nil {
		l.logger.Infof("Response Body: %s", responseBody)
	}

	response.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	return response, nil
}
