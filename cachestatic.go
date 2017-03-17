package cachestatic

import (
	"bytes"
	"net/http"
	"path"
	"sync"
)

type item struct {
	data   []byte
	header http.Header
}

// New creates new cachestatic middleware
func New() func(http.Handler) http.Handler {
	var (
		l     = &sync.RWMutex{}
		cache = make(map[string]*item)
	)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := path.Clean(r.URL.Path)
			l.RLock()
			if c := cache[p]; c != nil {
				l.RUnlock()
				wh := w.Header()
				for k, vs := range c.header {
					for _, v := range vs {
						wh.Set(k, v)
					}
				}
				w.Write(c.data)
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
