package myendpoint

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/ratelimit"

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
		getBannersEndpoint = MakeGetBannersEndpoint(ads)
		getBannersEndpoint = ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 1))(getBannersEndpoint)
	}
	return Set{GetBannersEndpoint: getBannersEndpoint}
}

func MakeGetBannersEndpoint(s myservice.AdService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(GetBannersRequest)
		v, err := s.GetBanners(ctx, req.ClientID, req.Size)
		return GetBannersResponse{V: v, Err: err}, nil
	}
}

type GetBannersRequest struct {
	ClientID int
	Size     string
}

type GetBannersResponse struct {
	V   []*models.Banner
	Err error
}
