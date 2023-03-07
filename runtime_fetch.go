package heresy

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type fetchConfig struct {
	eventLoop *eventloop.EventLoop
	scheduler *pond.WorkerPool
	client    *http.Client
}

func (f *fetchConfig) doFetch(req goja.Value) (any, error) {
	var (
		r   *http.Request
		err error
	)
	v := req.ExportType()
	switch v.Kind() {
	case reflect.String:
		r, err = http.NewRequest("GET", req.String(), nil)
	default:
		err = fmt.Errorf("not implemented")
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return string(body), nil
}

func (f *fetchConfig) runtimeWrapper(req goja.Value, resolve, reject func(interface{})) {
	f.scheduler.Submit(func() {
		result, err := f.doFetch(req)
		if err != nil {
			f.eventLoop.RunOnLoop(func(*goja.Runtime) {
				reject(err)
			})
			return
		}
		f.eventLoop.RunOnLoop(func(*goja.Runtime) {
			resolve(result)
		})
	})
}

func withFetch(rt *eventloop.EventLoop, vm *goja.Runtime, cfg fetchConfig) {
	if cfg.client == nil {
		cfg.client = &http.Client{
			Timeout: time.Second * 15,
		}
	}
	fetch := func(req goja.Value) *goja.Promise {
		promise, resolve, reject := vm.NewPromise()
		cfg.runtimeWrapper(req, resolve, reject)
		return promise
	}
	vm.Set("fetch", fetch)
}
