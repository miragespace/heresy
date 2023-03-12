"use strict";
const __runtimeFetch = (goWrapper) => {
    const fn = async (input, options) => {
        const request = new Request(input, options);
        const rawHeadersMap = {};
        request.headers.forEach((v, k) => {
            rawHeadersMap[k] = v;
        });
        const requestBody = request;
        let useBody;
        if (requestBody._bodyReadableStream) {
            useBody = requestBody._bodyReadableStream;
        }
        else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
            useBody = await requestBody.text();
        }
        const { statusText, statusCode, header, body } = await goWrapper.doFetch(request.url, request.method, rawHeadersMap, useBody);
        goWrapper.unsetCtx();
        return new Response(body, {
            status: statusCode,
            statusText: statusText,
            headers: header,
        });
    };
    // save the reference of NativeFetchWrapper,
    // needed for Fetch reuse
    fn.wrapper = goWrapper;
    return fn;
};
// this is a helper for FetchEvent.respondWith
const __runtimeResponseHelper = async (response) => {
    const { status, headers } = response;
    const rawHeadersMap = {};
    headers.forEach((v, k) => {
        rawHeadersMap[k] = v;
    });
    const requestBody = response;
    let useBody;
    if (requestBody._bodyReadableStream) {
        useBody = requestBody._bodyReadableStream;
    }
    else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
        useBody = await requestBody.text();
    }
    return [status, rawHeadersMap, useBody];
};
