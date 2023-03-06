package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func WithFetch(rt *eventloop.EventLoop, vm *goja.Runtime, client *http.Client) {
	fetch := func(url string) *goja.Promise {
		promise, resolve, _ := vm.NewPromise()
		go func() {
			rt.RunOnLoop(func(*goja.Runtime) {
				resolve(fmt.Sprintf("url %s has result: %s", url, "tada!"))
			})
		}()
		return promise
	}
	vm.Set("fetch", fetch)
}
