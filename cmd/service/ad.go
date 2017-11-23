package main

import (
	"flag"
	"os"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"jf/adservice/pkg/myservice"
	"jf/adservice/pkg/myendpoint"
)

const (
	defaultPort = "80"
)

func main() {
	var (
		port     = envString("PORT", defaultPort)
		httpAddr = flag.String("http.addr", ":"+port, "HTTP listen Ports")
	)
	flag.Parse()

	//Create a single logger, which we'll use and give to other components.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	}

	// Determine which tracer to use. We'll pass the tracer to all the
	// components that use it, as a dependency.
	// We have more choice for tracing , like zipkin, appdash, lightstep, jaeger
	var tracer stdopentracing.Tracer
	{
		tracer = stdopentracing.GlobalTracer()
	}

	//Create the (sparse) metrics we'll use in the service. They, too, are
	// dependencies that we pass to components that use them.
	var counts metrics.Counter
	{
		//Business-level metrics
		counts = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace:"service",
			Subsystem:"adservice",
			Name: "banners_get",
			Help: "Total count of banners get via the GetBanners method.",
		}, []string{})
	}
	var duration metrics.Histogram
	{
		duration = prometheus.NewSummary(stdprometheus.SummaryOpts{
			Namespace: "service",
			Subsystem: "adservice",
			Name: "request_duration_seconds",
			Help: "Request duration in seconds"
		}, []string{"method", "success"})
	}

	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())


	//Build the layers of the service "onion" from the inside out. First, the
	//business logic service; then , the set of endpoints that wrap the service;
	//and finnally, a serise of concrete transport adapters. The adapters, like
	//the HTTP handler or the gRPC server, are the bridge between Go Kit and
	//interfaces that the transports expect. Note that we're not binding
	//them to ports or anything yet; we'll do that next.
	var (
		adservice = myservice.New(logger, counts)
		endpoints = myendpoint.New(adservice, logger, duration, tracer)

	)




}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}
