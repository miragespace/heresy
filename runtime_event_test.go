package heresy

import (
	"bytes"
	"strings"
	"testing"
)

func TestBinaryResponse(t *testing.T) {
	for _, body := range []string{
		`new Uint8Array([0,128,255,65]).buffer`,
		`new Uint8Array([99,0,128,255,65,99]).subarray(1,5)`,
	} {
		t.Run(body, func(t *testing.T) {
			_, srv := testRuntime(t, `registerEventHandler(e => e.respondWith(new Response(`+body+`)))`, 1)
			status, _, got := testResponse(t, srv, "GET", "/", nil)
			if status != 200 || !bytes.Equal(got, []byte{0, 128, 255, 65}) {
				t.Fatalf("got status=%d body=%x", status, got)
			}
		})
	}
}

func TestLargeRequestBody(t *testing.T) {
	_, srv := testRuntime(t, `registerEventHandler(async e => e.respondWith(new Response(await e.request.arrayBuffer())))`, 1)
	want := []byte(strings.Repeat("first chunk!", 1000) + strings.Repeat("last chunk!", 1000))
	status, _, body := testResponse(t, srv, "POST", "/", want)
	if status != 200 || !bytes.Equal(body, want) {
		t.Fatalf("large body changed: status=%d got %d bytes, want %d", status, len(body), len(want))
	}
}

func TestBYOBOffset(t *testing.T) {
	_, srv := testRuntime(t, `registerEventHandler(async e => {
		const reader = e.request.body.getReader({mode: "byob"});
		const {value} = await reader.read(new Uint8Array(new ArrayBuffer(16), 4, 8));
		e.respondWith(new Response(value));
	})`, 1)
	status, _, body := testResponse(t, srv, "POST", "/", []byte("12345678"))
	if status != 200 || string(body) != "12345678" {
		t.Fatalf("BYOB view changed: %d %q", status, body)
	}
}

func TestRequestMetadata(t *testing.T) {
	_, srv := testRuntime(t, `registerEventHandler(e => {
		if (!e.request.headers.has("x-test") || e.request.headers.get("X-TEST") !== "present") throw Error("missing header");
		e.respondWith(new Response(e.request.url));
	})`, 1)
	status, _, body := testResponse(t, srv, "GET", "/path?query=1", nil)
	if status != 200 || string(body) != srv.URL+"/path?query=1" {
		t.Fatalf("got %d %q", status, body)
	}
}

func TestRequestBody(t *testing.T) {
	for _, method := range []string{"POST", "OPTIONS"} {
		t.Run(method, func(t *testing.T) {
			_, srv := testRuntime(t, `registerEventHandler(async e => e.respondWith(new Response(await e.request.text())))`, 1)
			status, _, body := testResponse(t, srv, method, "/", []byte("request body"))
			if status != 200 || string(body) != "request body" {
				t.Fatalf("got %d %q", status, body)
			}
		})
	}
}
