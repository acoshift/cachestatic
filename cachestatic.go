package cachestatic

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/acoshift/header"
	"github.com/acoshift/middleware"
)

// Config type
type Config struct {
	Skipper middleware.Skipper
	Indexer Indexer
}

// DefaultConfig is the default config
var DefaultConfig = Config{
	Skipper: middleware.DefaultSkipper,
	Indexer: DefaultIndexer,
}

// New creates new cachestatic middleware
func New(config Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if config.Skipper != nil {
		c.Skipper = config.Skipper
	}
	if config.Indexer != nil {
		c.Indexer = config.Indexer
	}

	var (
		l     = &sync.RWMutex{}
		cache = make(map[string]*item)
	)

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
				if !c.lastModified.IsZero() {
					if ts := r.Header.Get(header.IfModifiedSince); len(ts) > 0 {
						t, _ := time.Parse(time.RFC1123, ts)
						if c.lastModified.Equal(t) {
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
