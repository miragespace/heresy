package heresy

import (
	"text/template"

	"github.com/dop251/goja"
	pool "github.com/libp2p/go-buffer-pool"
)

const closure = `(arg)=>{const {fetch}=arg;return ({{.Function}})(arg)}`

var closureTemplate = template.Must(template.New("scope").Parse(closure))

// recompile the function when registerting to make fetch() ambient to the handler
func recompileWithClosure(scriptName string, fn goja.Value, vm *goja.Runtime) (goja.Value, error) {
	b := pool.NewBuffer(nil)
	defer b.Reset()

	closureTemplate.Execute(b, struct {
		Function string
	}{
		Function: fn.String(),
	})

	pp, err := goja.Compile(scriptName, b.String(), true)
	if err != nil {
		return nil, err
	}

	newFn, err := vm.RunProgram(pp)
	if err != nil {
		return nil, err
	}

	return newFn, nil
}
