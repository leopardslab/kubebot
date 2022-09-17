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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"go.opentelemetry.io/collector/model/pdata"
	"hash/fnv"
)

// Encoder for serializing a beat.Event to otlp.
type Encoder struct {
	version string
	config  Config
	logger  *logp.Logger
	OtlpLog *pdata.Logs
	ResourceIndex map[uint64]int
}

// Config is used to pass encoding parameters to New Encoder.
type Config struct {
	Resources []string
}

var defaultConfig = Config{Resources: []string{}}
var configCache = map[string]bool{}

func init() {
	codec.RegisterType("otlp", func(info beat.Info, cfg *common.Config) (codec.Codec, error) {
		config := defaultConfig
		if cfg != nil {
			if err := cfg.Unpack(&config); err != nil {
				return nil, err
			}
		}

		return New(info.Version, config), nil
	})
}

// fillGlobalConfigCache marks the values which should go to resource attributes
func (e *Encoder) fillGlobalConfigCache() {
	cfg := e.config
	for _, val := range cfg.Resources {
		configCache[val] = true
	}
}

// New creates a new otlp Encoder, and fills the global configCache
func New(version string, config Config) *Encoder {
	e := &Encoder{version: version, config: config, ResourceIndex: map[uint64]int{}}
	e.fillGlobalConfigCache()
	return e
}

// genResourceHash generates unique hash for given resource
func genResourceHash(e *Encoder, event *beat.Event) uint64 {
	var ResourceUIDBuffer bytes.Buffer
	GetResourceUID(event, &ResourceUIDBuffer)

	h := fnv.New64a()
	_, err := h.Write(ResourceUIDBuffer.Bytes())
	if err != nil {
		e.logger.Error("Error getting resource mentioned in config file")
	}
	return h.Sum64()
}

func (e *Encoder) EncodeBatch(index string, batch []publisher.Event) (interface{}, error) {

	for index, event := range batch {
		beatEvent := event.Content
		resourceUID := genResourceHash(e, &beatEvent)
		// for first event create empty ExportLogsServiceReques
		if index == 0 {
			exportLogService := CreateOTLPLogs()
			e.OtlpLog = &exportLogService
		}
		// add logs to existing resource packet or append new resource packet and add logs to it,
		// depending on whether resource is already present in ExportLogsServiceRequest or not
		if resourceIndex, ok := e.ResourceIndex[resourceUID]; ok {
			AppendLogsAtResourceIndex(e.OtlpLog, &beatEvent, resourceIndex)
		} else {
			AddLogsToNewResourceLogsAtEnd(e.OtlpLog, &beatEvent)
			e.ResourceIndex[resourceUID] = e.OtlpLog.ResourceLogs().Len() - 1
		}
	}

	// return Otlp packet, http client marshal this to buffer and grpc client directly use this packet
	exportLogService := e.OtlpLog

	// set map and otlp packet pointer to nil
	e.ResourceIndex = map[uint64]int{}
	e.OtlpLog = nil

	return exportLogService, nil
}

// Encode serializes a beat event to OTLP
func (e *Encoder) Encode(index string, event *beat.Event) ([]byte, error) {
	resourceUID := genResourceHash(e, event)

	// return Otlp packet, http client marshal this to buffer and grpc client directly use this packet
	exportLogService := CreateOTLPLogs()
	e.OtlpLog = &exportLogService
	AddLogsToNewResourceLogsAtEnd(e.OtlpLog, event)
	e.ResourceIndex[resourceUID] = e.OtlpLog.ResourceLogs().Len() - 1

	// set map and otlp packet pointer to nil
	e.ResourceIndex = map[uint64]int{}
	if e.OtlpLog == nil {
		return nil, nil
	}
	buf, err := MarshalLogs(*e.OtlpLog)
	if err != nil {
		e.logger.Error("unable to unmarshal")
		return nil, err
	}
	e.OtlpLog = nil
	return buf, nil
}
