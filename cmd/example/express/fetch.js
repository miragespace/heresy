"use strict";

async function httpHandler(ctx) {
    const { fetch, res } = ctx
	const resp = await fetch("http://127.0.0.1:8000")
    const html = await resp.text()
    res.send(html)
}

registerExpressHandler(httpHandler, {
    fetch: true
})