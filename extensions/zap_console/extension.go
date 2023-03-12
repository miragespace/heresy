package zap_console

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/util"
	"go.uber.org/zap"
)

const ModuleName = "node:console"

type Console struct {
	runtime *goja.Runtime
	util    *goja.Object
}

func (c *Console) log(log func(msg string, fields ...zap.Field)) func(goja.FunctionCall, *goja.Runtime) goja.Value {
	return func(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
		stacks := vm.CaptureCallStack(0, nil)
		caller := stacks[1]

		if format, ok := goja.AssertFunction(c.util.Get("format")); ok {
			ret, err := format(c.util, call.Arguments...)
			if err != nil {
				panic(err)
			}

			log(ret.String(),
				zap.String("position", caller.Position().String()),
				zap.String("funcName", caller.FuncName()),
				zap.String("script", caller.SrcName()),
			)
		} else {
			panic(c.runtime.NewTypeError("util.format is not a function"))
		}

		return nil
	}
}

func RequireWithLogger(logger *zap.Logger) require.ModuleLoader {
	return requireWithPrinter(logger)
}

func requireWithPrinter(logger *zap.Logger) require.ModuleLoader {
	return func(runtime *goja.Runtime, module *goja.Object) {
		c := &Console{
			runtime: runtime,
		}

		c.util = require.Require(runtime, util.ModuleName).(*goja.Object)

		o := module.Get("exports").(*goja.Object)
		o.Set("log", c.log(logger.Info))
		o.Set("error", c.log(logger.Error))
		o.Set("warn", c.log(logger.Warn))
	}
}

func Enable(runtime *goja.Runtime) {
	runtime.Set("console", require.Require(runtime, ModuleName))
}
