package realip_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bakins/go-real-ip"
	h "github.com/bakins/test-helpers"
)

func TestNew(t *testing.T) {
	_, err := realip.New([]string{"X-Forwarded-For"}, []string{"127.0.0.0/8"})
	h.Ok(t, err)
}

func TestNewFail(t *testing.T) {
	_, err := realip.New([]string{"X-Forwarded-For"}, []string{"0.0/8"})
	h.Assert(t, err != nil, "new should fail")
}

func newRequest(method, url string, remoteAddr, xff string) *http.Request {
	r, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	r.RemoteAddr = remoteAddr
	r.Header.Set("X-Forwarded-For", xff)
	return r
}

func TestHandlerPass(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "Hello World\n")
	})

	ri, err := realip.New([]string{"X-Forwarded-For"}, []string{"8.8.0.0/16"})
	h.Ok(t, err)

	request := newRequest("GET", "/foo", "8.8.8.8:9876", "64.63.62.61")
	recorder := httptest.NewRecorder()
	riHandler := ri.Handler(handler)

	riHandler.ServeHTTP(recorder, request)

	h.Equals(t, 200, recorder.Code)

	h.Equals(t, "64.63.62.61:9876", request.RemoteAddr)
}

// probably need a test setup helper, but just copy paste for now

func TestHandlerFail(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "Hello World\n")
	})

	ri, err := realip.New([]string{"X-Forwarded-For"}, []string{"8.8.0.0/16"})
	h.Ok(t, err)

	request := newRequest("GET", "/foo", "9.8.8.8:9876", "64.63.62.61")
	recorder := httptest.NewRecorder()
	riHandler := ri.Handler(handler)

	riHandler.ServeHTTP(recorder, request)

	h.Equals(t, 200, recorder.Code)

	h.Equals(t, "9.8.8.8:9876", request.RemoteAddr)
}

func ExampleNew() {
	// simple handler
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// realip will rewrite RemoteAddr
		fmt.Fprintf(w, "Hello: %s\n", r.RemoteAddr)
	})

	// we only will rewrite RemoteAddr if connection is from our "trusted" loadbalancers
	// we allow localhost for testing
	ri, err := realip.New([]string{"X-Forwarded-For", "X-Real-IP"}, []string{"8.8.0.0/16", "127.0.0.0/8"})
	if err != nil {
		// do something with error
	}

	// this wraps all HTTP requests
	http.ListenAndServe(":8080", ri.Handler(http.DefaultServeMux))
}
