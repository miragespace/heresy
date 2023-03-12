async function eventHandler(evt) {
    // if (evt.request.url.includes("resp")) {
    //     const resp = await evt.fetch("https://example.com")
    //     evt.respondWith(resp)
    // }

    // throw new Error("oops")
    if (evt.request.url.includes("resp")) {
        evt.respondWith(
            new Response(JSON.stringify({sup: "bro"}), {
                status: 201,
                headers: {
                    'content-type': 'application/json'
                }
            })
        )
    }
    // evt.respondWith(new Response(new ReadableStream()))
}

registerEventHandler(eventHandler, {
    fetch: true
})