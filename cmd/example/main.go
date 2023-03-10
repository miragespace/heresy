package main

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"

	"go.miragespace.co/heresy"
	"go.miragespace.co/heresy/transpile"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	args := os.Args

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	rt, err := heresy.NewRuntime(logger)
	if err != nil {
		panic(err)
	}
	defer rt.Stop()

	router := chi.NewRouter()
	router.Mount("/debug", middleware.Profiler())
	router.Mount("/reload", http.HandlerFunc(reloadScript(logger, rt)))
	router.Mount("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt.TestStream()
	}))

	index := chi.NewRouter()
	index.Use(rt.Middleware)
	index.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "index")
	}))

	router.Handle("/*", index)

	addr := ":8081"
	if len(args) > 1 {
		addr = args[1]
	}

	logger.Info("ready", zap.String("addr", addr))

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	defer srv.Shutdown(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	logger.Info("received signal to stop", zap.String("signal", (<-sig).String()))
}

func reloadScript(logger *zap.Logger, rt *heresy.Runtime) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var form *multipart.Reader
		form, err := r.MultipartReader()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Request is not multipart")
			return
		}

		var p *multipart.Part
		p, err = form.NextPart()
		if err != nil && err != io.EOF {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}

		if p.FormName() != "file" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Expecting \"file\" field in request")
			return
		}

		var script string
		if strings.HasSuffix(p.FileName(), ".ts") {
			// transpile typescript
			tCtx, tCancel := context.WithTimeout(r.Context(), time.Second*5)
			defer tCancel()
			script, err = transpile.TranspileTypescript(tCtx, p)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Failed to transpile TypeScript: %v", err)
				return
			}
		} else {
			scriptBytes, err := io.ReadAll(p)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Failed to read script from body: %v", err)
				return
			}
			script = string(scriptBytes)
		}

		err = rt.LoadScript(p.FileName(), script)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error reloading script: %v", err)
			return
		}

		logger.Info("script loaded", zap.String("filename", p.FileName()), zap.Int("size", len(script)))
		w.WriteHeader(http.StatusAccepted)
	}
}
