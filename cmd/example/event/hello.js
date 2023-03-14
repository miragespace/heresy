function eventHandler(event) {
    event.respondWith(new Response("Hello world!"))
}

registerEventHandler(eventHandler)