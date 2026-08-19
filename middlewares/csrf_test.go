package middlewares

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/muety/wakapi/config"
)

func csrfTestSubject(t *testing.T, publicUrl string) *CsrfMiddleware {
	t.Helper()
	cfg := config.Empty()
	cfg.Server.PublicUrl = publicUrl
	if parsed, err := url.Parse(publicUrl); err == nil {
		cfg.Server.PublicNetUrl = parsed
	}
	config.Set(cfg)
	return NewCsrfMiddleware()
}

// runCsrf sends a request through the middleware and reports the status plus whether
// the wrapped handler was reached.
func runCsrf(sut *CsrfMiddleware, r *http.Request) (int, bool) {
	reached := false
	rec := httptest.NewRecorder()
	sut.ServeHTTP(rec, r, func(http.ResponseWriter, *http.Request) { reached = true })
	return rec.Code, reached
}

func TestCsrfMiddleware_AllowsSafeMethodsWithoutHeader(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		r := httptest.NewRequest(method, "https://wakapi.example.org/api/ui/summary", nil)
		status, reached := runCsrf(sut, r)

		assert.True(t, reached, "%s must pass through", method)
		assert.Equal(t, http.StatusOK, status)
	}
}

func TestCsrfMiddleware_RejectsMutationWithoutHeader(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	r := httptest.NewRequest(http.MethodPost, "https://wakapi.example.org/api/ui/aliases", nil)
	r.Header.Set("Content-Type", "application/json")

	status, reached := runCsrf(sut, r)

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestCsrfMiddleware_AcceptsMutationWithHeader(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	r := httptest.NewRequest(http.MethodPost, "https://wakapi.example.org/api/ui/aliases", nil)
	r.Header.Set(CsrfHeader, "1")
	r.Header.Set("Content-Type", "application/json")

	status, reached := runCsrf(sut, r)

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, status)
}

// A cross-origin HTML form can only produce these media types, so they are refused
// even when a header is somehow present.
func TestCsrfMiddleware_RejectsFormContentTypes(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	for _, contentType := range []string{
		"application/x-www-form-urlencoded",
		"multipart/form-data; boundary=----abc",
		"text/plain;charset=UTF-8",
		"TEXT/PLAIN",
	} {
		r := httptest.NewRequest(http.MethodPost, "https://wakapi.example.org/api/ui/aliases", nil)
		r.Header.Set(CsrfHeader, "1")
		r.Header.Set("Content-Type", contentType)

		status, reached := runCsrf(sut, r)

		assert.False(t, reached, "content type %q must be refused", contentType)
		assert.Equal(t, http.StatusForbidden, status)
	}
}

func TestCsrfMiddleware_RejectsForeignOrigin(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	r := httptest.NewRequest(http.MethodPost, "https://wakapi.example.org/api/ui/aliases", nil)
	r.Header.Set(CsrfHeader, "1")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://evil.example.com")

	status, reached := runCsrf(sut, r)

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestCsrfMiddleware_AcceptsOwnOrigins(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	// the configured public url, and the host the request was actually addressed to
	for _, origin := range []string{"https://wakapi.example.org", "http://localhost:3001"} {
		r := httptest.NewRequest(http.MethodPost, "http://localhost:3001/api/ui/aliases", nil)
		r.Header.Set(CsrfHeader, "1")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)

		status, reached := runCsrf(sut, r)

		assert.True(t, reached, "origin %q must be allowed", origin)
		assert.Equal(t, http.StatusOK, status)
	}
}

func TestCsrfMiddleware_RejectsCrossSiteFetchMetadata(t *testing.T) {
	sut := csrfTestSubject(t, "https://wakapi.example.org")

	r := httptest.NewRequest(http.MethodPost, "https://wakapi.example.org/api/ui/aliases", nil)
	r.Header.Set(CsrfHeader, "1")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Sec-Fetch-Site", "cross-site")

	status, reached := runCsrf(sut, r)

	assert.False(t, reached)
	assert.Equal(t, http.StatusForbidden, status)
}
