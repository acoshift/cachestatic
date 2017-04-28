package cachestatic

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/acoshift/header"
	"github.com/acoshift/middleware"
)

// Config type
type Config struct {
	Skipper     middleware.Skipper
	Indexer     Indexer
	Invalidator chan string
}

// DefaultConfig is the default config
var DefaultConfig = Config{
	Skipper:     middleware.DefaultSkipper,
	Indexer:     DefaultIndexer,
	Invalidator: nil,
}

// New creates new cachestatic middleware
func New(c Config) func(http.Handler) http.Handler {
	if c.Skipper == nil {
		c.Skipper = DefaultConfig.Skipper
	}
	if c.Indexer == nil {
		c.Indexer = DefaultConfig.Indexer
	}
	if c.Invalidator == nil {
		c.Invalidator = DefaultConfig.Invalidator
	}

	var (
		l     = &sync.RWMutex{}
		cache = make(map[string]*item)
	)

	if c.Invalidator != nil {
		go func() {
			for {
				select {
				case p := <-c.Invalidator:
					l.Lock()
					if p == "" {
						cache = make(map[string]*item)
					} else {
						delete(cache, p)
					}
					l.Unlock()
				}
			}
		}()
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.Skipper(r) {
				h.ServeHTTP(w, r)
				return
			}

			p := c.Indexer(r)
			l.RLock()
			if c := cache[p]; c != nil {
				l.RUnlock()
				wh := w.Header()
				for k, vs := range c.header {
					wh[k] = vs
				}

				// check Last-Modified
				if !c.modTime.IsZero() {
					if ts := r.Header.Get(header.IfModifiedSince); len(ts) > 0 {
						t, _ := http.ParseTime(ts)
						if c.modTime.Equal(t) {
							wh.Del(header.ContentType)
							wh.Del(header.ContentLength)
							wh.Del(header.AcceptRanges)
							w.WriteHeader(http.StatusNotModified)
							return
						}
					}
				}

				io.Copy(w, bytes.NewReader(c.data))
				return
			}
			l.RUnlock()
			cw := &responseWriter{
				ResponseWriter: w,
				cache:          &bytes.Buffer{},
			}
			h.ServeHTTP(cw, r)

			// cache only status ok
			if cw.code == http.StatusOK {
				l.Lock()
				cache[p] = createItem(cw)
				l.Unlock()
			}
		})
	}
}
