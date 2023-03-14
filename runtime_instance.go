package heresy

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.miragespace.co/heresy/event"
	"go.miragespace.co/heresy/express"
	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/shared"
	"go.miragespace.co/heresy/extensions/console"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
	"go.uber.org/zap"
)

var nilInstance *runtimeInstance = nil

type handlerType int

const (
	handlerTypeUnset handlerType = iota
	handlerTypeExpress
	handlerTypeEvent
)

type runtimeInstance struct {
	logger            *zap.Logger
	middlewareHandler atomic.Value                         // goja.Value
	middlewareType    atomic.Value                         // handlerType
	handlerOption     atomic.Pointer[nativeHandlerOptions] // nativeHandlerOptions
	ioContextPool     *common.IOContextPool
	contextPool       *express.RequestContextPool
	eventPool         *event.FetchEventPool
	eventLoop         *eventloop.EventLoop
	resolver          *promise.PromiseResolver
	stream            *stream.StreamController
	fetcher           *fetch.Fetch
	vm                *goja.Runtime
}

func (inst *runtimeInstance) stop(interrupt bool) {
	if interrupt {
		inst.vm.Interrupt(context.Canceled)
	}
	inst.eventLoop.StopNoWait()
}

func (inst *runtimeInstance) optionHelper(vm *goja.Runtime, opt goja.Value) {
	if goja.IsUndefined(opt) {
		return
	}
	var options nativeHandlerOptions
	if err := vm.ExportTo(opt, &options); err == nil {
		inst.handlerOption.Store(&options)
	}
}

func (inst *runtimeInstance) prepareInstance(logger *zap.Logger, symbols *polyfill.RuntimeSymbols) (setup chan error) {
	setup = make(chan error, 1)

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		url.Enable(vm)
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Set("console", require.Require(vm, console.ModuleName))

		vm.Set("registerExpressHandler", func(fc goja.FunctionCall) (ret goja.Value) {
			ret = goja.Undefined()

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
				inst.middlewareType.Store(handlerTypeExpress)
			}

			return
		})

		vm.Set("registerEventHandler", func(fc goja.FunctionCall) (ret goja.Value) {
			ret = goja.Undefined()

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
				inst.middlewareType.Store(handlerTypeEvent)
			}

			return
		})

		headersPool := shared.NewHeadersProxyPool(vm, symbols)
		inst.ioContextPool = common.NewIOContextPool(logger, headersPool, 10)
		inst.contextPool = express.NewRequestContextPool(express.RequestContextDeps{
			Logger:    logger,
			Eventloop: inst.eventLoop,
			Fetch:     inst.fetcher,
		})
		inst.eventPool = event.NewFetchEventPool(event.FetchEventDeps{
			Logger:    logger,
			Symbols:   symbols,
			Eventloop: inst.eventLoop,
			Stream:    inst.stream,
			Resolver:  inst.resolver,
			Fetch:     inst.fetcher,
		})

		inst.vm = vm // reference is kept for .Interrupt

		setup <- nil
	})

	return
}

func (inst *runtimeInstance) loadProgram(prog *goja.Program) (setup chan error) {
	setup = make(chan error, 1)

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up handler script: %w", err)
			return
		}
		setup <- nil
	})

	return
}
