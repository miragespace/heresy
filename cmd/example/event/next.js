async function eventHandler(evt) {
    const { request, respondWith } = evt
    if (!request.url.endsWith("/")) {
         respondWith(new Response( JSON.stringify({error: "access denied"}),
            {
                header: {
                    'content-type': 'application/json'
                }
            }
        ))
    }
}

registerEventHandler(eventHandler)