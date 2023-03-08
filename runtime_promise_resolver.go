package heresy

import "github.com/dop251/goja"

const nativePromiseResolverScript = `
const __runtimeResolver = (handler, ctx, resolve, reject) => {
    Promise.resolve(handler(ctx)).then(resolve).catch(reject)
}
`

var nativePromiseResolverProg = goja.MustCompile("runtime", nativePromiseResolverScript, false)
