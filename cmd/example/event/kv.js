async function eventHandler(evt) {
    const { kv, request, respondWith } = evt
    const counter = Number(await kv.potato.get("sup"))
    if (request.url.endsWith("/set")) {
        await kv.potato.put("sup", String(counter + 1))
        respondWith(new Response("ok"))
    } else {
        respondWith(new Response(String(counter)))
    }
}

registerEventHandler(eventHandler)