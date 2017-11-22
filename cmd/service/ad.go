package main

import (
	"flag"
	"os"

	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"
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

}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}
