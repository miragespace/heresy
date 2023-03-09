package modules

import (
	"embed"

	"github.com/dop251/goja"
)

//go:embed node_modules/*
var ModulesFS embed.FS

const modulesExporterScript = `
// polyfill URLSearchParams
require('url-search-params-polyfill/index.js')

// polyfill Streams API
require('web-streams/polyfill.es6.min.js');

// polyfill Fetch API
require('fetch/polyfill.es6.min.js')
`

func LoadModulesExporter() (*goja.Program, error) {
	return goja.Compile("modules", modulesExporterScript, true)
}
