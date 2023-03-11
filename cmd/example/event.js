async function eventHandler(evt) {
    console.log(evt)
    console.log(Object.getOwnPropertyNames(evt))
    const { request } = evt
    console.log(request)
    console.log(request.url)
    const url = new URL(request.url)
    const params = new URLSearchParams(url.search)
    for (const [k, v] of params) {
        console.log(k, v)
    }
    console.log(request.headers)
    for (const [k, v] of request.headers) {
        console.log(k, v)
    }
    if (request.method == "POST") {
        console.log(request.body)
        console.log("body used", request.bodyUsed)
        const json = await request.json()
        console.log(JSON.stringify(json))
        console.log("body used", request.bodyUsed)
    }
}

registerEventHandler(eventHandler)