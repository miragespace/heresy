package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"go.miragespace.co/heresy"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

const testScript = `
"use strict";

async function httpHandler(url) {
	return fetch("https://example.com/")
}

registerRequestHandler(httpHandler)
`

func main() {
	args := os.Args

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	rt, err := heresy.NewRuntime(logger, "test.js", testScript)
	if err != nil {
		panic(err)
	}
	defer rt.Stop()

	router := chi.NewRouter()
	router.Mount("/debug", middleware.Profiler())
	router.Handle("/*", http.HandlerFunc(rt.Handler))

	addr := ":8081"
	if len(args) > 1 {
		addr = args[1]
	}

	logger.Info("ready", zap.String("addr", addr))

	http.ListenAndServe(addr, router)
}
