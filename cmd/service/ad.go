package main

import (
	"flag"
	"fmt"
	"jf/adservice/pkg/mytransport"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"jf/adservice/pkg/myendpoint"
	"jf/adservice/pkg/myservice"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/oklog/oklog/pkg/group"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort = "80"
)

func main() {
	fs := flag.NewFlagSet("ad", flag.ExitOnError)
	var (
		debugAddr = fs.String("debug.addr", ":8080", "Debug and metrics listen address")
		port      = envString("PORT", defaultPort)
		httpAddr  = flag.String("http.addr", ":"+port, "HTTP listen Ports")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	fs.Parse(os.Args[1:])

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
			Namespace: "service",
			Subsystem: "adservice",
			Name:      "banners_get",
			Help:      "Total count of banners get via the GetBanners method.",
		}, []string{})
	}
	var duration metrics.Histogram
	{
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "service",
			Subsystem: "adservice",
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds",
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
		adservice   = myservice.New(logger, counts)
		endpoints   = myendpoint.New(adservice, logger, duration, tracer)
		httpHandler = mytransport.NewHTTPHandler(endpoints, tracer, logger)
	)

	// Now we're to the part of the func main where we want to start actually
	// running things, like servers bound to listeners to receive connections.
	//
	// The method is the same for each component: add a new actor to the group
	// struct, which is a combination of 2 anonymous functions: the first
	// function actually runs the component, and the second function should
	// interrupt the first function and cause it to return. It's in these
	// functions that we actually bind the Go kit server/handler structs to the
	// concrete transports and run them.
	//
	// Putting each component into its own block is mostly for aesthetics: it
	// clearly demarcates the scope in which each listener/socket may be used.
	var g group.Group
	{
		debugListener, err := net.Listen("tcp", *debugAddr)
		if err != nil {
			logger.Log("transport", "debug/HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "debug/HTTP", "addr", *debugAddr)
			return http.Serve(debugListener, http.DefaultServeMux)
		}, func(error) {
			debugListener.Close()
		})
	}
	{
		// The HTTP listener mounts the Go kit HTTP handler we created.
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(httpListener, httpHandler)
		}, func(error) {
			httpListener.Close()
		})
	}

	{
		// This function just sits and waits for ctrl-C.
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	logger.Log("exit", g.Run())
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
