// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package otlp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/otlp"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"go.opentelemetry.io/collector/model/pdata"
)

type HttpClient struct {
	endpoint         string
	logger           *logp.Logger
	client           *http.Client
	observer         outputs.Observer
	connConfig       *OtlpConfig
	codec            codec.Codec
	logMax           int
	codecLogCount    int
	exporterLogCount int
	tickTime         int
	enableDebugTimer bool
	timerRunning     bool
}

// NewHttpClient is a Constructor for HttpClient
func NewHttpClient(
	config *OtlpConfig,
	logger *logp.Logger,
	observer outputs.Observer,
	codec codec.Codec,
	endpoint string,
) (*HttpClient, error) {
	client := &HttpClient{
		endpoint:         endpoint,
		logger:           logger,
		client:           nil,
		observer:         observer,
		connConfig:       config,
		codec:            codec,
		logMax:           config.LogMax,
		tickTime:         getTickTime(config.TickTime),
		enableDebugTimer: config.EnableDebugTimer,
		timerRunning:     false,
	}
	return client, nil
}

func (h *HttpClient) startTimer(tickTime int) {
	ticker := time.NewTicker(time.Duration(tickTime) * time.Second)
	for {
		select {
		case <-ticker.C:
			h.logger.Debug("Total logs processed by codec in ", strconv.Itoa(tickTime), " seconds ", " is ", h.codecLogCount)
			h.logger.Debug("Total logs exported by HTTP exporter in ", strconv.Itoa(tickTime), " seconds ", " is ", h.exporterLogCount)
			h.codecLogCount = 0
			h.exporterLogCount = 0
		}
	}
}

// Connect Create http transport and make connection to endpoint
func (h *HttpClient) Connect() error {
	tlsConfig, err := tlscommon.LoadTLSConfig(h.connConfig.TLS)
	if err != nil {
		h.logger.Errorf("Error in Connect(): " + err.Error())
		return err
	}

	transporter := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: h.connConfig.Timeout,
		}).DialContext,
		TLSClientConfig:   tlsConfig.ToConfig(),
		DisableKeepAlives: true,
	}

	var proxyURL *url.URL
	if !h.connConfig.ProxyDisable && h.connConfig.ProxyURL != "" {
		proxyURL, err = common.ParseURL(h.connConfig.ProxyURL)
		if err != nil {
			h.logger.Errorf("Error in Connect(): " + err.Error())
			return err
		}
		transporter.Proxy = http.ProxyURL(proxyURL)
	}

	h.client = &http.Client{
		Transport: transporter,
		Timeout:   h.connConfig.Timeout,
	}
	return nil
}

// Provide name of the plugin
func (h *HttpClient) String() string {
	return "OTLP/HTTP(" + h.endpoint + ")"
}

// Close connection to endpoint
func (h *HttpClient) Close() error {
	h.client.CloseIdleConnections()
	h.client = nil
	return nil
}

// Publish publishes the logs in OTLP format to endpoint
func (h *HttpClient) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	if h.connConfig.BatchPublish {
		err := h.PublishInBatch(events)
		if err != nil {
			batch.RetryEvents(events)
		} else {
			batch.ACK()
		}
		return err
	} else {
		h.observer.NewBatch(len(events))
		dropped, err := h.PublishEvents(events)
		if len(dropped) > 0 {
			batch.RetryEvents(dropped)
			h.observer.Acked(len(events) - len(dropped))
		} else {
			batch.ACK()
		}
		return err
	}

}

// PublishEvents publishes all the events to endpoint
func (h *HttpClient) PublishEvents(events []publisher.Event) ([]publisher.Event, error) {
	totalEvents := len(events)
	if totalEvents == 0 {
		return nil, nil
	}

	if h.client == nil {
		return events, transport.ErrNotConnected
	}

	var dropped []publisher.Event
	err := error(nil)
	for index, event := range events {
		err = h.PublishEvent(event)
		if err != nil {
			dropped = events[index:]
		}
	}
	if len(dropped) > 0 {
		return dropped, err
	}
	return nil, nil
}

// PublishInBatch makes one network call per batch of beat events
func (h *HttpClient) PublishInBatch(events []publisher.Event) error {
	if h.enableDebugTimer && !h.timerRunning {
		go h.startTimer(h.tickTime)
		h.timerRunning = true
	}
	t1 := time.Now()
	serializedEventInterface, err := h.codec.EncodeBatch("", events)
	if err != nil {
		h.observer.WriteError(err)
		fmt.Printf("error while encoding inside PublishEvent(): " + err.Error())
		h.logger.Errorf(err.Error())
		return err
	}
	LogCounter += len(events)
	h.codecLogCount += len(events)
	// log time taken by each event after every LogMax events
	if LogCounter >= h.logMax {
		h.logger.Debugf("Total time taken by Codec for translation of %s logs is ", len(events), time.Since(t1))
	}

	serializedEvent := serializedEventInterface.(*pdata.Logs)
	buf, err := otlp.MarshalLogs(*serializedEvent)
	if err != nil {
		h.observer.WriteError(err)
		h.logger.Warn("Error ", err, "on creating new otlp codec")
	}

	t1 = time.Now()
	status, _, err := h.SendRequest("POST", h.endpoint, bytes.NewBuffer(buf))
	if err != nil {
		h.observer.WriteError(err)
		h.logger.Errorf("Error while sending request from publishEvent(): " + err.Error())
		return err
	}
	h.exporterLogCount += len(events)
	// log time taken by each event after every LogMax events
	if LogCounter >= h.logMax {
		h.logger.Debugf("Total time taken by Exporter for exporting %s logs is ", len(events), time.Since(t1))
		LogCounter = 0
	}

	if status != 200 {
		err = fmt.Errorf("invalid respose status %v from http host : %s", status, h.endpoint)
		h.observer.WriteError(err)
		return err
	}

	h.observer.WriteBytes(len(buf) + 1)
	return nil
}

// PublishEvent publish a single event
func (h *HttpClient) PublishEvent(event publisher.Event) error {
	LogCounter++
	if h.enableDebugTimer && !h.timerRunning {
		go h.startTimer(h.tickTime)
		h.timerRunning = true
	}
	t1 := time.Now()
	serializedEvent, err := h.codec.Encode(h.connConfig.Index, &event.Content)
	if err != nil {
		h.observer.WriteError(err)
		fmt.Printf("error while encoding inside PublishEvent(): " + err.Error())
		h.logger.Errorf(err.Error())
		return err
	}
	h.codecLogCount++
	// log time taken by each event after every LogMax events
	if LogCounter == h.logMax {
		h.logger.Debug("Total time taken by Codec for translation single log = ", time.Since(t1))
	}

	t1 = time.Now()
	status, _, err := h.SendRequest("POST", h.endpoint, bytes.NewBuffer(serializedEvent))
	if err != nil {
		h.observer.WriteError(err)
		h.logger.Errorf("Error while sending request from publishEvent(): " + err.Error())
		return err
	}
	h.exporterLogCount++
	// log time taken by each event after every LogMax events
	if LogCounter == h.logMax {
		h.logger.Debug("Total time taken by Exporter for exporting single log = ", time.Since(t1))
	} else if LogCounter > h.logMax {
		LogCounter = 0
	}
	if status != 200 {
		err = fmt.Errorf("invalid respose status %v from http host : %s", status, h.endpoint)
		h.observer.WriteError(err)
		return err
	}

	h.observer.WriteBytes(len(serializedEvent) + 1)
	return nil
}

// SendRequest send the request to endpoint
func (h *HttpClient) SendRequest(method string, url string, body io.Reader) (int, []byte, error) {
	req, err := h.CreateNewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Errorf("Error while sending request: " + err.Error())
		return 0, nil, err
	}
	defer h.closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		return status, nil, fmt.Errorf("%v", resp.Status)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return status, nil, err
	}
	return status, respData, err
}

// CreateNewRequest create new Http request with user headers
func (h *HttpClient) CreateNewRequest(method string, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		h.logger.Errorf("Error while creating http request: " + err.Error())
		return nil, err
	}

	if h.connConfig.Auth.UserName != "" && h.connConfig.Auth.Password != "" {
		request.SetBasicAuth(h.connConfig.Auth.UserName, h.connConfig.Auth.Password)
	}

	for key, value := range h.connConfig.Headers {
		request.Header.Add(key, value)
	}
	request.Header.Add("Content-Type", "application/x-protobuf")
	return request, nil
}

// Closing: Close the remaining body after request is sent
func (h *HttpClient) closing(respBody io.Closer) {
	err := respBody.Close()
	if err != nil {
		h.logger.Errorf("Error while closing the response: " + err.Error())
	}
}
