package promise

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type PromiseResolver struct {
	eventLoop                *eventloop.EventLoop
	runtimeWrapperWithFunc   goja.Callable
	runtimeWrapperWithSpread goja.Callable
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

		promiseResolver := vm.Get(promiseResolverFuncWithArgSymbol)
		wrapper, ok := goja.AssertFunction(promiseResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", promiseResolverFuncWithArgSymbol)
			return
		}
		t.runtimeWrapperWithFunc = wrapper

		promiseResolver = vm.Get(promiseResolverFuncWithSpreadSymbol)
		wrapper, ok = goja.AssertFunction(promiseResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", promiseResolverFuncWithSpreadSymbol)
			return
		}
		t.runtimeWrapperWithSpread = wrapper

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (p *PromiseResolver) NewPromiseFuncWithArg(
	fn, arg, resolve, reject goja.Value,
) error {
	errCh := make(chan error, 1)
	p.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		errCh <- p.NewPromiseFuncWithArgVM(vm, fn, arg, resolve, reject)
	})

	return <-errCh
}

func (p *PromiseResolver) NewPromiseFuncWithArgVM(
	vm *goja.Runtime,
	fn, arg, resolve, reject goja.Value,
) error {
	_, err := p.runtimeWrapperWithFunc(
		goja.Undefined(),
		fn,
		arg,
		resolve,
		reject,
	)
	return err
}

func (p *PromiseResolver) NewPromiseFuncWithSpread(
	fn, arg, resolve, reject goja.Value,
) error {
	errCh := make(chan error, 1)
	p.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		errCh <- p.NewPromiseFuncWithSpreadVM(vm, fn, arg, resolve, reject)
	})

	return <-errCh
}

func (p *PromiseResolver) NewPromiseFuncWithSpreadVM(
	vm *goja.Runtime,
	fn, arg, resolve, reject goja.Value,
) error {
	_, err := p.runtimeWrapperWithSpread(
		goja.Undefined(),
		fn,
		arg,
		resolve,
		reject,
	)
	return err
}
