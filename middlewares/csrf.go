package middlewares

import (
	"net/http"
	"net/url"
	"strings"

	conf "github.com/muety/wakapi/config"
	routeutils "github.com/muety/wakapi/routes/utils"
)

// CsrfHeader must be present on every state-changing request to the UI API.
// A cross-origin form cannot set headers at all, and a cross-origin fetch that
// tries triggers a preflight which gets no CORS response, so the request never
// fires. That is the whole mechanism.
const CsrfHeader = "X-Wakapi-Request"

// formContentTypes are the only media types a cross-origin HTML form can produce.
// Rejecting them means a forged form submission cannot reach a mutation even if
// it somehow got past the header check.
var formContentTypes = []string{
	"application/x-www-form-urlencoded",
	"multipart/form-data",
	"text/plain",
}

// CsrfMiddleware protects cookie-authenticated mutations on the UI API.
//
// It is deliberately scoped to that router only. Mounting it globally would break
// every editor plugin, because heartbeat ingestion posts with an API key and knows
// nothing about this header.
type CsrfMiddleware struct {
	config *conf.Config
}

func NewCsrfMiddleware() *CsrfMiddleware {
	return &CsrfMiddleware{config: conf.Get()}
}

func (m *CsrfMiddleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.ServeHTTP(w, r, h.ServeHTTP)
	})
}

func (m *CsrfMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if isSafeMethod(r.Method) {
		next(w, r)
		return
	}

	if r.Header.Get(CsrfHeader) == "" {
		routeutils.RespondJSONError(w, http.StatusForbidden, "missing "+CsrfHeader+" header")
		return
	}

	if ct := r.Header.Get("Content-Type"); isFormContentType(ct) {
		routeutils.RespondJSONError(w, http.StatusForbidden, "unsupported content type for this endpoint")
		return
	}

	// Sec-Fetch-Site is set by the browser and cannot be forged from script.
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		routeutils.RespondJSONError(w, http.StatusForbidden, "cross-site request rejected")
		return
	}

	if origin := r.Header.Get("Origin"); origin != "" && !m.isAllowedOrigin(origin, r) {
		routeutils.RespondJSONError(w, http.StatusForbidden, "origin not allowed")
		return
	}

	next(w, r)
}

// isAllowedOrigin accepts the configured public url and the host the request was
// actually addressed to. The latter covers reverse proxies and accessing the
// instance under an alternative hostname.
func (m *CsrfMiddleware) isAllowedOrigin(origin string, r *http.Request) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	if strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	if pub := m.config.Server.PublicNetUrl; pub != nil && pub.Host != "" {
		return strings.EqualFold(parsed.Host, pub.Host)
	}
	return false
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func isFormContentType(contentType string) bool {
	// strip any parameters, e.g. "multipart/form-data; boundary=..."
	if idx := strings.IndexByte(contentType, ';'); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	for _, ct := range formContentTypes {
		if contentType == ct {
			return true
		}
	}
	return false
}
