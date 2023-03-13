"use strict";
const __runtimeFetch = (goWrapper) => {
    return async (input, options) => {
        const request = new Request(input, options);
        const requestBody = request;
        let useBody;
        if (requestBody._bodyReadableStream) {
            useBody = requestBody._bodyReadableStream;
        }
        else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
            useBody = await requestBody.text();
        }
        const { statusText, statusCode, header, body } = await goWrapper.doFetch(request.url, request.method, request.headers.map, // .map property is the backing storage of headers
        useBody);
        return new Response(body, {
            status: statusCode,
            statusText: statusText,
            headers: header,
        });
    };
};
// this is a helper for FetchEvent.respondWith
const __runtimeResponseHelper = async (response) => {
    if (!(response instanceof Response)) {
        return { ok: false };
    }
    const { status, headers } = response;
    const requestBody = response;
    let useBody;
    if (requestBody._bodyReadableStream) {
        useBody = requestBody._bodyReadableStream;
    }
    else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
        useBody = await requestBody.text();
    }
    // .map property is the backing storage of headers
    return { ok: true, status, headers: headers.map, body: useBody };
};
