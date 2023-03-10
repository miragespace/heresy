import Body, { BodyInit } from "./Body";
import Headers, { HeadersInit } from "./Headers";

export interface ResponseInit {
  status?: number;
  statusText?: string;
  headers?: HeadersInit;
}

class Response {
  readonly _body: Body;
  readonly headers: Headers;
  readonly ok: boolean;
  readonly status: number;
  readonly statusText: string;
  readonly url: string;

  constructor(body: BodyInit | null, options: ResponseInit | Response) {
    this.status = options.status ?? 200;
    this.ok = this.status >= 200 && this.status < 300;
    this.statusText = options.statusText ?? "";
    this.headers = new Headers(options.headers);
    if (body) {
      this._body = new Body(body);
    }

    if (options instanceof Response) {
      this.url = options.url;
    } else {
      this.url = "";
    }

    if (!this.headers.has("content-type") && this._body._mimeType) {
      this.headers.set("content-type", this._body._mimeType);
    }
  }

  get bodyUsed() {
    return this._body.bodyUsed;
  }

  clone() {
    return new Response(this._body.bodyInit, {
      status: this.status,
      statusText: this.statusText,
      headers: new Headers(this.headers),
      url: this.url,
    });
  }

  arrayBuffer() {
    return this._body.arrayBuffer();
  }

  text() {
    return this._body.text();
  }

  json() {
    return this._body.json();
  }

  get body() {
    return this._body.body;
  }

  static redirect(url: string, status: number) {
    const redirectStatuses = [301, 302, 303, 307, 308];

    if (!redirectStatuses.includes(status)) {
      throw new RangeError(`Invalid status code: ${status}`);
    }

    return new Response(null, { status: status, headers: { location: url } });
  }
}

export default Response;
