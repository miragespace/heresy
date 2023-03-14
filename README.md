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

Heresy is a pure Go runtime that lets you:
1. Embed the runtime and run JavaScript as middleware for `http.Server` in either Express.js style, or Web Worker `FetchEvent` style;
    - The handler script can be reloaded on-the-fly!
2. Run the runtime as a reverse proxy to some backend services, with the power of JavaScript as scripting language to intercept requests;
3. Or spin up the runtime as a standalone server to run JavaScript application, with the power of Go.

## What is it *not*?

1. It is *not* a secure/isolated runtime to run untrusted user code;
2. It is *not* a sandbox similar to `v8::Isolate`.

## Examples

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

The JavaScript runtime is provided by [goja](https://github.com/dop251/goja). Currently it supports most features up to ES2018, with the notable exceptions of:
1. async iterator (`async function* foo()` and `for await...of`);
2. `SharedArrayBuffer`;
3. ES2015 modules (`import foo from 'bar'`, please use a bundler that outputs UMD or CJS).

The recommended transpile target is ES2017. However, if you run into problems, ES6 can be used as a fallback.

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
| FetchEvent    | Implemented* | Works                                                           | Works                                                            | Works |
| Fetch API     | Implemented  |                                                                 |                                                                  |       |

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

## TODO: Complete this README