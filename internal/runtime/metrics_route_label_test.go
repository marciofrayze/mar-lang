package runtime

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestMetricsRouteLabelUsesRealPathForUnknownRoutes(t *testing.T) {
	requireSQLite3(t)

	r := mustNewRuntimeFromSource(t, filepath.Join(t.TempDir(), "metrics-route-label.db"), `
app TodoApi

entity Todo {
  title: String
  authorize all when true
}
`)

	cases := []struct {
		path string
		want string
	}{
		{path: "/favicon.ico", want: "/favicon.ico"},
		{path: "/apple-touch-icon.png", want: "/apple-touch-icon.png"},
		{path: "/auth/not-a-real-route", want: "/auth/not-a-real-route"},
		{path: "/actions/not/valid", want: "/actions/not/valid"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest("GET", tc.path, nil)
		got := r.metricsRouteLabel(req)
		if got != tc.want {
			t.Fatalf("metricsRouteLabel(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
