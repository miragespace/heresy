package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	_ "net/http/pprof"
	"os"

	"go.miragespace.co/heresy"

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
	router.Handle("/*", http.HandlerFunc(rt.Handler))

	addr := ":8081"
	if len(args) > 1 {
		addr = args[1]
	}

	logger.Info("ready", zap.String("addr", addr))

	http.ListenAndServe(addr, router)
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

		script, err := io.ReadAll(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to read script from body: %v", err)
			return
		}

		err = rt.LoadScript(p.FileName(), string(script))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error reloading script: %v", err)
			return
		}

		logger.Info("script loaded", zap.String("filename", p.FileName()), zap.Int("size", len(script)))
		w.WriteHeader(http.StatusAccepted)
	}
}
