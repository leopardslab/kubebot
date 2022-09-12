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
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	"testing"
	"time"
)

/*
TestOtlpCodec tests the functionality of the OTLP codec. It first creates a beat event and populates different fields of the beat event.
The beat event is created using the testCase struct, then the beat event is used to create the expected OTLP packet and is passed to the OTLP encoder
to get the serialized data which is the marshalled OTLP packet. The serialized data is then converted to actual data using unmarshalling,
and the assertion to check if the actual and expected values are the same is applied.
*/

// testing otlp codec marshalling
func TestOtlpCodec(t *testing.T) {
	type testCase struct {
		config Config
		ts     time.Time
		in     common.MapStr
	}

	cases := map[string]testCase{
		"testk8scontainer": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"container": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
		},
		"testk8snode": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"name": "docker-desktop"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8snode2": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"labels": common.MapStr{"beta_kubernetes_io/os": "linux"}}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8snamespace": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"namespace": common.MapStr{"name": "lca"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8sdeployment": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"deployment": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8spod": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"pod": common.MapStr{"ip": "10.1.0.46"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testnilvals": testCase{
			in: common.MapStr{"message": "",
				"k8s":             common.MapStr{"": common.MapStr{"": ""}},
				"agent":           common.MapStr{"": ""},
				"_message_parser": common.MapStr{"": ""}},
			config: Config{Resources: []string{"k8s"}},
		},
	}

	for name, test := range cases {
		cfg, ts, fields := test.config, test.ts, test.in

		t.Run(name, func(t *testing.T) {
			codec := New("1.2.3", cfg)
			serializedEvent, err := codec.Encode("test", &beat.Event{Fields: fields, Timestamp: ts})
			actual, er := UnmarshalLogs(serializedEvent)
			expected := pdata.NewLogs()
			createExpectedOTLPPacket(&expected, fields, ts)

			if err != nil {
				t.Errorf("Error during encoding event %v", err)
			}
			if er != nil {
				t.Errorf("Error during umarshaling %v", er)
			}
			assert.NoError(t, err)
			assert.NoError(t, er)
			assert.EqualValues(t, expected, actual)
		})
	}
}

func TestOtlpCodecBatch(t *testing.T) {
	type testCase struct {
		config Config
		ts     time.Time
		in     common.MapStr
	}

	cases := map[string]testCase{
		"testk8scontainer": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"container": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
		},
		"testk8snode": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"name": "docker-desktop"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8snode2": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"labels": common.MapStr{"beta_kubernetes_io/os": "linux"}}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8snamespace": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"namespace": common.MapStr{"name": "lca"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8sdeployment": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"deployment": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testk8spod": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"pod": common.MapStr{"ip": "10.1.0.46"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config: Config{Resources: []string{"k8s"}},
		},
		"testnilvals": testCase{
			in: common.MapStr{"message": "",
				"k8s":             common.MapStr{"": common.MapStr{"": ""}},
				"agent":           common.MapStr{"": ""},
				"_message_parser": common.MapStr{"": ""}},
			config: Config{Resources: []string{"k8s"}},
		},
	}

	for name, test := range cases {
		cfg, ts, fields := test.config, test.ts, test.in

		t.Run(name, func(t *testing.T) {
			codec := New("1.2.3", cfg)
			batch := make([]publisher.Event, 1)
			batch[0].Content = beat.Event{Fields: fields, Timestamp: ts}
			otlpPacket, err := codec.EncodeBatch("test", batch)
			actual := otlpPacket.(*pdata.Logs)
			expectedPacket := pdata.NewLogs()
			createExpectedOTLPPacket(&expectedPacket, fields, ts)
			var expected *pdata.Logs
			expected = &expectedPacket
			if err != nil {
				t.Errorf("Error during encoding event %v", err)
			}

			assert.NoError(t, err)
			assert.EqualValues(t, expected, actual)
		})
	}
}

// helper function to create expected otlp packet
func createExpectedOTLPPacket(ld *pdata.Logs, fields common.MapStr, ts time.Time) {
	resourceLogs := AddResourceLogs(ld)
	instrumentationLibraryLogs := AddInstrumentationLibraryLogs(&resourceLogs)
	logData := AddLogRecord(&instrumentationLibraryLogs)
	AddTimeStamp(&logData, ts)
	AddAttributes(ld, fields)
}

// helper function to add timestamp
func AddTimeStamp(lr *pdata.LogRecord, ts time.Time) {
	lr.SetTimestamp(pdata.NewTimestampFromTime(ts))
}

// helper function to add attributes
func AddAttributes(Logs *pdata.Logs, Fields common.MapStr) {
	for fields, val := range Fields {
		if fields == "agent" {
			instrumentationLibraryLog := GetInstrumentationLibraryLog(Logs)
			PopulateInstrumentationLibrary(&instrumentationLibraryLog, val)
		} else if fields == "message" {
			logRecord := GetLogRecord(Logs)
			PopulateLogRecord(&logRecord, val.(string))
		} else {
			dfsAddAttributes("", fields, val, Logs, false)
		}
	}
}
