package heresy

import (
	"embed"

	"github.com/dop251/goja"
)

//go:embed node_modules/*
var modulesFS embed.FS

const modulesExporterScript = `
// polyfill URLSearchParams
require('url-search-params-polyfill/index.js')

// polyfill Streams API
require('web-streams/polyfill.es6.min.js');
`

func loadModulesExporter() (*goja.Program, error) {
	return goja.Compile("modules", modulesExporterScript, true)
}
