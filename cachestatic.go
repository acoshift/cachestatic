package cachestatic

import (
	"bytes"
	"net/http"
	"sync"
)

// New creates new cachestatic middleware
func New() func(http.Handler) http.Handler {
	var (
		l      = &sync.RWMutex{}
		cache  []byte
		header http.Header
		code   = 200
		cached = false
	)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.RLock()
			if cached {
				wh := w.Header()
				for k, vs := range header {
					for _, v := range vs {
						wh.Set(k, v)
					}
				}
				w.WriteHeader(code)
				w.Write(cache)
				l.RUnlock()
				return
			}
			l.RUnlock()
			l.Lock()
			defer l.Unlock()
			cw := &responseWriter{
				ResponseWriter: w,
				cache:          &bytes.Buffer{},
			}
			h.ServeHTTP(cw, r)
			header = cw.Header()
			code = cw.code
			cache = cw.cache.Bytes()
			cached = true
		})
	}
}
