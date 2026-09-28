# 😅 Heresy

[![GoDoc](https://godoc.org/go.miragespace.co/heresy?status.svg)](https://pkg.go.dev/go.miragespace.co/heresy)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmiragespace%2Fheresy.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmiragespace%2Fheresy?ref=badge_shield)

## What is it?

```
Heresy
noun, /ˈher.ə.si/

(the act of having) an opinion or belief that is the opposite of
    or against what is the official or popular opinion,
    or an action that shows that you have no respect for the official opinion.
```

Heresy is a pure Go runtime that lets you:
1. Embed the runtime and run JavaScript as middleware for `http.Server` in either Express.js style, or Web Worker `FetchEvent` style;
    - The handler script can be reloaded on-the-fly!
2. Run the runtime as a reverse proxy to some backend services, with the power of JavaScript as scripting language to intercept requests;
3. Or spin up the runtime as a standalone server to run JavaScript application, with the power of Go.

## What is it *not*?

1. It is *not* a secure/isolated runtime to run untrusted user code;
2. It is *not* a sandbox similar to `v8::Isolate`.

## Examples

### Running locally

Use Go 1.26 or newer and Node.js 24 with npm. The Go runtime embeds its JavaScript
assets, so Node.js is only needed when rebuilding those assets.

```sh
make deps                   # initialize the pinned JS submodule; install lockfiles
make typecheck extensions js
make test                   # Go tests with the race detector
make example                # listen on 127.0.0.1:8081
```

In another terminal, load a handler and send a request:

```sh
make reload                 # load event/hello.js
curl http://127.0.0.1:8081/
make reload file=express/next.js
```

The example's `/reload` and `/debug` endpoints are development tools. Keep them
private: `/reload` accepts trusted JavaScript via PUT, with a 1 MiB upload limit.
The example returns 503 until a script has been loaded.

### Embedding

```go
rt, err := heresy.NewRuntime(logger, nil, 4) // nil creates an empty KV manager
if err != nil {
    return err
}
defer rt.Stop(true)

if err := rt.LoadScript("handler.js", script, false); err != nil {
    return err
}
handler := rt.Middleware(nextHTTPHandler)
```

Each shard has independent JavaScript globals. Configure shared storage through
`kv.NewKVManager()` before creating the runtime; the included `memory` backend
shares values across shards in this process.

`LoadScript` prepares every shard before publishing a new generation. Compilation
or initialization errors leave the current generation serving requests.
With `interrupt=false`, old requests and their `waitUntil` work drain in the
background. With `interrupt=true`, old JavaScript and native I/O are canceled and
its timers are removed; pending requests that have not sent a response receive
503. `Stop(false)` starts the same graceful drain, and `Stop(true)` interrupts all
generations and waits for native cleanup. Blocking Go handlers and KV backends
must honor cancellation for an interrupt to finish promptly.

### Express.js style

```javascript
function httpHandler({ req, res, next }) {
    if (req.path === "/") {
        next()
    } else {
        res.status(403).send({error: 'access denied'})
    }
}

registerExpressHandler(httpHandler)
```

### `FetchEvent` style

```javascript
async function eventHandler(event) {
    if (event.request.method === "POST") {
        event.respondWith(new Response(event.request.body, {
            headers: event.request.headers
        }))
    }
    // to the next handler in http.Server
}

registerEventHandler(eventHandler)
```

### With network access

```javascript
async function httpHandler({ res, fetch }) {
    const resp = await fetch("https://example.com/")
    res.send(await resp.text())
}

registerExpressHandler(httpHandler, {
    fetch: true
})

// ... similarly in FetchEvent
// async function eventHandler(event) {
//     const { fetch } = event
//     const resp = await fetch("https://example.com/")
//     event.respondWith(resp)
// }

// registerEventHandler(eventHandler, {
//     fetch: true
// })
```

## Supported ECMAScript Features

The JavaScript runtime is provided by [goja](https://github.com/dop251/goja).
Syntax support follows the version pinned in `go.mod`. The bundled polyfills
target ES2017; bundle application scripts to UMD or CommonJS before loading them.
The Web API implementations below provide a subset of browser/Workers behavior,
and Express-style handlers implement a subset of Express.

## Runtime Features Matrix

| **Supported Features via Polyfill**                                        |
|----------------------------------------------------------------------------|
| URLSearchParams                                                            |
| `TextEncoder`/`TextDecoder` (UTF-8 Only)                                   |
| Web Streams API (`ReadableStream`, etc), backed by `io.Reader`/`io.Writer` |
| Fetch API (`Headers`, `Request`, `Response`)                               |

| **Component** | Status       | req/request                                                     | resp/respondWith                                                 | next  |
|---------------|--------------|-----------------------------------------------------------------|------------------------------------------------------------------|-------|
| Express.js    | WIP          | Partial implementations <br> (see `request_context_request.go`) | Partial implementations <br> (see `request_context_response.go`) | Works |
| FetchEvent    | Implemented* | Native request bodies and headers                               | Response or Promise<Response>; binary and native stream bodies    | Works |
| Fetch API     | Partial      | Strings, binary buffers, and native stream bodies                | Response status, headers, and native stream bodies                |       |

Custom JavaScript-created streams cannot currently be sent as fetch or response
bodies. Native streams obtained from a request or an outbound fetch can be passed
through without converting them to text.

*: Even though ECMAScript is single-threaded in nature, heresy runtime manages data access and IOs asynchronously. Therefore, once your event handler returns, it should not call any methods from `FetchEvent`.

The following usage will result in a race _and_ crash the runtime:
```javascript
function eventHandler(evt) {
    // ...
    evt.respondWith(/* ... */)
    setTimeout(() => {
        evt.fetch(/* ... */)
    }, 100)
    // evt.fetch will be called after your handler returns!
}
```

Use `.waitUntil` instead:
```javascript
function eventHandler(evt) {
    // ...
    evt.respondWith(/* ... */)
    evt.waitUntil((async () => {
        // e.g. send request metrics
        await evt.fetch(/* ... */)
        await evt.fetch(/* ... */)
    })())
}
```

The first rule still applies if you use `.waitUntil` incorrectly. The following usage will also crash the runtime:
```javascript
function eventHandler(evt) {
    // ...
    evt.respondWith(/* ... */)
    setTimeout(() => {
        evt.waitUntil((async () => {
            await evt.fetch(/* ... */)
        })())
    }, 100)
    // evt.waitUntil will be called after your handler returns!
}
```

## Development

Go behavior tests are split by runtime lifecycle, handlers, fetch, request/response
semantics, and KV. CI checks Go 1.26 and 1.27, typechecks and rebuilds JavaScript,
checks generated assets against Git, and runs dependency audits.

The `js` directory is a separate Git submodule. Changes to its sources or build
dependencies must be committed there, followed by the updated submodule pointer
and regenerated `polyfill/node_modules` assets in this repository. Push the
submodule commit before a parent commit that references it.

TypeScript is pinned to 6.x because the Rollup TypeScript plugin uses the classic
compiler API. Dependency installation uses `npm ci --ignore-scripts`; builds run
through the explicit `build` scripts.

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmiragespace%2Fheresy.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmiragespace%2Fheresy?ref=badge_large)
