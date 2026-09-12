package httpapi_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/httpapi"
)

func TestSessionCookiePolicySetupLoginLogout(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		configured, tls, forwarded, want bool
	}{
		{name: "local HTTP"},
		{name: "direct TLS", tls: true, want: true},
		{name: "trusted HTTPS termination", configured: true, want: true},
		{name: "untrusted forwarded header", forwarded: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := setupTestContext(t, httpapi.RuntimePaths{SecureSessionCookie: tc.configured})
			var cookie *http.Cookie
			for _, methodPath := range [][2]string{{"POST", "/api/v1/setup"}, {"POST", "/api/v1/session"}, {"DELETE", "/api/v1/session"}} {
				req := httptest.NewRequest(methodPath[0], methodPath[1], strings.NewReader(`{"username":"fixture-admin","password":"fixture-password-only-123"}`))
				if tc.tls {
					req.TLS = &tls.ConnectionState{}
				}
				if tc.forwarded {
					req.Header.Set("X-Forwarded-Proto", "https")
					req.Header.Set("Forwarded", "proto=https")
				}
				if cookie != nil {
					req.AddCookie(cookie)
				}
				if methodPath[0] == "DELETE" {
					session, err := ctx.authService.Authenticate(t.Context(), cookie.Value)
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("X-CSRF-Token", session.CSRFToken)
				}
				rec := ctx.doRequest(req, false, false)
				if rec.Code >= 400 {
					t.Fatalf("%s %s status %d", methodPath[0], methodPath[1], rec.Code)
				}
				cookies := rec.Result().Cookies()
				if len(cookies) != 1 {
					t.Fatal("expected session cookie")
				}
				cookie = cookies[0]
				if cookie.Secure != tc.want || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
					t.Fatal("incorrect cookie policy")
				}
				if methodPath[0] == "DELETE" && cookie.MaxAge != -1 {
					t.Fatal("logout did not expire cookie")
				}
			}
		})
	}
}
