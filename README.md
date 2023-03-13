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
        respondWith(new Response(JSON.stringify(json), {
            headers: {
                'content-type': 'application/json'
            }
        }))
    }
    // to the next handler in http.Server
}

registerEventHandler(eventHandler)
```

### With network access

```javascript
// fetch() will be undefined outside of the handler
async function httpHandler(ctx) {
    const { res } = ctx
	const resp = await fetch("https://example.com/") // fetch() will be available in the handler
    res.send(await resp.text())
}

registerMiddlewareHandler(httpHandler, {
    fetch: true
})
```

## Features Matrix

| **Supported Features via Polyfill**                                        |
|----------------------------------------------------------------------------|
| URLSearchParams                                                            |
| `TextEncoder`/`TextDecoder` (UTF-8 Only)                                   |
| Web Streams API (`ReadableStream`, etc), backed by `io.Reader`/`io.Writer` |
| Fetch API (`Headers`, `Request`, `Response`)                               |

| **Component** | Status                     | req/request                                                     | resp/respondWith                                                 | next  |
|---------------|----------------------------|-----------------------------------------------------------------|------------------------------------------------------------------|-------|
| Express.js    | WIP                        | Partial implementations <br> (see `request_context_request.go`) | Partial implementations <br> (see `request_context_response.go`) | Works |
| FetchEvent    | Implemented*               | Works                                                           | Works                                                            | Works |
| Fetch API     | Available in handler scope |                                                                 |                                                                  |       |

*: Even though ECMAScript is single-threaded in nature, heresy runtime manages data access and IOs asynchronously. Therefore, once your event handler returns, it should not call any methods from `FetchEvent`.

The following usage will result in a race _and_ crash the runtime:
```javascript
function eventHandler(evt) {
    // ...
    evt.respondWith(/* ... */)
    setTimeout(() => {
        fetch(/* ... */)
    }, 100)
    // fetch will be called after your handler returns!
}
```

Use `.waitUntil` instead:
```javascript
function eventHandler(evt) {
    // ...
    evt.respondWith(/* ... */)
    evt.waitUntil((async () => {
        // e.g. send request metrics
        await fetch(/* ... */)
        await fetch(/* ... */)
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
            await fetch(/* ... */)
        })())
    }, 100)
    // evt.waitUntil will be called after your handler returns!
}
```

## TODO: Complete this README