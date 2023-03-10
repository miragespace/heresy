import Body, { BodyInit } from "./Body";
import Headers, { HeadersInit } from "./Headers";

export type Method = "HEAD" | "GET" | "POST" | "PUT" | "DELETE" | "OPTIONS";

export interface RequestInit {
  method?: Method;
  headers?: HeadersInit;
  body?: BodyInit | null;
}

class Request {
  readonly _body: Body;
  readonly url: string;
  readonly headers: Headers;
  readonly method: Method;

  constructor(input: Request | string, options: Request | RequestInit) {
    if (input instanceof Request) {
      if (input.body && input.bodyUsed) {
        throw new TypeError("Already read");
      }

      this.url = input.url;
      this.method = input.method;
      this.headers = new Headers(options.headers ?? input.headers);

      if (!options.body && input._body.bodyInit) {
        this._body = new Body(input._body.bodyInit);
        input._body.bodyUsed = true;
      }
    } else {
      this.url = input;
    }

    if (!this._body && options.body) {
      this._body = this._body ?? new Body(options.body);
    }

    this.method = options.method ?? "GET";

    if (this._body.bodyInit && ["GET", "HEAD"].includes(this.method)) {
      throw new TypeError("Body not allowed for GET or HEAD requests");
    }

    this.headers = this.headers ?? new Headers(options.headers);

    if (!this.headers.has("content-type") && this._body._mimeType) {
      this.headers.set("content-type", this._body._mimeType);
    }
  }

  get body() {
    return this._body.body;
  }

  get bodyUsed() {
    return this._body.bodyUsed;
  }

  clone() {
    return new Request(this, { body: this._body.bodyInit });
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
}

export default Request;
