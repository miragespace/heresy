declare function runtimeFetch(
  input: Request | string,
  options?: Request | RequestInit
): Promise<Response>;

interface RuntimeFetchResult {
  statusText: string;
  statusCode: number;
  headers: Headers;
  body: ReadableStream;
}

interface RuntimeFetchHandler {
  doFetch(
    url: string,
    method: string,
    headers: Record<string, string>,
    body?: ReadableStream | string | ArrayBuffer
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

    const requestBody = request as Body;
    let useBody: ReadableStream | string | ArrayBuffer | undefined;
    if (requestBody._bodyReadableStream) {
      useBody = requestBody._bodyReadableStream;
    } else if (requestBody._bodyArrayBuffer) {
      useBody = await request.arrayBuffer();
    } else if (requestBody._bodyText !== undefined) {
      useBody = await requestBody.text();
    }

    const { statusText, statusCode, headers, body } = await goWrapper.doFetch(
      request.url,
      request.method,
      (request.headers as any).map, // .map property is the backing storage of headers
      useBody
    );

    return new Response(body, {
      status: statusCode,
      statusText: statusText,
      headers,
    });
  };
};

// this is a helper for FetchEvent.respondWith
const __runtimeResponseHelper = async (input: Response | Promise<Response>) => {
  const response = await input;
  if (!(response instanceof Response)) {
    return { ok: false };
  }

  const { status, headers } = response;

  const requestBody = response as Body;
  let useBody: ReadableStream | string | ArrayBuffer | undefined;
  if (requestBody._bodyReadableStream) {
    useBody = requestBody._bodyReadableStream;
  } else if (requestBody._bodyArrayBuffer) {
    useBody = await response.arrayBuffer();
  } else if (requestBody._bodyText !== undefined) {
    useBody = await requestBody.text();
  }

  // .map property is the backing storage of headers
  return { ok: true, status, headers: (headers as any).map, body: useBody };
};
