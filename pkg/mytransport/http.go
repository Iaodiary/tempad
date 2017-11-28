package mytransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"

	"jf/adservice/pkg/myendpoint"
	"jf/adservice/pkg/myservice"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predefined paths.
func NewHTTPHandler(endpoints myendpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorLogger(logger),
	}
	m := http.NewServeMux()
	m.Handle("/banners", httptransport.NewServer(
		endpoints.GetBannersEndpoint,
		decodeHTTPGetBannersRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetBanners", logger)))...,
	))
	return m
}

// NewHTTPClient returns an MyService backed by an HTTP server living at the
// remote instance. We expect instance to come from a service discovery system,
// so likely of the form "host:port". We bake-in certain middlewares,
// implementing the client library pattern.
func NewHTTPClient(instance string, tracer stdopentracing.Tracer, logger log.Logger) (myservice.AdService, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	// We construct a single ratelimiter middleware, to limit the total outgoing
	// QPS from this client to all methods on the remote instance. We also
	// construct per-endpoint circuitbreaker middlewares to demonstrate how
	// that's done, although they could easily be combined into a single breaker
	// for the entire remote instance, too.
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var getBannersEndpoint endpoint.Endpoint
	{
		getBannersEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/banners"),
			encodeHTTPGenericRequest,
			decodeHTTPGetBannersResponse,
			httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
		).Endpoint()
		getBannersEndpoint = opentracing.TraceClient(tracer, "GetBanners")(getBannersEndpoint)
		getBannersEndpoint = limiter(getBannersEndpoint)
		getBannersEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetBanners",
			Timeout: 30 * time.Second,
		}))(getBannersEndpoint)
	}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return myendpoint.Set{
		GetBannersEndpoint: getBannersEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	//if err != nil {
	//	return http.StatusBadRequest
	//}
	return http.StatusBadRequest

	//return http.StatusInternalServerError
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

func decodeHTTPGetBannersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req myendpoint.GetBannersRequest
	var err error

	if "GET" == r.Method {
		req.ClientID, err = strconv.Atoi(r.FormValue("client"))
		req.Size = r.FormValue("size")
	} else {
		err = json.NewDecoder(r.Body).Decode(&req)
		if nil != err {
			err = errors.New("json-decode:" + err.Error())
		}
	}
	return req, err
}

// decodeHTTPSumResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded sum response from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func decodeHTTPGetBannersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp myendpoint.GetBannersResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

//encodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that
//Json-encodes any reponse to response result.
func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeHTTPHtmlResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Context-Type", "text/html; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
