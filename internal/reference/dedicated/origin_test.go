//go:build reference

package dedicated

import (
	"net/http/httptest"
	"testing"
)

func TestConfiguredOriginRemainsRestricted(t *testing.T) {
	r := NewRuntime("ds1", "", "", "", "")
	defer r.Close()
	r.Origin = "http://localhost:18080"
	for _, tc := range []struct {
		origin string
		status int
	}{
		{"http://localhost:18080", 400}, // Origin admitted; missing Upgrade headers.
		{"http://localhost:8080", 403},
		{"https://evil.invalid", 403},
	} {
		q := httptest.NewRequest("GET", "/v1/ws", nil)
		q.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		r.Handler().ServeHTTP(w, q)
		if w.Code != tc.status {
			t.Fatalf("origin=%s status=%d want=%d", tc.origin, w.Code, tc.status)
		}
	}
}
