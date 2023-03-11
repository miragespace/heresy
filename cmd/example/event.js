async function eventHandler(evt) {
    console.log(Object.getOwnPropertyNames(evt))
    const { request } = evt
    console.log(request.url)
    for (const [k, v] of request.headers) {
        console.log(k, v)
    }
    if (request.method == "POST") {
        const json = await request.json()
        console.log(JSON.stringify(json))
    }
}

registerEventHandler(eventHandler)