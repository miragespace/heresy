"use strict";

async function httpHandler(url) {
	return `Here's a random number: ${Math.random()}`
}

registerRequestHandler(httpHandler)