package promise

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type PromiseResolver struct {
	eventLoop              *eventloop.EventLoop
	runtimeWarpperResult   goja.Callable
	runtimeWrapperWithFunc goja.Callable
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

		// NOTE: this is better than running the same block of code N times
		// trying to get the helper function out and assign it
		for _, assignment := range []struct {
			target *goja.Callable
			name   string
		}{
			{
				target: &t.runtimeWarpperResult,
				name:   promiseResolverResultSymbol,
			},
			{
				target: &t.runtimeWrapperWithFunc,
				name:   promiseResolverFuncWithArgSymbol,
			},
		} {
			promiseResolver := vm.Get(assignment.name)
			wrapper, ok := goja.AssertFunction(promiseResolver)
			if !ok {
				setup <- fmt.Errorf("internal error: %s is not a function", assignment.name)
				return
			}
			*assignment.target = wrapper
		}

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (p *PromiseResolver) NewPromiseResultVM(
	vm *goja.Runtime,
	arg, resolve, reject goja.Value,
) error {
	_, err := p.runtimeWarpperResult(
		goja.Undefined(),
		arg,
		resolve,
		reject,
	)
	return err
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

// func (p *PromiseResolver) NewPromiseFuncWithSpread(
// 	fn, arg, resolve, reject goja.Value,
// ) error {
// 	errCh := make(chan error, 1)
// 	p.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
// 		errCh <- p.NewPromiseFuncWithSpreadVM(vm, fn, arg, resolve, reject)
// 	})

// 	return <-errCh
// }
