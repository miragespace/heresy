package promise

import (
	_ "embed"

	"github.com/dop251/goja"
)

const (
	promiseResolverFuncWithArgSymbol    = "__runtimeResolverFuncWithArg"
	promiseResolverFuncWithSpreadSymbol = "__runtimeResolverFuncWithSpread"
)

//go:embed wrapper.js
var promiseResolverScript string

var promiseResolverProg = goja.MustCompile("promiseResolver", promiseResolverScript, false)
