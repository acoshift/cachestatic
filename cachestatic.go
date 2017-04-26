package cachestatic

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"sync"

	"github.com/acoshift/middleware"
)

type item struct {
	data   []byte
	header http.Header
}

// Config type
type Config struct {
	Skipper middleware.Skipper
}

// DefaultConfig is the default config
var DefaultConfig = Config{
	Skipper: middleware.DefaultSkipper,
}

// New creates new cachestatic middleware
func New(config Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if config.Skipper != nil {
		c.Skipper = config.Skipper
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
			l.Lock()
			cache[p] = &item{
				header: cw.h,
				data:   cw.cache.Bytes(),
			}
			l.Unlock()
		})
	}
}
