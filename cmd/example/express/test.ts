"use strict";

function randomNumber(ctx: MiddlewareContext) {
	const { req, res } = ctx
	res.set('Content-Type', 'text/plain')
		.status(201)
		.end(`Hi ${req.ip}, here's a random number: ${Math.random()}`)
}

registerExpressHandler(randomNumber)

// heresy runtime types. similar to Express.js
type MiddlewareRequest = {
	readonly ip: string
	readonly method: string
	readonly path: string
	readonly protocol: 'http' | 'https'
	readonly secure: boolean
	readonly res: MiddlewareResponse

	get(headerKey: string): string | undefined
}

type MiddlewareResponse = {
	readonly headersSent: boolean

	send(body?: string | object | boolean | Array<unknown>): void
	json(body?: string | object | number | boolean | null | Array<unknown>): void
	end(body?: string): void

	get(headerKey: string): string | string[] | number | undefined
	set(headerKey: string, headerValue: string | string[] | number): MiddlewareResponse
	header(headerKey: string): MiddlewareResponse
	status(code: number): MiddlewareResponse
}

type MiddlewareContext = {
	req: MiddlewareRequest
	res: MiddlewareResponse
	next(): void
	fetch?(url: string): Promise<string>
}

type MiddlewareHandler = (ctx: MiddlewareContext) => void | Promise<void>

type MiddlewareOptions = {
	fetch: boolean
}

declare function registerExpressHandler(handler: MiddlewareHandler, options?: MiddlewareOptions): void