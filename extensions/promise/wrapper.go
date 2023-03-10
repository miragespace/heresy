package promise

import (
	_ "embed"

	"github.com/dop251/goja"
)

const (
	promiseResolverSymbol = "__runtimeResolver"
)

//go:embed wrapper.js
var promiseResolverScript string

var promiseResolverProg = goja.MustCompile("promiseResolver", promiseResolverScript, false)
