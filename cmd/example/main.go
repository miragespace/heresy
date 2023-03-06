package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.miragespace.co/heresy"
	"go.uber.org/zap"
)

const testScript = `
"use strict";

async function httpHandler(url) {
    const results = await Promise.all([
        fetch("http://google.com"),
        fetch("http://baidu.com"),
    ])
    return results.join(", ")
}

onRequest(httpHandler)
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
	router.Handle("/*", http.HandlerFunc(rt.Handle))

	addr := ":8081"
	if len(args) > 1 {
		addr = args[1]
	}

	logger.Info("ready", zap.String("addr", addr))

	http.ListenAndServe(addr, router)
}
