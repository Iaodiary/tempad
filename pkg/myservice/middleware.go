package myservice

import (
	"context"

	"jf/adservice/models"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

//Middleware describe a AdService (as opposed to endpoint) endpoint
type Middleware func(AdService) AdService

//LoggingMiddleware takes a logger as a denpendency
//and returns a AdServiceMiddleware
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next AdService) AdService {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   AdService
}

func (mw loggingMiddleware) GetBanners(ctx context.Context, clientID int, size string) (banners []*models.Banner, err error) {
	defer func() {
		mw.logger.Log("method", "GetBanners", "clientID", clientID, "size", size, "banners", banners, "err", err)
	}()
	return mw.next.GetBanners(ctx, clientID, size)
}

//InstrumentingMiddleware returns a service middleware that instruments
//the banners num returned over the lifetime ofthe service
func InstrumentingMiddleware(counts metrics.Counter) Middleware {
	return func(next AdService) AdService {
		return instrumentingMiddleware{
			counts: counts,
			next:   next,
		}
	}
}

type instrumentingMiddleware struct {
	counts metrics.Counter
	next   AdService
}

func (mw instrumentingMiddleware) GetBanners(ctx context.Context, clientID int, size string) ([]*models.Banner, error) {
	v, err := mw.next.GetBanners(ctx, clientID, size)
	mw.counts.Add(float64(len(v)))
	return v, err
}
