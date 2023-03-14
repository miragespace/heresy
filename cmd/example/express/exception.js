async function httpHandler(ctx) {
    console.log(Object.getOwnPropertyNames(ctx))
    throw new Error("exception!")
}

registerExpressHandler(httpHandler, {
    fetch: true
})