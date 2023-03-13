"use strict";

async function httpHandler(ctx) {
    const { res } = ctx
	const resp = await fetch("http://127.0.0.1:8000", {
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