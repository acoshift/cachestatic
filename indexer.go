package cachestatic

import (
	"net/http"
	"path"
	"strings"

	"github.com/acoshift/header"
)

// Indexer is the function for map request to cache index
type Indexer func(*http.Request) string

// DefaultIndexer is the default indexer function
func DefaultIndexer(r *http.Request) string {
	return r.Method + ":" + path.Clean(r.URL.Path)
}

// EncodingIndexer creates an indexer for adds encoding into index
func EncodingIndexer(encoding string) Indexer {
	return func(r *http.Request) string {
		p := r.Method
		if strings.Contains(r.Header.Get(header.AcceptEncoding), encoding) {
			p += ":" + encoding
		}
		p += ":" + path.Clean(r.URL.Path)
		return p
	}
}
