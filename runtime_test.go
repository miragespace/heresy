package heresy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
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

	rt, err := NewRuntime(logger, 1)
	as.NoError(err)
	defer rt.Stop(true)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Use(rt.Middleware)
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	router.ServeHTTP(w, req)

	as.Equal(http.StatusServiceUnavailable, w.Result().StatusCode)
}

func TestRuntimeNoHandler(t *testing.T) {
	as := require.New(t)
	logger := zaptest.NewLogger(t)

	rt, err := NewRuntime(logger, 1)
	as.NoError(err)
	defer rt.Stop(true)

	err = rt.LoadScript("no_handler.js", testScriptNoHandler, false)
	as.NoError(err)

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	router.Use(rt.Middleware)
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	router.ServeHTTP(w, req)

	as.Equal(http.StatusBadGateway, w.Result().StatusCode)
}

func TestRuntimeScriptReload(t *testing.T) {
	as := require.New(t)
	logger := zaptest.NewLogger(t)

	rt, err := NewRuntime(logger, 1)
	as.NoError(err)
	defer rt.Stop(true)

	err = rt.LoadScript("promise.js", testScriptPromise, false)
	as.NoError(err)

	router := chi.NewRouter()
	router.Use(rt.Middleware)
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	as.Equal(http.StatusOK, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	as.NoError(err)
	as.Equal("promise", string(body))

	err = rt.LoadScript("plain.js", testScriptPlain, false)
	as.NoError(err)

	req = httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	as.Equal(http.StatusOK, w.Result().StatusCode)
	body, err = io.ReadAll(w.Result().Body)
	as.NoError(err)
	as.Equal("plain", string(body))
}
