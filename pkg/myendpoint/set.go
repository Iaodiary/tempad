package myendpoint

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"

	stdopentracing "github.com/opentracing/opentracing-go"

	"jf/adservice/models"
	"jf/adservice/pkg/myservice"
)

// Set collects all endpoints that compose an ad service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	GetBannersEndpoint endpoint.Endpoint
}

// New returned a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(ads myservice.AdService, logger log.Logger, duration metrics.Histogram, trace stdopentracing.Tracer) Set {
	var getBannersEndpoint endpoint.Endpoint
	{
		//
		getBannersEndpoint = MakeGetBannersEndpoint(ads)
		//getBannersEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))(getBannersEndpoint)
		getBannersEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))(getBannersEndpoint)
		getBannersEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(getBannersEndpoint)
		getBannersEndpoint = opentracing.TraceServer(trace, "GetBanners")(getBannersEndpoint)
		getBannersEndpoint = InstrumentingMiddleware(duration.With("method", "GetBanners"))(getBannersEndpoint)
	}
	return Set{GetBannersEndpoint: getBannersEndpoint}
}

//GetBanners implements the service interface, so Set may be used as a service.
//This is primarily userful in the context of a client library
func (s Set) GetBanners(ctx context.Context, clientID int, size string) ([]*models.Banner, error) {
	resp, err := s.GetBannersEndpoint(ctx, GetBannersRequest{ClientID: clientID, Size: size})
	if err != nil {
		return []*models.Banner{}, err
	}
	response := resp.(GetBannersResponse)
	return response.V, response.Err
}

//MakeGetBannersEndpoint constructs a Sum endpoint wrapping the service
func MakeGetBannersEndpoint(s myservice.AdService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(GetBannersRequest)
		v, err := s.GetBanners(ctx, req.ClientID, req.Size)
		return GetBannersResponse{V: v, Err: err}, nil
	}
}

//GetBannersRequest service standard request
type GetBannersRequest struct {
	ClientID int
	Size     string
}

//GetBannersResponse service standard response
type GetBannersResponse struct {
	V   []*models.Banner
	Err error
}
