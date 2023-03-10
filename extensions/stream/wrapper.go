package stream

import (
	_ "embed"

	"github.com/dop251/goja"
)

const (
	streamWrapperSymbol = "__runtimeIOReaderWrapper"
)

//go:embed wrapper.js
var streamWrapperScript string

var streamWrapperProg = goja.MustCompile("streamWrapper", streamWrapperScript, true)
