"use strict";

async function httpHandler(url) {
    console.log("got url", url)
	return fetch("https://example.com/")
}

registerRequestHandler(httpHandler)