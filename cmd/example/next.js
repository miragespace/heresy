"use strict";

function httpHandler(ctx) {
	const { next } = ctx
    next()
}

registerMiddlewareHandler(httpHandler)