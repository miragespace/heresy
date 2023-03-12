

async function httpHandler(ctx) {
    throw new Error("exception!")
}

registerMiddlewareHandler(httpHandler, {
    fetch: true
})