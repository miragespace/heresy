declare function runtimeFetch(
  input: Request | string,
  options?: Request | RequestInit
): Promise<Response>;

interface RuntimeFetchResult {
  statusText: string;
  statusCode: number;
  header: Headers;
  body: ReadableStream;
}

interface RuntimeFetchHandler {
  doFetch(
    url: string,
    method: string,
    headers: Record<string, string>,
    body?: ReadableStream | string
  ): Promise<RuntimeFetchResult>;
}

interface Body {
  readonly _bodyReadableStream?: ReadableStream;
  readonly _bodyArrayBuffer?: ArrayBuffer;
  readonly _bodyText?: string;
  text(): Promise<string>;
}

const __runtimeFetch = (
  goWrapper: RuntimeFetchHandler
): typeof runtimeFetch => {
  return async (
    input: Request | string,
    options?: Request | RequestInit
  ): Promise<Response> => {
    const request = new Request(input, options);

    const rawHeadersMap: Record<string, string> = {};
    request.headers.forEach((v: string, k: string) => {
      rawHeadersMap[k] = v;
    });

    const requestBody = request as Body;
    let useBody: ReadableStream | string | undefined;
    if (requestBody._bodyReadableStream) {
      useBody = requestBody._bodyReadableStream;
    } else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
      useBody = await requestBody.text();
    }

    const { statusText, statusCode, header, body } = await goWrapper.doFetch(
      request.url,
      request.method,
      rawHeadersMap,
      useBody
    );

    return new Response(body, {
      status: statusCode,
      statusText: statusText,
      headers: header,
    });
  };
};

// this is a helper for FetchEvent.respondWith
const __runtimeResponseHelper = async (response: Response) => {
  if (!(response instanceof Response)) {
    return [false];
  }

  const { status, headers } = response;

  const rawHeadersMap: Record<string, string> = {};
  headers.forEach((v: string, k: string) => {
    rawHeadersMap[k] = v;
  });

  const requestBody = response as Body;
  let useBody: ReadableStream | string | undefined;
  if (requestBody._bodyReadableStream) {
    useBody = requestBody._bodyReadableStream;
  } else if (requestBody._bodyArrayBuffer || requestBody._bodyText) {
    useBody = await requestBody.text();
  }

  return [true, status, rawHeadersMap, useBody];
};
