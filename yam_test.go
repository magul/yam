// Unittests for Yam

package yam

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type TestRoute struct {
	Path    string
	Methods []string
	Handler func(http.ResponseWriter, *http.Request)
}

type TestRequest struct {
	Path   string
	Method string
}

type TestResponse struct {
	Status int
	Body   []byte
}

// Table Driven Tests
var tests = []struct {
	// Request to make
	config   *Config
	route    TestRoute
	req      TestRequest
	response TestResponse
}{
	// Root Route
	{
		NewConfig(),
		TestRoute{"/", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("root"))
		}},
		TestRequest{"/", "GET"},
		TestResponse{http.StatusOK, []byte("root")},
	},
	// Simplest Route
	{
		NewConfig(),
		TestRoute{"/foo", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Path))
		}},
		TestRequest{"/foo", "GET"},
		TestResponse{http.StatusOK, []byte("/foo")},
	},
	// 404 Handling
	{
		NewConfig(),
		TestRoute{"/foo", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Path))
		}},
		TestRequest{"/bar", "GET"},
		TestResponse{http.StatusNotFound, nil},
	},
	// 405 Handling
	{
		NewConfig(),
		TestRoute{"/foo", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Path))
		}},
		TestRequest{"/foo", "POST"},
		TestResponse{http.StatusMethodNotAllowed, nil},
	},
	// Pattern Matching & Added to Query
	{
		NewConfig(),
		TestRoute{"/foo/:bar", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Query().Get(":bar")))
		}},
		TestRequest{"/foo/bar", "GET"},
		TestResponse{http.StatusOK, []byte("bar")},
	},
	// Deep Nesting
	{
		NewConfig(),
		TestRoute{"/a/b/c/:d/e/f/g/:h/i/j/:k", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(r.URL.Query().Get(":d")))
			w.Write([]byte(r.URL.Query().Get(":h")))
			w.Write([]byte(r.URL.Query().Get(":k")))
		}},
		TestRequest{"/a/b/c/f/e/f/g/o/i/j/o", "GET"},
		TestResponse{http.StatusOK, []byte("foo")},
	},
}

func TestTables(t *testing.T) {
	for _, test := range tests {
		y := New()
		y.Config = test.config
		r := y.Route(test.route.Path)
		for _, method := range test.route.Methods {
			r.Add(method, http.HandlerFunc(test.route.Handler))
		}

		s := httptest.NewServer(y)
		req, _ := http.NewRequest(test.req.Method, s.URL+test.req.Path, nil)
		c := &http.Client{}
		res, _ := c.Do(req)

		if res.StatusCode != test.response.Status {
			t.Errorf("Status was %v, should be %v", res.StatusCode, test.response.Status)
		}

		body, _ := ioutil.ReadAll(res.Body)
		if !bytes.Equal(body, test.response.Body) {
			t.Errorf("Body was %v, should be %v", string(body[:]), string(test.response.Body[:]))
		}

		s.Close()
	}
}

func TestTraceDisabled(t *testing.T) {
	mux := New()
	mux.Route("/")

	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("TRACE", s.URL, nil)
	c := &http.Client{}
	res, _ := c.Do(req)

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Status was %v, should be %v", res.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestTraceEnabled(t *testing.T) {
	mux := New()
	mux.Config.Trace = true
	mux.Route("/")

	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("TRACE", s.URL, nil)
	c := &http.Client{}
	res, _ := c.Do(req)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Status was %v, should be %v", res.StatusCode, http.StatusOK)
	}

	body, _ := ioutil.ReadAll(res.Body)
	url, _ := url.Parse(s.URL)
	expected := []byte(fmt.Sprintf("TRACE / HTTP/1.1\r\nHost: %s\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\n\r\n", url.Host))
	if !bytes.Equal(body, expected) {
		t.Errorf("Body was\n%vshould be:\n%v", string(body[:]), string(expected[:]))
	}
}
