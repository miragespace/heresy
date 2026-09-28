package heresy

import (
	"testing"
)

func TestRuntimeReadinessAndFailedReload(t *testing.T) {
	rt, srv := testRuntime(t, "", 2)
	status, _, _ := testResponse(t, srv, "GET", "/", nil)
	if status != 503 {
		t.Fatalf("empty runtime returned %d", status)
	}
	if err := rt.LoadScript("good.js", `registerEventHandler(e => e.respondWith(new Response("working")))`, false); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`function {`, `throw Error("bad initialization")`} {
		if err := rt.LoadScript("bad.js", bad, true); err == nil {
			t.Fatal("expected reload error")
		}
		for i := 0; i < 2; i++ {
			status, _, body := testResponse(t, srv, "GET", "/", nil)
			if status != 200 || string(body) != "working" {
				t.Fatalf("failed reload replaced working handler: %d %q", status, body)
			}
		}
	}
}
