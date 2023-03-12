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
  unsetCtx(): void;
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
  const fn = async (
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

    goWrapper.unsetCtx();

    return new Response(body, {
      status: statusCode,
      statusText: statusText,
      headers: header,
    });
  };
  // save the reference of NativeFetchWrapper,
  // needed for Fetch reuse
  (fn as any).wrapper = goWrapper;
  return fn;
};

// this is a helper for FetchEvent.respondWith
const __runtimeResponseHelper = async (response: Response) => {
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

  return [status, rawHeadersMap, useBody];
};
