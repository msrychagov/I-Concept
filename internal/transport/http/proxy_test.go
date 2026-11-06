package httptransport

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

func TestSecureBehindProxy(t *testing.T) {
	forwarded := httptest.NewRequest("POST", "/", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	direct := httptest.NewRequest("POST", "/", nil)
	direct.TLS = &tls.ConnectionState{}
	plain := httptest.NewRequest("POST", "/", nil)
	trusted := &Handler{config: Config{TrustProxy: true}}
	untrusted := &Handler{config: Config{}}
	if !trusted.secure(forwarded) {
		t.Error("trusted proxy with https must give a secure cookie")
	}
	if untrusted.secure(forwarded) {
		t.Error("X-Forwarded-Proto must be ignored without TrustProxy")
	}
	if !untrusted.secure(direct) {
		t.Error("direct TLS must give a secure cookie")
	}
	if trusted.secure(plain) {
		t.Error("plain http must not give a secure cookie")
	}
}
