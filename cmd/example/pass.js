async function eventHandler(evt) {
    const { fetch, request, respondWith } = evt
    const resp = await fetch(new Request("http://127.0.0.1:8000", {
        method: "POST",
        body: request.body,
    }))
    respondWith(new Response(resp.body))
}

registerEventHandler(eventHandler, {
    fetch: true
})