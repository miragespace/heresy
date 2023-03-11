# 😅 Heresy

[![GoDoc](https://godoc.org/go.miragespace.co/heresy?status.svg)](https://pkg.go.dev/go.miragespace.co/heresy)

## What is it?

```
Heresy
noun, /ˈher.ə.si/

(the act of having) an opinion or belief that is the opposite of
    or against what is the official or popular opinion,
    or an action that shows that you have no respect for the official opinion.
```

Heresy is a runtime to allow you to run JavaScript as middleware for `http.Server` in either Express.js style or Web Worker `FetchEvent` style, with support for hot-reloading the JavaScript.

## Features Matrix

| **Supported Features via Polyfill**                                        |
|----------------------------------------------------------------------------|
| URLSearchParams                                                            |
| `TextEncoder`/`TextDecoder` (UTF-8 Only)                                   |
| Web Streams API (`ReadableStream`, etc), backed by `io.Reader`/`io.Writer` |
| Fetch API (`Headers`, `Request`, `Response`)                               |

| **Style**  | Status | req/request                                                                                 | resp/respondWith                                                                                        | next       |
|------------|--------|---------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------|------------|
| Express.js | WIP    | Partial implementations <br> (`"ip", "method", "path", "protocol", "secure", "get", "res"`) | Partial implementations <br> (`"status", "send", "json", "get", "end", "set", "header", "headersSent"`) | Works      |
| FetchEvent | WIP    | With native `ReadableStream` support backed by `io.Reader`                                  | WIP                                                                                                     | Bypass WIP |


## Examples

### Express.js style

```javascript
function httpHandler(ctx) {
	const { req, res, next } = ctx
    if (req.path === "/") {
        next()
    } else {
        res.status(403).send({error: 'access denied'})
    }
}

registerMiddlewareHandler(httpHandler)
```

### `FetchEvent` style

```javascript
async function eventHandler(evt) {
    const { request, respondWith } = evt
    if (request.method === "POST") {
        const json = await request.json()
        respondWith(new Response(JSON.stringify(json)))
    }
    // to the next handler in http.Server
}

registerEventHandler(eventHandler)
```

### With network access

```javascript
async function httpHandler(ctx) {
    const { fetch, res } = ctx
	const resp = await fetch("https://example.com/")
    res.send(await resp.text())
}

registerMiddlewareHandler(httpHandler, {
    fetch: true
})
```

## TODO: Complete this README