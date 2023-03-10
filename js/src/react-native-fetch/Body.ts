import { drainStream, readArrayBufferAsText } from "./utils";

export type BodyInit = URLSearchParams | ArrayBuffer | ReadableStream | string;

class Body {
  bodyUsed: boolean = false;
  bodyInit: BodyInit | null;
  _mimeType: string;
  _bodyText: string;
  _bodyArrayBuffer: ArrayBuffer;
  _bodyReadableStream: ReadableStream;

  constructor(body: BodyInit | null) {
    this.bodyInit = body;

    if (body instanceof URLSearchParams) {
      // URLSearchParams is not handled natively so we reassign bodyInit for fetch to send it as text
      this._bodyText = this.bodyInit = body.toString();
      this._mimeType = "application/x-www-form-urlencoded;charset=UTF-8";
      return this;
    }

    if (body instanceof ArrayBuffer) {
      this._bodyArrayBuffer = body.slice(0);
      this._mimeType = "application/octet-stream";
      return this;
    }

    if (ArrayBuffer.isView(body)) {
      this._bodyArrayBuffer = body.buffer;
      this._mimeType = "application/octet-stream";
      return this;
    }

    if (body instanceof ReadableStream) {
      this._bodyReadableStream = body;
      this._mimeType = "application/octet-stream";
      return this;
    }

    if (body) {
      this._bodyText = body.toString();
      this._mimeType = "text/plain;charset=UTF-8";
    }
  }

  __consumed() {
    if (this.bodyUsed) {
      return Promise.reject(new TypeError("Already read"));
    }
    this.bodyUsed = true;
  }

  async arrayBuffer() {
    const alreadyConsumed = this.__consumed();
    if (alreadyConsumed) {
      return alreadyConsumed;
    }

    if (this._bodyReadableStream) {
      const typedArray = await drainStream(this._bodyReadableStream);

      return typedArray.buffer;
    }

    if (this._bodyArrayBuffer) {
      if (ArrayBuffer.isView(this._bodyArrayBuffer)) {
        const { buffer, byteOffset, byteLength } = this._bodyArrayBuffer;

        return Promise.resolve(
          buffer.slice(byteOffset, byteOffset + byteLength)
        );
      }

      return Promise.resolve(this._bodyArrayBuffer);
    }

    const text = this._bodyText;
    const encoder = new TextEncoder();

    return encoder.encode(text);
  }

  async text() {
    const alreadyConsumed = this.__consumed();
    if (alreadyConsumed) {
      return alreadyConsumed;
    }

    if (this._bodyReadableStream) {
      const typedArray = await drainStream(this._bodyReadableStream);

      return readArrayBufferAsText(typedArray);
    }

    if (this._bodyArrayBuffer) {
      return readArrayBufferAsText(this._bodyArrayBuffer);
    }

    return this._bodyText;
  }

  async json<T>(): Promise<T> {
    const text = await this.text();

    return JSON.parse(text);
  }

  get body(): ReadableStream | null {
    if (this._bodyReadableStream) {
      return this._bodyReadableStream;
    }

    if (this._bodyArrayBuffer) {
      const typedArray = new Uint8Array(this._bodyArrayBuffer);

      return new ReadableStream({
        start(controller) {
          typedArray.forEach((chunk) => {
            controller.enqueue(chunk);
          });

          controller.close();
        },
      });
    }

    if (this._bodyText) {
      const text = this._bodyText;
      const encoder = new TextEncoder();

      return new ReadableStream({
        start: async (controller) => {
          const typedArray = new Uint8Array(encoder.encode(text));

          typedArray.forEach((chunk) => {
            controller.enqueue(chunk);
          });

          controller.close();
        },
      });
    }

    return null;
  }
}

export default Body;
