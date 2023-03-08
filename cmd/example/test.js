"use strict";

async function httpHandler(ctx) {
	const { req, res } = ctx
	res.set('Content-Type', 'text/plain')
		.status(201)
		.end(`Hi ${req.ip}, here's a random number: ${Math.random()}`)
}

registerMiddlewareHandler(httpHandler)