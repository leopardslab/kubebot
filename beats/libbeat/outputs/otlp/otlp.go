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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/otlp"
)

func init() {
	outputs.RegisterType("otlp", makeOtlpOutput)
}

const (
	otlpSelector             = "otlp"
	defaultGRPCEndpooint     = "localhost:4317"
	defaultHTTPEndpoint      = "localhost:8080"
	maxHTTPResponseReadBytes = 64 * 1024
)

// OTLP output plugin function which is registered with output module.
func makeOtlpOutput(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	//create logger, default config and update it with user config
	log := logp.NewLogger(otlpSelector)
	config, err := readConfig(cfg, beat)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	//Default encoder is otlp
	var enc codec.Codec
	if config.Codec.Namespace.IsSet() {
		enc, err = codec.CreateEncoder(beat, config.Codec)
		if err != nil {
			return outputs.Fail(err)
		}
	} else {
		//var cfg otlp.Config
		enc = otlp.New(beat.Version, otlp.Config{})
	}

	//Create http/grpc clients based on user config
	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		var client outputs.NetworkClient
		if config.Protocol == "grpc" {
			client, err = NewGrpcClient(config, log, observer, enc, host)
			if err != nil {
				return outputs.Fail(err)
			}
		} else {
			client, err = NewHttpClient(config, log, observer, enc, host)
			if err != nil {
				return outputs.Fail(err)
			}
		}
		clients[i] = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
	}
	return outputs.SuccessNet(true, config.BatchSize, config.Retry.MaxTries, clients)
}
