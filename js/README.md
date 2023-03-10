## This folder contains various polyfills for the goja runtime.

---

[url-search-params](https://github.com/jerrybendy/url-search-params-polyfill)

- Needed to bring `URLSearchParams` into the runtime

[text-encoding](https://github.com/anonyco/FastestSmallestTextEncoderDecoder)

- Needed to bring TextEncoder/TextDecoder (UTF-8) only to support converting between `string` and `ArrayBuffer`

[react-native-fetch](https://github.com/react-native-community/fetch)

- Modified to use TypeScript without Fetch implementation. Only `Headers`, `Request`, and `Response` are used.

[web-streams](https://github.com/MattiasBuelens/web-streams-polyfill/)

- Needed to support the Streams API in the runtime (e.g. `ReadableStream`)
