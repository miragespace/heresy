async function eventHandler(evt) {
    const { waitUntil, respondWith } = evt

    const p = async () => {
        try {
            const resp = await fetch("http://127.0.0.1:8000")
            const text = await resp.text()
            console.log("got", text)
            // nested waitUntil
            waitUntil((async () => {
                await (new Promise((r) => {
                    setTimeout(r, 500)
                }))
                const resp  = await fetch("http://127.0.0.1:8000")
                console.log("got", resp.statusText)
            })())
        }catch (e) {
            console.log("exploded", e)
        }
    }
    waitUntil(p())

    const resp = await fetch("http://127.0.0.1:8000")
    respondWith(new Response(resp.body))
}

registerEventHandler(eventHandler, {
    fetch: true
})
