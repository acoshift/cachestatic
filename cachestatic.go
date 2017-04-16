package cachestatic

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"sync"
)

type item struct {
	data   []byte
	header http.Header
}

// Config type
type Config struct {
	// Skip returns true to skip cache
	Skip func(*http.Request) bool
}

// DefaultConfig is the default config
var DefaultConfig = Config{
	Skip: func(r *http.Request) bool { return false },
}

// New creates new cachestatic middleware
func New(config Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if config.Skip != nil {
		c.Skip = config.Skip
	}

	var (
		l     = &sync.RWMutex{}
		cache = make(map[string]*item)
	)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.Skip(r) {
				h.ServeHTTP(w, r)
				return
			}

			p := r.Method + ":" + path.Clean(r.URL.Path)
			l.RLock()
			if c := cache[p]; c != nil {
				l.RUnlock()
				wh := w.Header()
				for k, vs := range c.header {
					wh[k] = vs
				}
				buf := bytes.NewReader(c.data)
				io.Copy(w, buf)
				return
			}
			l.RUnlock()
			cw := &responseWriter{
				ResponseWriter: w,
				cache:          &bytes.Buffer{},
			}
			h.ServeHTTP(cw, r)
			t := &item{
				header: cw.Header(),
				data:   cw.cache.Bytes(),
			}
			l.Lock()
			cache[p] = t
			l.Unlock()
		})
	}
}
