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
	"fmt"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	beat2 "github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/otlp"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
)

func TestPublish(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	logger := logp.NewLogger("otlp_test")
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"},},
	}
	batch := outest.NewBatch(events...)
	config := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     false,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),
	}
	common_config, _ := common.NewConfigFrom(config)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	cfg, _ := readConfig(common_config, beat)
	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(cfg, logger, outputs.NewNilObserver(), enc, svr.URL)
	if err := c.Connect(); err != nil {
		fmt.Print(err.Error())
		return
	}
	err := c.Publish(nil, batch)
	fmt.Println(err)
	assert.Equal(t, err, nil)
}

func TestPublishEvents(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	logger := logp.NewLogger("otlp_test")
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"}},
	}
	batch := outest.NewBatch(events...)

	config_data := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     false,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),
	}
	common_config, _ := common.NewConfigFrom(config_data)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	config, _ := readConfig(common_config, beat)

	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(config, logger, outputs.NewNilObserver(), enc, svr.URL)
	c.Connect()

	//Publish all events
	_, err :=  c.PublishEvents(batch.Events())
	assert.NilError(t, err)
}

func TestPublishInBatch(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	logger := logp.NewLogger("otlp_test")
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"}},
	}
	batch := outest.NewBatch(events...)

	config_data := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     true,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),

	}
	common_config, _ := common.NewConfigFrom(config_data)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	config, _ := readConfig(common_config, beat)

	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(config, logger, outputs.NewNilObserver(), enc, svr.URL)
	c.Connect()

	//Publish only 1 event
	beatEvents := batch.Events()
	err :=  c.PublishInBatch(beatEvents)
	assert.NilError(t, err)
}

func TestPublishEvent(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	logger := logp.NewLogger("otlp_test")
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"}},
	}
	batch := outest.NewBatch(events...)

	config_data := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     false,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),
	}
	common_config, _ := common.NewConfigFrom(config_data)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	config, _ := readConfig(common_config, beat)

	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(config, logger, outputs.NewNilObserver(), enc, svr.URL)
	c.Connect()

	//Publish only 1 event
	event := batch.Events()[0]
	err :=  c.PublishEvent(event)
	assert.NilError(t, err)
}

func TestSendRequest(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	logger := logp.NewLogger("otlp_test")
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"}},
	}
	batch := outest.NewBatch(events...)

	config_data := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     false,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),
	}
	common_config, _ := common.NewConfigFrom(config_data)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	config, _ := readConfig(common_config, beat)

	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(config, logger, outputs.NewNilObserver(), enc, svr.URL)
	c.Connect()

	//Send POST request to server
	event := batch.Events()[0]
	serializedEvent, err := c.codec.Encode(c.connConfig.Index, &event.Content)
	status, _, err :=  c.SendRequest("POST", svr.URL, bytes.NewBuffer(serializedEvent))
	assert.NilError(t, err)
	assert.Equal(t, 200, status)
}

func TestCreateRequest(t *testing.T) {
	//Create a test server
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "dummy_data")
	}))
	defer svr.Close()

	method := "POST"
	logger := logp.NewLogger("otlp_test")
	config_data := map[string]interface{}{
		"protocol":          "http", //Default protocol is http
		"batch_publish":     false,
		"batch_size":        1,
		"authentication":    DefaultCredentialConfig(),
		"header":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
		"ssl":               nil,
		"api_key":           "",
		"timeout":           5 * time.Second,
		"retry":             DefaultRetryConfig(),
		"proxy_disable":     true,
		"proxy_url":         "",
		"compression_level": 0,
		"backoff":           DefaultBackoffConfig(),
		"grpc_config":       DefaultGRPCConfig(),
		"http_config":       DefaultHTTPConfig(),
	}
	common_config, _ := common.NewConfigFrom(config_data)
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	config, _ := readConfig(common_config, beat)

	enc := otlp.New(beat.Version, otlp.Config{})
	c, _ := NewHttpClient(config, logger, outputs.NewNilObserver(), enc, svr.URL)

	//Create a request and check it with expected values
	req, err := c.CreateNewRequest(method, svr.URL, nil)
	assert.NilError(t, err)
	assert.Equal(t, "application/x-protobuf", req.Header["Content-Type"][0])
	assert.Equal(t, svr.URL, req.URL.String())
	assert.Equal(t, method, req.Method)
}
