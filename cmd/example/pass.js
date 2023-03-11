async function eventHandler(evt) {
    const { fetch, request } = evt
    const resp = await fetch(new Request("https://eni4mgzd1a5em.x.pipedream.net", {
        method: "POST",
        body: request.body,
    }))
    console.log(await resp.text())
}

registerEventHandler(eventHandler, {
    fetch: true
})