package heresy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const testScriptNoHandler = `
"use strict"
`

const testScriptPromise = `
"use strict"

registerRequestHandler(async () => {
	return new Promise((resolve) => {
		setTimeout(() => {
			resolve("promise")
		}, 50)
	})
})
`

const testScriptPlain = `
"use strict"

registerRequestHandler(() => {
	return "plain"
})
`

func TestEmptyRuntime(t *testing.T) {
	as := require.New(t)
	logger := zaptest.NewLogger(t)

	rt, err := NewRuntime(logger)
	as.NoError(err)
	defer rt.Stop()

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	as.Equal(http.StatusServiceUnavailable, w.Result().StatusCode)
}

func TestRuntimeNoHandler(t *testing.T) {
	as := require.New(t)
	logger := zaptest.NewLogger(t)

	rt, err := NewRuntime(logger)
	as.NoError(err)
	defer rt.Stop()

	err = rt.LoadScript("no_handler.js", testScriptNoHandler)
	as.NoError(err)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	as.Equal(http.StatusBadGateway, w.Result().StatusCode)
}

func TestRuntimeScriptReload(t *testing.T) {
	as := require.New(t)
	logger := zaptest.NewLogger(t)

	rt, err := NewRuntime(logger)
	as.NoError(err)
	defer rt.Stop()

	err = rt.LoadScript("promise.js", testScriptPromise)
	as.NoError(err)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	as.Equal(http.StatusOK, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	as.NoError(err)
	as.Equal("promise", string(body))

	err = rt.LoadScript("plain.js", testScriptPlain)
	as.NoError(err)

	req = httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w = httptest.NewRecorder()

	rt.Handler(w, req)

	as.Equal(http.StatusOK, w.Result().StatusCode)
	body, err = io.ReadAll(w.Result().Body)
	as.NoError(err)
	as.Equal("plain", string(body))
}
