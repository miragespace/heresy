async function eventHandler(evt) {
    console.log(Object.getOwnPropertyNames(evt))
    const { request } = evt
    console.log(request)
    console.log(request.url)
    console.log(request.headers)
    for (const [k, v] of request.headers) {
        console.log(k, v)
    }
    if (request.method == "POST") {
        const json = await request.json()
        console.log(JSON.stringify(json))
    }
    // TODO: TypeError: Could not convert &{{0xc000908990 0xc0009081b0} 0xc0005aa080} to primitive
    // despite setting the prototype to be an instance of FetchEvent
    // console.log(evt)
}

registerEventHandler(eventHandler)