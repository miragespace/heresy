"use strict";

async function httpHandler(ctx) {
    const { fetch, res } = ctx
	const html = await fetch("https://example.com/")
    res.send(html)
}

registerMiddlewareHandler(httpHandler, {
    fetch: true
})