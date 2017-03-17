package cachestatic

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createTestHandler() http.Handler {
	i := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i == 0 {
			i++
			w.Header().Set("Content-Type", "text/plain; utf-8")
			w.Header().Set("Custom-Header", "0")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
			return
		}
		w.Header().Set("Custom-Header", "1")
		w.WriteHeader(200)
		w.Write([]byte("Not first response"))
	})
}

func TestCachestatic(t *testing.T) {
	ts := httptest.NewServer(New()(createTestHandler()))
	defer ts.Close()

	verify := func(resp *http.Response, err error) {
		if err != nil {
			t.Fatalf("expected error to be nil; got %v", err)
		}
		if resp.Header.Get("Content-Type") != "text/plain; utf-8" {
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

func BenchmarkCacheStatic(b *testing.B) {
	ts := httptest.NewServer(New()(createTestHandler()))
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
