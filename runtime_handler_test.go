package heresy

import (
	"strings"
	"testing"
)

func TestHandlerResponses(t *testing.T) {
	for _, tt := range []struct {
		name, script, body string
		status             int
	}{
		{"event", `registerEventHandler(e => e.respondWith(new Response("hello")))`, "hello", 200},
		{"event fallthrough", `registerEventHandler(() => {})`, "next", 200},
		{"promised response", `registerEventHandler(e => e.respondWith(Promise.resolve(new Response("promised"))))`, "promised", 200},
		{"express fallthrough", `registerExpressHandler(({next}) => next())`, "next", 200},
		{"express empty", `registerExpressHandler(() => {})`, "", 204},
		{"express status only", `registerExpressHandler(({res}) => res.status(418))`, "", 418},
		{"express response", `registerExpressHandler(({res}) => res.status(201).send("created"))`, "created", 201},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, srv := testRuntime(t, tt.script, 1)
			// Exercise recycled contexts as well as their first allocation.
			for i := 0; i < 3; i++ {
				status, _, body := testResponse(t, srv, "GET", "/", nil)
				if status != tt.status || string(body) != tt.body {
					t.Fatalf("got %d %q, want %d %q", status, body, tt.status, tt.body)
				}
			}
		})
	}
}

func TestHandlerErrors(t *testing.T) {
	for _, script := range []string{
		`registerEventHandler(() => { throw Error("failure") })`,
		`registerEventHandler(async () => { throw Error("failure") })`,
		`registerEventHandler(e => e.respondWith(Promise.reject()))`,
		`registerEventHandler(e => e.respondWith(new Response("invalid", {status: 0})))`,
		`registerExpressHandler(() => { throw Error("failure") })`,
	} {
		t.Run(script, func(t *testing.T) {
			_, srv := testRuntime(t, script, 1)
			status, _, _ := testResponse(t, srv, "GET", "/", nil)
			if status != 500 {
				t.Fatalf("got status %d, want 500", status)
			}
		})
	}
}

func TestResponseThenThrow(t *testing.T) {
	_, srv := testRuntime(t, `registerEventHandler(e => { e.respondWith(new Response("sent")); throw Error("after response") })`, 1)
	status, _, body := testResponse(t, srv, "GET", "/", nil)
	if status != 200 || !strings.HasPrefix(string(body), "sent") {
		t.Fatalf("got %d %q", status, body)
	}
}
