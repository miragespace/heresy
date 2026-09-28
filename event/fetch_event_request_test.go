package event

import (
	"net/http/httptest"
	"testing"
)

func TestRequestURL(t *testing.T) {
	for _, url := range []string{"http://example.test/path?q=1", "https://example.test/path?q=1"} {
		request := httptest.NewRequest("GET", url, nil)
		if got := makeUrl(request); got != url {
			t.Errorf("makeUrl() = %q, want %q", got, url)
		}
	}
}
