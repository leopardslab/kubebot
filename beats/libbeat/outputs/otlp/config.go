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
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
)

/*
RetryConfig struct is used to define configuration for retrial of sending logs
Enabled: Whether retry should work or not in case of failure
InitialWait: wait for specified time before retrying after first failure
MaxTries: Maximum no of time it will retry before final failure
*/
type RetryConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	InitialWait time.Duration `mapstructure:"initial_wait"`
	MaxTries    int           `mapstructure:"max_tries"`
}

//Provide default config fo retry
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Enabled:     true,
		InitialWait: 5 * time.Second,
		MaxTries:    5,
	}
}

//CredentialConfig to provide username and password for authentication
type CredentialConfig struct {
	UserName string `mapstructure:"user_name"`
	Password string `mapstructure:"password"`
}

// Generator function to create new credential config
func DefaultCredentialConfig() *CredentialConfig {
	return &CredentialConfig{
		UserName: "",
		Password: "",
	}
}

//Config for grpc connection
type GRPCConfig struct {
	KeepAlive    *keepalive.ClientParameters `config:"keep_alive"`
	WaitForReady bool                        `config:"wait_for_ready"`
	Compression  string                      `config:"compression"`
}

//constructor for grpc config
func DefaultGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		KeepAlive:    nil,
		WaitForReady: false,
	}
}

// config for HTTP connection
type HTTPConfig struct {
	KeepAlive    *keepalive.ClientParameters `config:"keep_alive"`
	WaitForReady bool                        `config:"wait_for_ready"`
}

//Constructor for HTTPConfig
func DefaultHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		KeepAlive:    nil,
		WaitForReady: false,
	}
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

func DefaultBackoffConfig() *backoff {
	return &backoff{
		Init: 1 * time.Second,
		Max:  60 * time.Second,
	}
}

// Main config for OTLP output plugin
type OtlpConfig struct {
	Index            string            `config:"index"`
	BatchPublish     bool              `config:"batch_publish"`
	BatchSize        int               `config:"batch_size" validate:"min=1"`
	Protocol         string            `config:"protocol"`
	Auth             *CredentialConfig `config:"authentication"`
	Headers          map[string]string `config:"headers"`
	TLS              *tlscommon.Config `config:"ssl"`
	Codec            codec.Config      `config:"codec"`
	APIKey           string            `config:"api_key"`
	Timeout          time.Duration     `config:"timeout"`
	Retry            *RetryConfig      `config:"retry"`
	ProxyDisable     bool              `config:"proxy_disable"`
	ProxyURL         string            `config:"proxy_url"`
	CompressionLevel int               `config:"compression_level"`
	Backoff          *backoff          `config:"backoff"`
	GRPC             *GRPCConfig       `config:"grpc_config"`
	HTTP             *HTTPConfig       `config:"http_config"`
	LogMax			 int			   `config:"logmax"`
	TickTime		 int			   `config:"ticktime"`
	EnableDebugTimer bool			   `config:"enableDebugTimer"`
}

//Constructor to create default OTLP config
func defaultOLTPConfig() *OtlpConfig {
	return &OtlpConfig{
		Protocol:         "http", //Default protocol is http
		BatchPublish:     false,
		BatchSize:        1,
		Auth:             DefaultCredentialConfig(),
		TLS:              nil,
		APIKey:           "",
		Timeout:          5 * time.Second,
		Retry:            DefaultRetryConfig(),
		ProxyDisable:     true,
		ProxyURL:         "",
		CompressionLevel: 0,
		Backoff:          DefaultBackoffConfig(),
		GRPC:             DefaultGRPCConfig(),
		HTTP:             DefaultHTTPConfig(),
	}
}

//Read User config
func readConfig(cfg *common.Config, info beat.Info) (*OtlpConfig, error) {
	c := defaultOLTPConfig()

	if err := cfg.Unpack(c); err != nil {
		return nil, err
	}

	if c.Index == "" {
		c.Index = strings.ToLower(info.IndexPrefix)
	}

	return c, nil
}

//Validate few params from User config
func (c *OtlpConfig) Validate() error {
	if c.ProxyURL != "" && !c.ProxyDisable {
		if _, err := common.ParseURL(c.ProxyURL); err != nil {
			return err
		}
	}

	if c.TLS.IsEnabled() && (c.TLS.Certificate.Certificate == "" || c.TLS.Certificate.Key == "") {
		return fmt.Errorf("As TLS is enabled, both certificate and key must be set")
	}

	if c.Protocol == "" {
		return fmt.Errorf("Protocol must be provided from [grpc, http]")
	}

	return nil
}
