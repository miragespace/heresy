package heresy

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
)

func (rt *Runtime) TestStream() {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	instance := rt.instance

	f, err := os.Open("cmd/example/stream.js")
	if err != nil {
		panic(err)
	}

	w, err := instance.stream.NewReadableStream(f)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})

	err = instance.resolver.NewPromise(
		instance.vm.Get("drainStream"),
		w,
		instance.vm.ToValue(func(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
			var buf []byte
			vm.ExportTo(fc.Argument(0), &buf)
			fmt.Printf("result: %+s\n", buf)
			close(done)
			return goja.Undefined()
		}),
		instance.vm.ToValue(func(fc goja.FunctionCall) goja.Value {
			fmt.Printf("exploded: %+v\n", fc.Arguments)
			close(done)
			return goja.Undefined()
		}),
	)
	if err != nil {
		panic(err)
	}

	<-done
}
