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
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"testing"
	"time"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/stretchr/testify/assert"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestReadConfig(t *testing.T) {
	tests := []struct{
		title string
		config map[string]interface{}
		expected OtlpConfig
	}{
		{
		"test default config",
		map[string]interface{}{
				"protocol":          "http", //Default protocol is http
				"batch_publish":     false,
				"batch_size":        1,
				"authentication":    DefaultCredentialConfig(),
				"headers":            map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
				"ssl":               nil,
				"api_key":           "",
				"timeout":           5 * time.Second,
				"retry":             DefaultRetryConfig(),
				"proxy_disable":     true,
				"proxy_url":         "",
				"compression_level": 1,
				"backoff":           DefaultBackoffConfig(),
				"grpc_config":       DefaultGRPCConfig(),
				"http_config":       DefaultHTTPConfig(),
			},
			OtlpConfig{
				Index:            "",
				BatchPublish:     false,
				BatchSize:        1,
				Protocol:         "http",
				Auth:             nil,
				Headers:          map[string]string{"Authorization": "Bearer some-token", "Accept": "application/x-protobuf"},
				TLS:              nil,
				Codec:            codec.Config{},
				APIKey:           "",
				Timeout:          5 * time.Second,
				Retry:            nil,
				ProxyDisable:     true,
				ProxyURL:         "",
				CompressionLevel: 1,
				Backoff:          nil,
				GRPC:             nil,
				HTTP:             nil,
			},
		},
		{
			"test default grpc config",
			map[string]interface{}{
				"protocol":          "grpc", //Default protocol is http
				"batch_publish":     true,
				"batch_size":        1,
				"authentication":    DefaultCredentialConfig(),
				"headers":            map[string]string{"Header1": "Value1"},
				"ssl":               nil,
				"api_key":           "",
				"timeout":           5 * time.Second,
				"retry":             DefaultRetryConfig(),
				"proxy_disable":     true,
				"proxy_url":         "",
				"compression_level": 1,
				"backoff":           DefaultBackoffConfig(),
				"grpc_config":       DefaultGRPCConfig(),
				"http_config":       DefaultHTTPConfig(),
			},
			OtlpConfig{
				Index:            "",
				BatchPublish:     true,
				BatchSize:        1,
				Protocol:         "grpc",
				Auth:             nil,
				Headers:          map[string]string{"Header1": "Value1"},
				TLS:              nil,
				Codec:            codec.Config{},
				APIKey:           "",
				Timeout:          5 * time.Second,
				Retry:            nil,
				ProxyDisable:     true,
				ProxyURL:         "",
				CompressionLevel: 1,
				Backoff:          nil,
				GRPC:             nil,
				HTTP:             nil,
			},
		},
	}
	beat := beat.Info{Version: "1.0", IndexPrefix: "mock"}
	for _, test := range tests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			common_config, _ := common.NewConfigFrom(test.config)
			config, _ := readConfig(common_config, beat)
			assert.Equal(t, test.expected.BatchPublish, config.BatchPublish)
			assert.Equal(t, test.expected.ProxyDisable, config.ProxyDisable)
			assert.Equal(t, test.expected.Protocol, config.Protocol)
			assert.Equal(t, test.expected.BatchSize, config.BatchSize)
			assert.Equal(t, test.expected.CompressionLevel, config.CompressionLevel)
			assert.Equal(t, test.expected.Timeout, config.Timeout)
			assert.Equal(t, test.expected.Headers, config.Headers)
		})
	}
}

func TestValidateProxy(t *testing.T) {
	config := OtlpConfig{
		Index:            "",
		BatchPublish:     false,
		BatchSize:        0,
		Protocol:         "http",
		Auth:             nil,
		Headers:          nil,
		TLS:              nil,
		Codec:            codec.Config{},
		APIKey:           "",
		Timeout:          0,
		Retry:            nil,
		ProxyDisable:     true,
		ProxyURL:         "http://dummy.com",
		CompressionLevel: 0,
		Backoff:          nil,
		GRPC:             nil,
		HTTP:             nil,
	}
	error := config.Validate()
	assert.Equal(t, error, nil)
}

func TestValidateTLS(t *testing.T) {
	enabled := false
	config := OtlpConfig{
		Index:            "",
		BatchPublish:     false,
		BatchSize:        0,
		Protocol:         "http",
		Auth:             nil,
		Headers:          nil,
		TLS:              &tlscommon.Config{Enabled: &enabled, Certificate: tlscommon.CertificateConfig{Key:"sample_key"}},
		Codec:            codec.Config{},
		APIKey:           "",
		Timeout:          0,
		Retry:            nil,
		ProxyDisable:     false,
		ProxyURL:         "",
		CompressionLevel: 0,
		Backoff:          nil,
		GRPC:             nil,
		HTTP:             nil,
	}
	error := config.Validate()
	assert.NoError(t, error, "certificate and key must be set ")
}

func TestValidateProtocol(t *testing.T) {
	config := OtlpConfig{
		Index:            "",
		BatchPublish:     false,
		BatchSize:        0,
		Protocol:         "",
		Auth:             nil,
		Headers:          nil,
		TLS:              nil,
		Codec:            codec.Config{},
		APIKey:           "",
		Timeout:          0,
		Retry:            nil,
		ProxyDisable:     false,
		ProxyURL:         "",
		CompressionLevel: 0,
		Backoff:          nil,
		GRPC:             nil,
		HTTP:             nil,
	}
	error := config.Validate()
	assert.Equal(t, "Protocol must be provided from [grpc, http]", error.Error())
}
