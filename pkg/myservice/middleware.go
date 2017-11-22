package myservice

import (
	"context"

	"jf/adservice/models"

	"github.com/go-kit/kit/log"
)

//Middleware describe a service (as opposed to endpoint) endpoint
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
	return GetBanners(ctx, clientID, size)
}
