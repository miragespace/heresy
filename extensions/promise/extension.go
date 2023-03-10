package promise

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type PromiseResolver struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
}

func NewResolver(eventLoop *eventloop.EventLoop) (*PromiseResolver, error) {
	t := &PromiseResolver{
		eventLoop: eventLoop,
	}

	setup := make(chan error, 1)
	eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(promiseResolverProg)
		if err != nil {
			setup <- err
			return
		}

		promiseResolver := vm.Get(promiseResolverSymbol)
		wrapper, ok := goja.AssertFunction(promiseResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", promiseResolverSymbol)
			return
		}
		t.runtimeWrapper = wrapper

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (p *PromiseResolver) NewPromise(
	fn, arg, resolve, reject goja.Value,
) error {
	errCh := make(chan error, 1)
	p.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		errCh <- p.NewPromiseVM(vm, fn, arg, resolve, reject)
	})

	return <-errCh
}

func (p *PromiseResolver) NewPromiseVM(
	vm *goja.Runtime,
	fn, arg, resolve, reject goja.Value,
) error {
	_, err := p.runtimeWrapper(
		goja.Undefined(),
		fn,
		arg,
		resolve,
		reject,
	)
	return err
}
