async function eventHandler(evt) {
    evt.respondWith(new Response(evt.request.body, {
        headers: evt.request.headers
    }))
}

registerEventHandler(eventHandler, {
    fetch: true
})