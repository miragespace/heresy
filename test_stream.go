package heresy

import (
	"fmt"
	"net/http"
	"os"

	"github.com/dop251/goja"
)

func (rt *Runtime) TestStream(w http.ResponseWriter, r *http.Request) {
	rt.shardRun(func(instance *runtimeInstance) {
		f, err := os.Open("cmd/example/stream.js")
		if err != nil {
			panic(err)
		}

		s, err := instance.stream.NewReadableStream(f)
		if err != nil {
			panic(err)
		}

		done := make(chan struct{})

		err = instance.resolver.NewPromise(
			instance._testDrainStream,
			s,
			instance.vm.ToValue(func(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
				var buf []byte
				vm.ExportTo(fc.Argument(0), &buf)
				w.Write(buf)
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
	})
}
