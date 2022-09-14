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
	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc/credentials/insecure"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/otlp"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"go.opentelemetry.io/collector/model/otlpgrpc"
	"go.opentelemetry.io/collector/model/pdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GrpcClient connection
type GrpcClient struct {
	endpoint         string
	log              *logp.Logger
	observer         outputs.Observer
	clientConn       *grpc.ClientConn
	metadata         metadata.MD
	callOptions      []grpc.CallOption
	connConfig       *OtlpConfig
	codec            codec.Codec
	logMax           int
	codecLogCount    int
	exporterLogCount int
	tickTime         int
	enableDebugTimer bool
	timerRunning     bool
}

// NewGrpcClient constructor for GrpcClient
func NewGrpcClient(
	config *OtlpConfig,
	log *logp.Logger,
	observer outputs.Observer,
	codec codec.Codec,
	endpoint string,
) (*GrpcClient, error) {
	client := &GrpcClient{
		endpoint:   endpoint,
		log:        log,
		connConfig: config,
		metadata:   metadata.New(config.Headers),
		codec:      codec,
		observer:   observer,
		callOptions: []grpc.CallOption{
			grpc.WaitForReady(config.GRPC.WaitForReady),
		},
		logMax:           config.LogMax,
		tickTime:         getTickTime(config.TickTime),
		enableDebugTimer: config.EnableDebugTimer,
		timerRunning:     false,
	}
	return client, nil
}

func (c *GrpcClient) startTimer(tickTime int) {
	ticker := time.NewTicker(time.Duration(tickTime) * time.Second)
	for {
		select {
		case <-ticker.C:
			c.log.Debug("Total logs processed by codec in ", strconv.Itoa(tickTime), " seconds ", " is ", c.codecLogCount)
			c.log.Debug("Total logs exported by gRPC exporter in ", strconv.Itoa(tickTime), " seconds ", " is ", c.exporterLogCount)
			c.codecLogCount = 0
			c.exporterLogCount = 0
		}
	}
}

// Connect function to connect to OTLP endpoint via GRPC protocol
func (c *GrpcClient) Connect() error {

	var opts []grpc.DialOption

	tlsCfg, err := tlscommon.LoadTLSConfig(c.connConfig.TLS)
	if err != nil {
		return err
	}
	tlsDialOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	if tlsCfg != nil {
		tlsDialOption = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg.ToConfig()))
	}
	opts = append(opts, tlsDialOption)
	opts = append(opts, grpc.WithBlock())

	if c.connConfig.GRPC.KeepAlive != nil {
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.connConfig.GRPC.KeepAlive.Time,
			Timeout:             c.connConfig.GRPC.KeepAlive.Timeout,
			PermitWithoutStream: c.connConfig.GRPC.KeepAlive.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}
	opts = append(opts, grpc.WithUnaryInterceptor(newUnaryClientInterceptor(c.connConfig.Headers)))
	c.clientConn, err = grpc.Dial(c.endpoint, opts...)
	if err != nil {
		c.log.Errorf("Error while creating grpc client")
		return err
	}
	return nil
}

func newUnaryClientInterceptor(headers map[string]string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md := metadata.New(headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// loadTLSCredentials(): load the cert from
func loadTLSCredentials(tlsConfig *tlscommon.Config) credentials.TransportCredentials {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile(tlsConfig.Certificate.Certificate)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		panic(errors.New("Failed to add server CA's certificate"))
	}

	return credentials.NewTLS(&tls.Config{RootCAs: certPool})
}

// Close closes GRPC connection
func (c *GrpcClient) Close() error {
	err := c.clientConn.Close()
	if err != nil {
		c.log.Errorf("Error while closing the grpc connection")
	}
	c.clientConn = nil
	return nil
}

// String return connection name
func (c *GrpcClient) String() string {
	return "OTLP/GRPC(" + c.endpoint + ")"
}

// Publish publishes to endpoint using GRPC protocol
func (c *GrpcClient) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	if c.connConfig.BatchPublish {
		err := c.PublishInBatch(ctx, events)
		if err != nil {
			batch.RetryEvents(events)
		} else {
			batch.ACK()
		}
		return err
	} else {
		c.observer.NewBatch(len(events))
		dropped, err := c.PublishEvents(ctx, events)
		if err != nil {
			return err
		}
		if len(dropped) > 0 {
			batch.RetryEvents(dropped)
			c.observer.Acked(len(events) - len(dropped))
		} else {
			batch.ACK()
		}
	}
	return nil
}

// PublishEvents publish all events
func (c *GrpcClient) PublishEvents(ctx context.Context, events []publisher.Event) ([]publisher.Event, error) {
	if c.metadata.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, c.metadata)
	}
	var err = error(nil)
	var dropped []publisher.Event

	for _, event := range events {
		err = c.PublishEvent(ctx, event)
		if err != nil {
			dropped = append(dropped, event)
		}
	}
	if len(dropped) > 0 {
		return dropped, err
	}
	return nil, nil
}

// PublishInBatch makes one network call per batch of beat events
func (c *GrpcClient) PublishInBatch(ctx context.Context, events []publisher.Event) error {
	if c.enableDebugTimer && !c.timerRunning {
		go c.startTimer(c.tickTime)
		c.timerRunning = true
	}
	t1 := time.Now()
	serializedEventInterface, err := c.codec.EncodeBatch("", events)
	if err != nil {
		c.observer.WriteError(err)
		c.log.Error("Error while encoding to OTLP")
		return err
	}
	LogCounter += len(events)
	c.codecLogCount += len(events)
	// log time taken by each event after every LogMax events
	if LogCounter >= c.logMax {
		c.log.Debugf("Total time taken by Codec for translation of %s logs is ", len(events), time.Since(t1))
	}
	log := serializedEventInterface.(*pdata.Logs)
	t1 = time.Now()
	logsRequest := otlpgrpc.NewLogsRequest()
	logsRequest.SetLogs(*log)
	logsClient := otlpgrpc.NewLogsClient(c.clientConn)
	_, err = logsClient.Export(ctx, logsRequest, c.callOptions...)
	if err != nil {
		c.observer.WriteError(err)
		return err
	}

	c.exporterLogCount += len(events)
	// log time taken by each event after every LogMax events
	if LogCounter >= c.logMax {
		c.log.Debugf("Total time taken by Exporter for exporting %s logs is ", len(events), time.Since(t1))
		LogCounter = 0
	}

	logsSize := otlp.LogsSize(*log)
	c.observer.WriteBytes(logsSize + 1)
	return nil
}

// PublishEvent publishes a single event
func (c *GrpcClient) PublishEvent(ctx context.Context, event publisher.Event) error {
	LogCounter++
	if c.enableDebugTimer && !c.timerRunning {
		go c.startTimer(c.tickTime)
		c.timerRunning = true
	}
	t1 := time.Now()
	serializedEvent, err := c.codec.Encode("", &event.Content)
	if err != nil {
		c.observer.WriteError(err)
		c.log.Error("Error while encoding to OTLP")
		return err
	}

	c.codecLogCount++
	// log time taken by each event after every LogMax events
	if LogCounter == c.logMax {
		c.log.Debug("Total time taken by Codec for translation of single log = ", time.Since(t1))
	}

	t1 = time.Now()
	//UnmarshalLogs() is required to convert serializedEvent from []bytes to pdata.Logs
	log, err := otlp.UnmarshalLogs(serializedEvent)
	if err != nil {
		c.observer.WriteError(err)
		c.log.Error("Error while unmarshalling")
		return err
	}
	logsRequest := otlpgrpc.NewLogsRequest()
	logsRequest.SetLogs(log)
	logsclient := otlpgrpc.NewLogsClient(c.clientConn)
	_, err = logsclient.Export(ctx, logsRequest, c.callOptions...)
	if err != nil {
		c.observer.WriteError(err)
		return err
	}
	c.observer.WriteBytes(len(serializedEvent) + 1)
	c.exporterLogCount++
	// log time taken by each event after every LogMax events
	if LogCounter == c.logMax {
		c.log.Debug("Total time taken by Exporter for exporting single log = ", time.Since(t1))
	} else if LogCounter > c.logMax {
		LogCounter = 0
	}
	return nil
}
