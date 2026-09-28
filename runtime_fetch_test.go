package heresy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchSemantics(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "preserved")
		w.WriteHeader(http.StatusCreated)
		io.Copy(w, r.Body)
	}))
	t.Cleanup(upstream.Close)
	script := fmt.Sprintf(`registerEventHandler(async e => {
		const resp = await e.fetch(%q, {method: "POST", body: new Uint8Array([0,128,255,65]).buffer});
		e.respondWith(resp);
	}, {fetch: true})`, upstream.URL)
	_, srv := testRuntime(t, script, 1)
	status, headers, body := testResponse(t, srv, "GET", "/", nil)
	if status != http.StatusCreated || headers.Get("X-Upstream") != "preserved" || !bytes.Equal(body, []byte{0, 128, 255, 65}) {
		t.Fatalf("fetch changed response: status=%d headers=%v body=%x", status, headers, body)
	}
}

func TestFetchCancellation(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
	}))
	t.Cleanup(upstream.Close)
	_, srv := testRuntime(t, fmt.Sprintf(`registerEventHandler(async e => e.respondWith(await e.fetch(%q)), {fetch:true})`, upstream.URL), 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := srv.Client().Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fetch did not start")
	}
	cancel()
	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("outbound fetch did not cancel")
	}
	<-done
}
