package middlewares

import (
	"net/http"
	"strings"

	conf "github.com/muety/wakapi/config"
)

// spaAssetPrefix is where Vite writes content-hashed chunks. Those file names change
// whenever their content does, so they can be cached indefinitely.
const spaAssetPrefix = "/assets/app/static/"

type StaticCacheMiddleware struct {
	config *conf.Config
}

func NewStaticCacheMiddleware() *StaticCacheMiddleware {
	return &StaticCacheMiddleware{config: conf.Get()}
}

func (m *StaticCacheMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.ServeHTTP(w, r, h.ServeHTTP)
	})
}

func (m *StaticCacheMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Never cache in dev - the build:watch loop rewrites these files in place.
	if !m.config.IsDev() && strings.HasPrefix(r.URL.Path, spaAssetPrefix) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	next(w, r)
}
