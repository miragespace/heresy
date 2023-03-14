"use strict";

function httpHandler(ctx) {
	const { req, res, next } = ctx
    if (req.path === "/") {
        next()
    } else {
        res.status(403).send({error: 'access denied'})
    }
}

registerExpressHandler(httpHandler)