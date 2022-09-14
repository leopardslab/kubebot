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
	"context"
	beat2 "github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/otlp"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"google.golang.org/grpc"
	"net"
	"testing"
)

//struct to provide grpc test server
type mockReceiver struct {
	srv          *grpc.Server
}

//struct which inherit the Logserver interface from otlpgrpc package
type mockLogsReceiver struct {
	mockReceiver
}

func (r *mockLogsReceiver) Export(ctx context.Context, req otlpgrpc.LogsRequest) (otlpgrpc.LogsResponse, error) {
	return otlpgrpc.NewLogsResponse(), nil
}

//otlpLogsReceiverOnGRPCServer() gives grpc test server to send the data
func otlpLogsReceiverOnGRPCServer(ln net.Listener) *mockLogsReceiver {
	rcv := &mockLogsReceiver{
		mockReceiver: mockReceiver{
			srv: grpc.NewServer(),
		},
	}
	// Now run it as a gRPC server
	otlpgrpc.RegisterLogsServer(rcv.srv, rcv)
	go func() {
		_ = rcv.srv.Serve(ln)
	}()

	return rcv
}

func TestGrpcClient_Connect(t *testing.T) {
	ln, err := net.Listen("tcp", defaultGRPCEndpooint)
	assert.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()

	cfg := defaultOLTPConfig()
	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	logger := logp.NewLogger("grpc_test")
	enc := otlp.New(beat.Version, otlp.Config{})

	c, err := NewGrpcClient(cfg, logger, outputs.NewNilObserver(), enc, ln.Addr().String())
	err = c.Connect()
	assert.NoError(t, err,"failed to connect to testing grpc server")
}

func TestGrpcClient_Publish(t *testing.T) {
	ln, err := net.Listen("tcp", defaultGRPCEndpooint)
	assert.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()

	cfg := defaultOLTPConfig()
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"},},
	}
	batch := outest.NewBatch(events...)

	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	logger := logp.NewLogger("grpc_test")
	enc := otlp.New(beat.Version, otlp.Config{})

	c, err := NewGrpcClient(cfg, logger, outputs.NewNilObserver(), enc, ln.Addr().String())
	err = c.Connect()
	assert.NoError(t, err,"failed to connect to testing grpc server")
	ctx := context.Background()
	err = c.Publish(ctx, batch)
	assert.Equal(t, err, nil)
}

func TestGrpcClient_PublishInBatch(t *testing.T) {
	ln, err := net.Listen("tcp", defaultGRPCEndpooint)
	assert.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()

	cfg := defaultOLTPConfig()
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"},},
	}
	batch := outest.NewBatch(events...)

	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	logger := logp.NewLogger("grpc_test")
	enc := otlp.New(beat.Version, otlp.Config{})

	c, err := NewGrpcClient(cfg, logger, outputs.NewNilObserver(), enc, ln.Addr().String())
	err = c.Connect()
	assert.NoError(t, err,"failed to connect to testing grpc server")
	ctx := context.Background()
	err =  c.PublishInBatch(ctx, batch.Events())
	assert.NoError(t, err, "Error while calling PublishEvents()")
}

func TestGrpcClient_PublishEvents(t *testing.T) {
	ln, err := net.Listen("tcp", defaultGRPCEndpooint)
	assert.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()

	cfg := defaultOLTPConfig()
	events := []beat2.Event{
		{Fields: common.MapStr{"sample_key": "sample_value"},},
	}
	batch := outest.NewBatch(events...)

	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	logger := logp.NewLogger("grpc_test")
	enc := otlp.New(beat.Version, otlp.Config{})

	c, err := NewGrpcClient(cfg, logger, outputs.NewNilObserver(), enc, ln.Addr().String())
	err = c.Connect()
	assert.NoError(t, err,"failed to connect to testing grpc server")
	ctx := context.Background()
	failedEvents, err :=  c.PublishEvents(ctx, batch.Events())
	assert.NoError(t, err, "Error while calling PublishEvents()")
	assert.Equal(t,0, len(failedEvents))
}

func TestGrpcClient_PublishEvent(t *testing.T) {
	ln, err := net.Listen("tcp", defaultGRPCEndpooint)
	assert.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)
	rcv := otlpLogsReceiverOnGRPCServer(ln)
	defer rcv.srv.GracefulStop()

	beat := beat2.Info{Version: "1.0", IndexPrefix: "mock"}
	logger := logp.NewLogger("grpc_test")
	enc := otlp.New(beat.Version, otlp.Config{})
	cfg := defaultOLTPConfig()

	c, err := NewGrpcClient(cfg, logger, outputs.NewNilObserver(), enc, ln.Addr().String())
	err = c.Connect()
	assert.NoError(t, err,"failed to connect to testing grpc server")

	event := publisher.Event{
		Content: beat2.Event{
			Fields: common.MapStr{"sample_key": "sample_value"},
		}}
	ctx := context.Background()
	err = c.PublishEvent(ctx, event)
	assert.NoError(t, err, "error while calling PublishEvent()")
}
