package heresy

import "github.com/dop251/goja"

const runtimeResolverScript = `
const __runtimeResolver = (handler, req, resolve, reject) => {
    Promise.resolve(handler(req)).then(resolve).catch(reject)
}
`

var runtimeResolverProg = goja.MustCompile("runtime", runtimeResolverScript, false)
