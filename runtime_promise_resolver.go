package heresy

import "github.com/dop251/goja"

const nativePromiseResolverScript = `
const __runtimeResolver = (handler, req, resolve, reject) => {
    Promise.resolve(handler(req)).then(resolve).catch(reject)
}
`

var nativePromiseResolverProg = goja.MustCompile("runtime", nativePromiseResolverScript, false)
