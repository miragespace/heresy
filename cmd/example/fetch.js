"use strict";

async function httpHandler(ctx) {
    const { fetch, res } = ctx
	const resp = await fetch("https://example.com/", {
        headers: {
            'User-Agent': 'heresy/fetcher'
        }
    })
    const html = await resp.text()
    res.send(html)
}

registerMiddlewareHandler(httpHandler, {
    fetch: true
})