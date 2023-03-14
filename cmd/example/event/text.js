async function eventHandler(evt) {
    const { request, respondWith } = evt
    const text = await request.text()
    respondWith(new Response(text))
}

registerEventHandler(eventHandler)