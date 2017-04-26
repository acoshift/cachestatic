package cachestatic

import (
	"net/http"
	"time"

	"github.com/acoshift/header"
)

type item struct {
	data         []byte
	header       http.Header
	lastModified time.Time
}

func createItem(w *responseWriter) *item {
	it := item{
		data:   w.cache.Bytes(),
		header: w.h,
	}
	if w.h != nil {
		if v := w.h.Get(header.LastModified); len(v) > 0 {
			it.lastModified, _ = time.Parse(time.RFC1123, v)
		}
	}
	return &it
}
