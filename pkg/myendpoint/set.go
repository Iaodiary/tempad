package myendpoint

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

// Set collects all endpoints that compose an ad service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	GetAdEndpoint endpoint.Endpoint
}

// New returned a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(svc myservice.Service, logger log.Logger, duration metrics.Histogram, trace stdopentracing.Tracer) Set {
	var getAdEndpoint endpoint.Endpoint
	{

	}
	return Set{GetAdEndpoint: getAdEndpoint}
}
