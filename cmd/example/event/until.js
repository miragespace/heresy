function eventHandler(evt) {
    const { respondWith, waitUntil, fetch } = evt
    respondWith(new Response("Hello World!"))
    waitUntil((async () => {
        const resp = await fetch("http://127.0.0.1:8000")
        console.log("got", await resp.text())
    })())
}

registerEventHandler(eventHandler, {
    fetch: true
})