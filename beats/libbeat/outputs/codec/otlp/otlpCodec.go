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
	"sort"
	"time"

	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// CreateOTLPCodec creates the required otlp data structure
func CreateOTLPLogs() pdata.Logs {
	return pdata.NewLogs()
}

// AppendLogsAtResourceIndex appends log data at given resource index
func AppendLogsAtResourceIndex(Logs *pdata.Logs, event *beat.Event, resourceIndex int) {
	instrumentationLibraryLogs := Logs.ResourceLogs().At(resourceIndex).InstrumentationLibraryLogs().At(0)
	logData := AddLogRecord(&instrumentationLibraryLogs)
	PopulateTimeStamp(&logData, event)
	PopulateLogAttributes(&logData, event)
}

// AddLogsToNewResourceLogsAtEnd appends a new resource log packet in ExportLogsServiceRequest,
// and populates resource attributes, instrumentation details, and log details
func AddLogsToNewResourceLogsAtEnd(Logs *pdata.Logs, event *beat.Event) {
	resourceLogs := AddResourceLogs(Logs)
	instrumentationLibraryLogs := AddInstrumentationLibraryLogs(&resourceLogs)
	logData := AddLogRecord(&instrumentationLibraryLogs)
	PopulateTimeStamp(&logData, event)
	PopulateAttributes(Logs, event)
}

// PopulateLogAttributes populates log attributes to given resource packet
func PopulateLogAttributes(logRecord *pdata.LogRecord, event *beat.Event) {
	for fields, val := range event.Fields {
		if fields == "message" {
			PopulateLogRecord(logRecord, val.(string))
		} else {
			dfsAddLogAttributes("", fields, val, logRecord, false)
		}
	}

}

// dfsGetResourceAttributes recursively iterate over fields of beat event to get resource attributes
func dfsGetResourceAttributes(name, key string, val interface{}, parentInConfig bool, ResourceUID []string) []string {
	var childKey string
	if name != "" {
		childKey = name + "." + key
	} else {
		childKey = key
	}
	switch value := val.(type) {
	case string:
		if configCache[childKey] || parentInConfig {
			ResourceUID = append(ResourceUID, value)
		}
	case common.MapStr:
		for kk, vv := range value {
			ResourceUID = dfsGetResourceAttributes(childKey, kk, vv, parentInConfig || configCache[childKey], ResourceUID)
		}
	}
	return ResourceUID
}

/*
PopulateAttributes populates instrumentationLibrary data, log body, and move different attributes to
either log attributes or resource attributes depending on config provided in filebeat.yml
*/
func GetResourceUID(event *beat.Event, ResourceUIDBuffer *bytes.Buffer) {
	ResourceUID := make([]string, 0)
	for fields, val := range event.Fields {
		ResourceUID = dfsGetResourceAttributes("", fields, val, false, ResourceUID)
	}
	sort.Strings(ResourceUID)
	for _, val := range ResourceUID {
		ResourceUIDBuffer.WriteString(val)
	}
}

// dfsAddLogAttributes recursively iterate over fields of beat event to get log attributes
func dfsAddLogAttributes(name, key string, val interface{}, Logs *pdata.LogRecord, parentInConfig bool) {
	var childKey string
	if name != "" {
		childKey = name + "." + key
	} else {
		childKey = key
	}
	switch value := val.(type) {
	case string:
		if !(configCache[childKey] || parentInConfig) {
			Logs.Attributes().InsertString(childKey, value)
		}
	case common.MapStr:
		for kk, vv := range value {
			dfsAddLogAttributes(childKey, kk, vv, Logs, parentInConfig || configCache[childKey])
		}
	case []interface{}:
		/*
			Scenarios where fields have multi values.  For example -
			fields:
				type: grok
				pattern:
				- '%{DATESTAMP:time} %{LOGLEVEL:severity} %{WORD:class}:%{NUMBER:line} - %{GREEDYDATA:data}'
				- '%{DATESTAMP_RFC2822:time} %{LOGLEVEL:severity} %{GREEDYDATA:data}'
		*/
		attrValueArr := pdata.NewAttributeValueArray()
		for _, vv := range value {
			switch vvalue := vv.(type) {
			case string:
				attrValueArr.ArrayVal().AppendEmpty().SetStringVal(vvalue)
			}
		}
		if !(configCache[childKey] || parentInConfig) {
			Logs.Attributes().Insert(childKey, attrValueArr)
		}
	}
}

// AddResourceAttributes adds resource attributes to resource
func AddResourceAttributes(childKey string, value interface{}, Logs *pdata.Logs) {
	resourceLog := GetResourceLog(Logs)
	switch vvalue := value.(type) {
	case string:
		resourceLog.Resource().Attributes().InsertString(childKey, vvalue)
	case pdata.AttributeValue:
		resourceLog.Resource().Attributes().Insert(childKey, vvalue)
	}
}

// AddLogAttributes adds log attributes to log record
func AddLogAttributes(childKey string, value interface{}, Logs *pdata.Logs) {
	logRecord := GetLogRecord(Logs)
	switch vvalue := value.(type) {
	case string:
		logRecord.Attributes().InsertString(childKey, vvalue)
	case pdata.AttributeValue:
		logRecord.Attributes().Insert(childKey, vvalue)
	}
}

/*
		dfsAddAttributes populates log record attributes and log attributes depending on,
		configuration provided in filebeat.yml, the operation takes O(N) time, and O(1) space
		where N is the total number of attributes present in a beat event.
		eg configuration:
		codec.otlp:
	        resources:
	              - k8s.cluster.name
	              - k8s.replicaset.name
	              - k8s.pod
		in this case k8s.cluster.name, k8s.replicaset.name, and everything inside k8s.pod subtree will go to resource attributes,
		remaining things will go to log attributes
*/
func dfsAddAttributes(name, key string, val interface{}, Logs *pdata.Logs, parentInConfig bool) {
	var childKey string
	if name != "" {
		if key != "" {
			childKey = name + "." + key
		} else {
			childKey = name
		}
	} else {
		childKey = key
	}
	switch value := val.(type) {
	case string:
		if configCache[childKey] || parentInConfig {
			AddResourceAttributes(childKey, value, Logs)
		} else {
			AddLogAttributes(childKey, value, Logs)
		}
	case common.MapStr:
		for kk, vv := range value {
			dfsAddAttributes(childKey, kk, vv, Logs, parentInConfig || configCache[childKey])
		}
	case []interface{}:
		/*
			Scenarios where fields have multi values.  For example -
			fields:
				type: grok
				pattern:
				- '%{DATESTAMP:time} %{LOGLEVEL:severity} %{WORD:class}:%{NUMBER:line} - %{GREEDYDATA:data}'
				- '%{DATESTAMP_RFC2822:time} %{LOGLEVEL:severity} %{GREEDYDATA:data}'
		*/
		attrValueArr := pdata.NewAttributeValueArray()
		for _, vv := range value {
			switch vvalue := vv.(type) {
			case string:
				attrValueArr.ArrayVal().AppendEmpty().SetStringVal(vvalue)
			}
		}
		if configCache[childKey] || parentInConfig {
			AddResourceAttributes(childKey, attrValueArr, Logs)
		} else {
			AddLogAttributes(childKey, attrValueArr, Logs)
		}
	}
}

/*
PopulateAttributes populates instrumentationLibrary data, logbody, and move different attributes to
either log attributes or resource attributes depending on config provided in filebeat.yml
*/
func PopulateAttributes(Logs *pdata.Logs, event *beat.Event) {
	for fields, val := range event.Fields {
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

// GetResourceLog retrieves last ResourceLog from ResourceLog slice
func GetResourceLog(Logs *pdata.Logs) pdata.ResourceLogs {
	return Logs.ResourceLogs().At(Logs.ResourceLogs().Len() - 1)
}

// GetInstrumentationLibraryLog retrieves last InstrumentationLibraryLog from InstrumentationLibraryLog slice
func GetInstrumentationLibraryLog(Logs *pdata.Logs) pdata.InstrumentationLibraryLogs {
	rs := GetResourceLog(Logs)
	return rs.InstrumentationLibraryLogs().At(rs.InstrumentationLibraryLogs().Len() - 1)
}

// GetLogRecord retrieves LogRecord from InstrumentationLibraryLogs
func GetLogRecord(Logs *pdata.Logs) pdata.LogRecord {
	lr := GetInstrumentationLibraryLog(Logs)
	return lr.Logs().At(lr.Logs().Len() - 1)
}

// AddResourceLogs adds a ResourceLog slice with an empty ResourceLog
func AddResourceLogs(ld *pdata.Logs) pdata.ResourceLogs {
	ld.ResourceLogs().AppendEmpty()
	return ld.ResourceLogs().At(ld.ResourceLogs().Len() - 1)
}

// AddInstrumentationLibraryLogs adds an InstrumentationLibraryLogsSlice with an empty InstrumentationLibraryLogs
func AddInstrumentationLibraryLogs(rs *pdata.ResourceLogs) pdata.InstrumentationLibraryLogs {
	rs.InstrumentationLibraryLogs().AppendEmpty()
	return rs.InstrumentationLibraryLogs().At(rs.InstrumentationLibraryLogs().Len() - 1)
}

// PopulateInstrumentationLibrary populates the instrumentation library with the details of the instrumentation agent present in beat event
func PopulateInstrumentationLibrary(ld *pdata.InstrumentationLibraryLogs, agent interface{}) {
	if agentMap, ok := agent.(common.MapStr); ok {
		for k, v := range agentMap {
			if val, ok := v.(string); ok {
				if k == "type" {
					ld.InstrumentationLibrary().SetName(val)
				} else if k == "version" {
					ld.InstrumentationLibrary().SetVersion(val)
				}
			}
		}
	}
}

// AddLogRecord appends LogRecordSlice with an empty LogRecord
func AddLogRecord(lr *pdata.InstrumentationLibraryLogs) pdata.LogRecord {
	lr.Logs().AppendEmpty()
	return lr.Logs().At(lr.Logs().Len() - 1)
}

// PopulateTimeStamp populates timestamp to otlp LogRecord
func PopulateTimeStamp(lr *pdata.LogRecord, event *beat.Event) {
	lr.SetTimestamp(pdata.NewTimestampFromTime(getTimestampFromEvent(event)))
}

// PopulateLogRecord populates the LogRecord with the collected log message of the application present in beat event
func PopulateLogRecord(lr *pdata.LogRecord, val string) {
	lr.Body().SetStringVal(val)
}

// helper function to extract timestamp from beat event and populates it into LogRecord
func getTimestampFromEvent(event *beat.Event) time.Time {
	var timestamp time.Time
	if val, err := event.GetValue("@timestamp"); err == nil {
		timestamp = val.(time.Time)
	}
	return timestamp
}

// MarshalLogs marshals the otlp packet to protobuf
func MarshalLogs(ld pdata.Logs) ([]byte, error) {
	logsMarshaler := otlp.NewProtobufLogsMarshaler()
	return logsMarshaler.MarshalLogs(ld)
}

// UnmarshalLogs marshals the otlp packet to protobuf
func UnmarshalLogs(buf []byte) (pdata.Logs, error) {
	logsUnmarshaler := otlp.NewProtobufLogsUnmarshaler()
	return logsUnmarshaler.UnmarshalLogs(buf)
}

// LogsSize returns size of the OTLP logs data structure
func LogsSize(l pdata.Logs) int {
	logsSizer := otlp.NewProtobufLogsMarshaler().(pdata.LogsSizer)
	return logsSizer.LogsSize(l)
}
