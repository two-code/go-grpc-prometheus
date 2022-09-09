// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

// gRPC Prometheus monitoring interceptors for server-side gRPC.

package grpc_prometheus

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	// DefaultServerMetrics is the default instance of ServerMetrics. It is
	// intended to be used in conjunction the default Prometheus metrics
	// registry.
	DefaultServerMetrics = NewServerMetrics()

	// UnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
	UnaryServerInterceptor = DefaultServerMetrics.UnaryServerInterceptor()

	// StreamServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Streaming RPCs.
	StreamServerInterceptor = DefaultServerMetrics.StreamServerInterceptor()

	defaultServerMetricsPromRegistration   = map[prom.Registerer]struct{}{}
	defaultServerMetricsPromRegistrationMu sync.Mutex
)

// Register takes a gRPC server and pre-initializes all counters to 0. This
// allows for easier monitoring in Prometheus (no missing metrics), and should
// be called *after* all services have been registered with the server. This
// function acts on the DefaultServerMetrics variable.
func Register(server *grpc.Server) {
	DefaultServerMetrics.InitializeMetrics(server)
}

// EnableHandlingTimeHistogram turns on recording of handling time
// of RPCs. Histogram metrics can be very expensive for Prometheus
// to retain and query. This function acts on the DefaultServerMetrics.
func EnableHandlingTimeHistogram(opts ...HistogramOption) {
	DefaultServerMetrics.EnableHandlingTimeHistogram(opts...)
}

func RegisterDefaultServerMetricsWithRegisterer(reg prom.Registerer) (alreadyRegistered bool, err error) {
	defaultServerMetricsPromRegistrationMu.Lock()

	if _, ok := defaultServerMetricsPromRegistration[reg]; ok {
		defaultServerMetricsPromRegistrationMu.Unlock()

		alreadyRegistered = true

		return
	}

	registeredMetrics := make([]prom.Collector, 0, 5)

	defer func() {
		if err != nil {
			for i := 0; i < len(registeredMetrics); i++ {
				reg.Unregister(registeredMetrics[i])
			}

			defaultServerMetricsPromRegistrationMu.Unlock()

			return
		}

		defaultServerMetricsPromRegistration[reg] = struct{}{}
		defaultServerMetricsPromRegistrationMu.Unlock()
	}()

	if err = reg.Register(DefaultServerMetrics.serverStartedCounter); err != nil {
		return
	}
	registeredMetrics = append(registeredMetrics, DefaultServerMetrics.serverStartedCounter)

	if err = reg.Register(DefaultServerMetrics.serverHandledCounter); err != nil {
		return
	}
	registeredMetrics = append(registeredMetrics, DefaultServerMetrics.serverHandledCounter)

	if err = reg.Register(DefaultServerMetrics.serverStreamMsgReceived); err != nil {
		return
	}
	registeredMetrics = append(registeredMetrics, DefaultServerMetrics.serverStreamMsgReceived)

	if err = reg.Register(DefaultServerMetrics.serverStreamMsgSent); err != nil {
		return
	}
	registeredMetrics = append(registeredMetrics, DefaultServerMetrics.serverStreamMsgSent)

	if DefaultServerMetrics.serverHandledHistogramEnabled && DefaultServerMetrics.serverHandledHistogram != nil {
		if err = reg.Register(DefaultServerMetrics.serverHandledHistogram); err != nil {
			return
		}
		registeredMetrics = append(registeredMetrics, DefaultServerMetrics.serverHandledHistogram)
	}

	return
}
