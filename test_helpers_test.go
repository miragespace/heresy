package heresy

import (
	"bytes"
	"fmt"
	"go.miragespace.co/heresy/extensions/kv"
	_ "go.miragespace.co/heresy/extensions/kv/memory"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testRuntime(t *testing.T, script string, shards int) (*Runtime, *httptest.Server) {
	t.Helper()
	kvs := kv.NewKVManager()
	if err := kvs.Configure("test", "memory"); err != nil {
		t.Fatal(err)
	}
	rt, err := NewRuntime(zap.NewNop(), kvs, shards)
	if err != nil {
		t.Fatal(err)
	}
	if script != "" {
		if err := rt.LoadScript("test.js", script, false); err != nil {
			rt.Stop(true)
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(rt.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "next")
	})))
	srv.Client().Timeout = 3 * time.Second
	t.Cleanup(func() {
		rt.Stop(true)
		srv.Close()
	})
	return rt, srv
}

func testResponse(t *testing.T, srv *httptest.Server, method, path string, body []byte) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Test", "present")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, data
}
