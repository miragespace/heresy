package heresy

import (
	"testing"
)

func TestKV(t *testing.T) {
	_, srv := testRuntime(t, `registerEventHandler(async e => {
		await e.kv.test.put("key", "value");
		const value = await e.kv.test.get("key");
		const deleted = await e.kv.test.del("key");
		const missing = await e.kv.test.get("key");
		e.respondWith(new Response(JSON.stringify([value, deleted, missing])));
	})`, 2)
	status, _, body := testResponse(t, srv, "GET", "/", nil)
	if status != 200 || string(body) != `["value",true,null]` {
		t.Fatalf("got %d %q", status, body)
	}
}
