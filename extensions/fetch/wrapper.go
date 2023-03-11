package fetch

import (
	_ "embed"

	"github.com/dop251/goja"
)

const (
	fetchWrapperSymbol = "__runtimeFetch"
)

//go:embed wrapper.js
var fetchWrapperScript string

var fetchWrapperProg = goja.MustCompile("fetch", fetchWrapperScript, false)
