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
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
)

type testCase struct {
	config                 Config
	ts                     time.Time
	in                     common.MapStr
	resourceAttributeKey   string
	resourceAttributeValue string
	logAttributeKey        string
	logAttributeValue      interface{}
}

func TestCreateOTLPLogs(t *testing.T) {
	cases := map[string]testCase{
		"testk8scontainer": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"container": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config: Config{Resources: []string{"k8s"}},
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
		cfg, _, _ := test.config, test.ts, test.in

		t.Run(name, func(t *testing.T) {
			e := &Encoder{version: name, config: cfg}
			e.fillGlobalConfigCache()

			expected := pdata.NewLogs()

			actual := CreateOTLPLogs()
			assert.Equal(t, expected, actual)
		})
	}
}

// helper function to create empty OTLP log packet
func NewEmptyOTLPLog(ld *pdata.Logs) {
	ld.ResourceLogs().AppendEmpty().InstrumentationLibraryLogs().AppendEmpty().Logs().AppendEmpty()
}

// helper function to add time stamp
func AddLogTimeStamp(lr *pdata.Logs, ts time.Time) {
	lr.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).SetTimestamp(pdata.NewTimestampFromTime(ts))
}

// testing addition of resource attributes
func TestAddResourceAttributes(t *testing.T) {
	attributeTestKey := "testAttributeKey"
	attributeTestValue := "testAttributeValue"

	expected := pdata.NewLogs()
	NewEmptyOTLPLog(&expected)
	expected.ResourceLogs().At(0).Resource().Attributes().InsertString(attributeTestKey, attributeTestValue)

	actual := pdata.NewLogs()
	NewEmptyOTLPLog(&actual)
	AddResourceAttributes(attributeTestKey, attributeTestValue, &actual)

	assert.Equal(t, expected, actual)
}

// testing addition of log attributes
func TestAddLogAttributes(t *testing.T) {
	attributeTestKey := "testAttributeKey"
	attributeTestValue := "testAttributeValue"

	expected := pdata.NewLogs()
	NewEmptyOTLPLog(&expected)
	expected.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Attributes().InsertString(attributeTestKey, attributeTestValue)

	actual := pdata.NewLogs()
	NewEmptyOTLPLog(&actual)
	AddLogAttributes(attributeTestKey, attributeTestValue, &actual)

	assert.Equal(t, expected, actual)
}

// testing depth first search population of attributes
func TestDfsAddAttributes(t *testing.T) {
	value := []interface{}{"%{DATESTAMP:time} %{LOGLEVEL:severity} %{WORD:class}:%{NUMBER:line} - %{GREEDYDATA:data}",
		"%{DATESTAMP_RFC2822:time} %{LOGLEVEL:severity} %{GREEDYDATA:data}"}
	attrValueArr := pdata.NewAttributeValueArray()
	for _, vv := range value {
		switch vvalue := vv.(type) {
		case string:
			attrValueArr.ArrayVal().AppendEmpty().SetStringVal(vvalue)
		}
	}

	cases := map[string]testCase{
		"testk8scontainer": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"container": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.container.name",
			resourceAttributeValue: "log-gen-app-log4j",
			logAttributeKey:        "_message_parser.pattern",
			logAttributeValue:      "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n",
		},
		"testk8snode": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"name": "docker-desktop"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.node.name",
			resourceAttributeValue: "docker-desktop",
			logAttributeKey:        "_message_parser.type",
			logAttributeValue:      "log4j",
		},
		"testk8snode2": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"node": common.MapStr{"labels": common.MapStr{"beta_kubernetes_io/os": "linux"}}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.node.labels.beta_kubernetes_io/os",
			resourceAttributeValue: "linux",
			logAttributeKey:        "_message_parser.type",
			logAttributeValue:      "log4j",
		},
		"testk8snamespace": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"namespace": common.MapStr{"name": "lca"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.namespace.name",
			resourceAttributeValue: "lca",
			logAttributeKey:        "_message_parser.pattern",
			logAttributeValue:      "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n",
		},
		"testk8sdeployment": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"deployment": common.MapStr{"name": "log-gen-app-log4j"}},
				"agent":           common.MapStr{"type": "filebeat"},
				"_message_parser": common.MapStr{"type": "log4j"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.deployment.name",
			resourceAttributeValue: "log-gen-app-log4j",
			logAttributeKey:        "_message_parser.type",
			logAttributeValue:      "log4j",
		},
		"testk8spod": testCase{
			in: common.MapStr{"message": "/var/log/containers/log-gen-app-log4j-764cdfd897-gmczr_lca_log-gen-app-log4j-12cee5b5eb22f8dffeab583b0c2855250d81213ffc859ae2a12338e8209b0b5f.log",
				"k8s":             common.MapStr{"pod": common.MapStr{"ip": "10.1.0.46"}},
				"agent":           common.MapStr{"version": "8.0.0"},
				"_message_parser": common.MapStr{"pattern": "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n"}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.pod.ip",
			resourceAttributeValue: "10.1.0.46",
			logAttributeKey:        "_message_parser.pattern",
			logAttributeValue:      "%d{yyyy-MM-dd'T'HH:mm:ss.SSSSSSSSS} %highlight{%p} %style{%C{1.} [%t] %m}{bold,red}%n",
		},
		"testnilvals": testCase{
			in: common.MapStr{"message": "",
				"k8s":             common.MapStr{"": common.MapStr{"": ""}},
				"agent":           common.MapStr{"": ""},
				"_message_parser": common.MapStr{"": ""}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s",
			resourceAttributeValue: "",
			logAttributeKey:        "_message_parser",
			logAttributeValue:      "",
		},
		"testmultipattern": testCase{
			in: common.MapStr{"k8s": common.MapStr{"container": common.MapStr{"name": "log-gen-app-log4j"}},
				"_message_parser": common.MapStr{"pattern": []interface{}{"%{DATESTAMP:time} %{LOGLEVEL:severity} %{WORD:class}:%{NUMBER:line} - %{GREEDYDATA:data}",
					"%{DATESTAMP_RFC2822:time} %{LOGLEVEL:severity} %{GREEDYDATA:data}"}}},
			config:                 Config{Resources: []string{"k8s"}},
			resourceAttributeKey:   "k8s.container.name",
			resourceAttributeValue: "log-gen-app-log4j",
			logAttributeKey:        "_message_parser.pattern",
			logAttributeValue:      attrValueArr,
		},
	}

	for name, test := range cases {
		cfg, _, fields, attributeKey, attributeValue, logAttributeKey, logAttributeValue := test.config, test.ts, test.in, test.resourceAttributeKey, test.resourceAttributeValue, test.logAttributeKey, test.logAttributeValue

		t.Run(name, func(t *testing.T) {
			e := &Encoder{version: name, config: cfg}
			e.fillGlobalConfigCache()

			expected := pdata.NewLogs()
			NewEmptyOTLPLog(&expected)
			AddResourceAttributes(attributeKey, attributeValue, &expected)
			AddLogAttributes(logAttributeKey, logAttributeValue, &expected)

			actual := pdata.NewLogs()
			NewEmptyOTLPLog(&actual)
			for keys, val := range fields {
				if keys != "agent" && keys != "message" {
					dfsAddAttributes("", keys, val, &actual, false)
				}
			}
			assert.Equal(t, expected, actual)

		})
	}
}
