package heresy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dop251/goja"
)

type requestResult struct {
	status int
	body   string
	err    error
}

func requestAsync(srv *httptest.Server) <-chan requestResult {
	done := make(chan requestResult, 1)
	go func() {
		resp, err := srv.Client().Get(srv.URL)
		if err != nil {
			done <- requestResult{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		done <- requestResult{status: resp.StatusCode, body: string(body), err: err}
	}()
	return done
}

func awaitSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func onInstance(t *testing.T, inst *runtimeInstance, fn func(*goja.Runtime)) {
	t.Helper()
	done := make(chan struct{})
	if !inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		fn(vm)
		close(done)
	}) {
		t.Fatal("instance already stopped")
	}
	awaitSignal(t, done, "VM callback")
}

func TestGracefulReload(t *testing.T) {
	rt, srv := testRuntime(t, `registerEventHandler(async e => {
		testStarted();
		await new Promise(resolve => { globalThis.testResume = resolve });
		e.respondWith(new Response("old"));
	})`, 1)
	old := rt.shards[0].Load()
	started := make(chan struct{})
	onInstance(t, old, func(vm *goja.Runtime) { vm.Set("testStarted", func() { close(started) }) })
	pending := requestAsync(srv)
	awaitSignal(t, started, "old handler")
	if err := rt.LoadScript("new.js", `registerEventHandler(e => e.respondWith(new Response("new")))`, false); err != nil {
		t.Fatal(err)
	}
	status, _, body := testResponse(t, srv, "GET", "/", nil)
	if status != 200 || string(body) != "new" {
		t.Fatalf("new generation returned %d %q", status, body)
	}
	onInstance(t, old, func(vm *goja.Runtime) { vm.RunString("testResume()") })
	result := <-pending
	if result.err != nil || result.status != 200 || result.body != "old" {
		t.Fatalf("old request failed during graceful reload: %+v", result)
	}
	awaitSignal(t, old.stopped, "retired instance cleanup")
	if old.eventLoop.RunOnLoop(func(*goja.Runtime) {}) {
		t.Fatal("retired loop still accepts jobs")
	}
}

func TestForcedReload(t *testing.T) {
	for _, script := range []string{
		`registerEventHandler(async () => { testStarted(); await new Promise(() => {}) })`,
		`registerExpressHandler(async () => { testStarted(); await new Promise(() => {}) })`,
		`registerEventHandler(e => { testStarted(); e.respondWith(new Promise(() => {})) })`,
	} {
		t.Run(script, func(t *testing.T) {
			rt, srv := testRuntime(t, `setInterval(() => {}, 10); `+script, 1)
			old := rt.shards[0].Load()
			started := make(chan struct{})
			onInstance(t, old, func(vm *goja.Runtime) { vm.Set("testStarted", func() { close(started) }) })
			pending := requestAsync(srv)
			awaitSignal(t, started, "pending handler")
			if err := rt.LoadScript("new.js", `registerEventHandler(e => e.respondWith(new Response("new")))`, true); err != nil {
				t.Fatal(err)
			}
			result := <-pending
			if result.err != nil || result.status != 503 {
				t.Fatalf("forced reload did not end pending request: %+v", result)
			}
			awaitSignal(t, old.stopped, "interrupted instance")
			status, _, body := testResponse(t, srv, "GET", "/", nil)
			if status != 200 || string(body) != "new" {
				t.Fatalf("new generation returned %d %q", status, body)
			}
		})
	}
}

func TestStopInterruptsInitialization(t *testing.T) {
	rt, srv := testRuntime(t, `registerEventHandler(() => {})`, 1)
	current := rt.shards[0].Load()
	loaded := make(chan error, 1)
	go func() {
		loaded <- rt.LoadScript("busy-init.js", `registerEventHandler(() => {}); while (true) {}`, false)
	}()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	var preparing *runtimeInstance
	for preparing == nil {
		rt.shardMu.RLock()
		for inst := range rt.instances {
			if inst != current && inst.middlewareType.Load().(handlerType) == handlerTypeEvent {
				preparing = inst
			}
		}
		rt.shardMu.RUnlock()
		if preparing != nil {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("initialization did not begin")
		}
	}
	rt.Stop(true)
	select {
	case err := <-loaded:
		if err == nil {
			t.Fatal("interrupted initialization succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reload did not return after interruption")
	}
	awaitSignal(t, preparing.stopped, "initializing VM termination")
	status, _, _ := testResponse(t, srv, "GET", "/", nil)
	if status != 503 {
		t.Fatalf("stopped runtime returned %d", status)
	}
}

func TestConcurrentFetchAndReload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "present")
		fmt.Fprint(w, "upstream")
	}))
	t.Cleanup(upstream.Close)
	script := fmt.Sprintf(`registerEventHandler(async e => {
		const response = await e.fetch(%q);
		e.respondWith(new Response(await response.text(), {headers: response.headers}));
	}, {fetch:true})`, upstream.URL)
	rt, srv := testRuntime(t, script, 4)
	done := make(chan error, 4)
	for range 4 {
		go func() {
			for range 20 {
				resp, err := srv.Client().Get(srv.URL)
				if err != nil {
					done <- err
					return
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					done <- err
					return
				}
				if resp.StatusCode == 503 {
					continue
				}
				if resp.StatusCode != 200 || string(body) != "upstream" || resp.Header.Get("X-Test") != "present" {
					done <- fmt.Errorf("response changed during reload: %d %q", resp.StatusCode, body)
					return
				}
			}
			done <- nil
		}()
	}
	for range 3 {
		if err := rt.LoadScript("fetch.js", script, true); err != nil {
			t.Fatal(err)
		}
	}
	for range 4 {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}

func TestForceStopDuringFetchBody(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(canceled)
	}))
	t.Cleanup(upstream.Close)
	rt, srv := testRuntime(t, fmt.Sprintf(`registerEventHandler(async e => {
		const response = await e.fetch(%q);
		e.respondWith(new Response(await response.text()));
	}, {fetch:true})`, upstream.URL), 1)
	pending := requestAsync(srv)
	awaitSignal(t, started, "upstream headers")
	rt.Stop(true)
	result := <-pending
	if result.err != nil || result.status != 503 {
		t.Fatalf("stop did not cancel pending response body: %+v", result)
	}
	awaitSignal(t, canceled, "upstream body cancellation")
}

func TestForceStopDrainingGeneration(t *testing.T) {
	rt, srv := testRuntime(t, `registerEventHandler(async () => { testStarted(); await new Promise(() => {}) })`, 1)
	old := rt.shards[0].Load()
	started := make(chan struct{})
	onInstance(t, old, func(vm *goja.Runtime) { vm.Set("testStarted", func() { close(started) }) })
	pending := requestAsync(srv)
	awaitSignal(t, started, "pending handler")
	if err := rt.LoadScript("new.js", `registerEventHandler(() => {})`, false); err != nil {
		t.Fatal(err)
	}
	rt.Stop(true)
	result := <-pending
	if result.err != nil || result.status != 503 {
		t.Fatalf("stop did not cancel retired generation: %+v", result)
	}
	awaitSignal(t, old.stopped, "retired generation termination")
	status, _, _ := testResponse(t, srv, "GET", "/", nil)
	if status != 503 {
		t.Fatalf("stopped runtime returned %d", status)
	}
}

func TestInterruptBusyScript(t *testing.T) {
	rt, srv := testRuntime(t, `registerEventHandler(() => { testStarted(); while (true) {} })`, 1)
	started := make(chan struct{})
	onInstance(t, rt.shards[0].Load(), func(vm *goja.Runtime) { vm.Set("testStarted", func() { close(started) }) })
	pending := requestAsync(srv)
	awaitSignal(t, started, "busy script")
	rt.Stop(true)
	result := <-pending
	if result.err != nil || result.status != 503 {
		t.Fatalf("busy script was not interrupted: %+v", result)
	}
}

func TestWaitUntilSurvivesGracefulReload(t *testing.T) {
	first, second := make(chan struct{}), make(chan struct{})
	allowFirst, allowSecond := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started, release := first, allowFirst
		if r.URL.Path == "/second" {
			started, release = second, allowSecond
		}
		close(started)
		select {
		case <-release:
			fmt.Fprint(w, "background")
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(upstream.Close)
	script := fmt.Sprintf(`registerEventHandler(e => {
		e.waitUntil((async () => {
			await (await e.fetch(%q + "/first")).text();
			e.waitUntil((async () => {
				await (await e.fetch(%q + "/second")).text();
				testFinished();
			})());
		})());
		e.respondWith(new Response("foreground"));
	}, {fetch: true})`, upstream.URL, upstream.URL)
	rt, srv := testRuntime(t, script, 1)
	old := rt.shards[0].Load()
	finished := make(chan struct{})
	onInstance(t, old, func(vm *goja.Runtime) { vm.Set("testFinished", func() { close(finished) }) })
	status, _, body := testResponse(t, srv, "GET", "/", nil)
	if status != 200 || string(body) != "foreground" {
		t.Fatalf("background work blocked response: %d %q", status, body)
	}
	awaitSignal(t, first, "first background fetch")
	if err := rt.LoadScript("new.js", `registerEventHandler(() => {})`, false); err != nil {
		t.Fatal(err)
	}
	close(allowFirst)
	awaitSignal(t, second, "nested waitUntil fetch")
	close(allowSecond)
	awaitSignal(t, finished, "background work completion")
	awaitSignal(t, old.stopped, "background work cleanup")
}
