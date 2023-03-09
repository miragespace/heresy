package heresy

import "github.com/dop251/goja"

const nativePromiseResolverSymbol = "__runtimeResolver"

const nativePromiseResolverScript = `
const __runtimeResolver = (handler, ctx, resolve, reject) => {
    Promise.resolve(handler(ctx)).then(resolve).catch(reject)
}
`

func loadNativePromiseResolver() (*goja.Program, error) {
	return goja.Compile("resolver", nativePromiseResolverScript, true)
}
