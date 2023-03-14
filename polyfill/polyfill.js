// polyfill URLSearchParams
require('url-search-params/polyfill.es6.min.js')

// polyfill Streams API
require('web-streams/polyfill.es6.min.js');

// polyfill TexeEncoder/TextDecoder
require('text-encoding/FastestTextEncoderPolyfill.es6.min.js')
require('text-encoding/FastestTextDecoderPolyfill.es6.min.js')

// polyfill Fetch API types (Headers, Request, Response)
require('react-native-fetch/polyfill.es6.min.js')

const __runtimeFetchEventInstance = new FetchEvent()
const __runtimeRequestInstance = new Request()
const __runtimeResponseInstance = new Response()