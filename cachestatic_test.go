package cachestatic

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	mgzip "github.com/acoshift/gzip"
	"github.com/acoshift/middleware"
)

func createTestHandler() http.Handler {
	i := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Custom-Header", "0")
			w.WriteHeader(200)
			return
		}
		if i == 0 {
			i++
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Custom-Header", "0")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Custom-Header", "1")
		w.WriteHeader(200)
		w.Write([]byte("Not first response"))
	})
}

func createStaticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Custom-Header", "0")
			w.WriteHeader(200)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Custom-Header", "0")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
}

func TestCachestatic(t *testing.T) {
	ts := httptest.NewServer(New(DefaultConfig)(createTestHandler()))
	defer ts.Close()

	verify := func(resp *http.Response, err error) {
		if err != nil {
			t.Fatalf("expected error to be nil; got %v", err)
		}
		if resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Fatalf("invalid Content-Type; got %v", resp.Header.Get("Content-Type"))
		}
		if resp.Header.Get("Custom-Header") != "0" {
			t.Fatalf("invalid Custom-Header; got %v", resp.Header.Get("Content-Type"))
		}
		defer resp.Body.Close()
		r, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read response body error; %v", err)
		}
		if bytes.Compare(r, []byte("OK")) != 0 {
			t.Fatalf("invalid response body; got %v", string(r))
		}
	}

	verify(http.Get(ts.URL))
	verify(http.Get(ts.URL))
	verify(http.Get(ts.URL))
	verify(http.Get(ts.URL))
}

func TestWithGzip(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}

	wg := &sync.WaitGroup{}

	verify := func(resp *http.Response, err error) {
		defer wg.Done()
		if err != nil {
			t.Fatalf("expected error to be nil; got %v", err)
		}
		if resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Fatalf("invalid Content-Type; got %v", resp.Header.Get("Content-Type"))
		}
		if resp.Header.Get("Custom-Header") != "0" {
			t.Fatalf("invalid Custom-Header; got %v", resp.Header.Get("Custom-Header"))
		}
		defer resp.Body.Close()
		if resp.Request.Method == http.MethodHead {
			return
		}
		if resp.Header.Get("Content-Encoding") == "gzip" && resp.Request.Header.Get("Accept-Encoding") != "gzip" {
			t.Fatalf("request non gzip; got gzip response")
		}
		var body io.Reader
		if resp.Header.Get("Content-Encoding") == "gzip" {
			body, _ = gzip.NewReader(resp.Body)
		} else {
			body = resp.Body
		}
		r, err := ioutil.ReadAll(body)
		if err != nil {
			t.Fatalf("read response body error; %v", err)
		}
		if bytes.Compare(r, []byte("OK")) != 0 {
			t.Fatalf("invalid response body; got %v", string(r))
		}
	}

	var h http.Handler

	l := 100
	run := func() {
		ts := httptest.NewServer(h)
		defer ts.Close()
		wg.Add(l)
		for i := 0; i < l; i++ {
			req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
			if rand.Int()%2 == 0 {
				req.Header.Set("Accept-Encoding", "gzip")
			}
			if rand.Int()%2 == 0 {
				req.Method = http.MethodHead
			}
			go verify(client.Do(req))
		}
		wg.Wait()
	}

	// default config
	h = middleware.Chain(
		mgzip.New(mgzip.Config{Level: mgzip.BestSpeed}),
		New(DefaultConfig),
	)(createTestHandler())
	run()

	// with skip gzip
	h = middleware.Chain(
		New(Config{
			Skipper: func(r *http.Request) bool {
				return !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
			},
		}),
		mgzip.New(mgzip.Config{Level: mgzip.BestSpeed}),
	)(createStaticHandler())
	run()

	// with index gzip
	h = middleware.Chain(
		New(Config{
			Indexer: func(r *http.Request) string {
				p := r.Method
				if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
					p += ":gzip"
				}
				p += ":" + path.Clean(r.URL.Path)
				return p
			},
		}),
		mgzip.New(mgzip.Config{Level: mgzip.BestSpeed}),
	)(createStaticHandler())
	run()
}

func BenchmarkCacheStatic(b *testing.B) {
	ts := httptest.NewServer(New(DefaultConfig)(createTestHandler()))
	defer ts.Close()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(ts.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkNoCacheStatic(b *testing.B) {
	ts := httptest.NewServer(createTestHandler())
	defer ts.Close()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(ts.URL)
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}
