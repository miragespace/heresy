"use strict";
const __runtimeFetch = (goWrapper) => {
    return async (input, options) => {
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
        return new Response(body, {
            status: statusCode,
            statusText: statusText,
            headers: header,
        });
    };
};
