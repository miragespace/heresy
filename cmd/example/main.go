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

	"go.miragespace.co/heresy"
	"go.miragespace.co/heresy/extensions/kv"
	_ "go.miragespace.co/heresy/extensions/kv/memory"

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

	kvManager := kv.NewKVManager()
	if err := kvManager.Configure("potato", "memory"); err != nil {
		panic(err)
	}

	rt, err := heresy.NewRuntime(logger, kvManager, 4)
	if err != nil {
		panic(err)
	}
	defer rt.Stop(true)

	router := chi.NewRouter()
	router.Mount("/debug", middleware.Profiler())
	router.Mount("/reload", http.HandlerFunc(reloadScript(logger, rt)))

	index := chi.NewRouter()
	index.Use(rt.Middleware)
	index.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "index")
	}))

	router.Handle("/*", index)

	addr := ":8081"
	if len(args) > 1 {
		addr = args[1]
	}

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
		scriptBytes, err := io.ReadAll(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to read script from body: %v", err)
			return
		}
		script = string(scriptBytes)

		err = rt.LoadScript(p.FileName(), script, true)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error reloading script: %v", err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
