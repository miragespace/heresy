package main

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"go.miragespace.co/heresy"
	"go.uber.org/zap"
)

func TestReloadScript(t *testing.T) {
	rt, err := heresy.NewRuntime(zap.NewNop(), nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.Stop(true) })
	for _, tt := range []struct {
		name, method, field, source string
		status                      int
	}{
		{"method", "GET", "file", "", 405},
		{"empty multipart", "PUT", "", "", 400},
		{"wrong field", "PUT", "other", "", 400},
		{"invalid script", "PUT", "file", "function {", 400},
		{"oversized script", "PUT", "file", strings.Repeat(" ", 1<<20), 413},
		{"valid script", "PUT", "file", `registerEventHandler(() => {})`, 202},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			if tt.field != "" {
				part, err := form.CreateFormFile(tt.field, "test.js")
				if err != nil {
					t.Fatal(err)
				}
				part.Write([]byte(tt.source))
			}
			form.Close()
			req := httptest.NewRequest(tt.method, "/reload", &body)
			req.Header.Set("Content-Type", form.FormDataContentType())
			w := httptest.NewRecorder()
			reloadScript(zap.NewNop(), rt)(w, req)
			if w.Code != tt.status {
				t.Fatalf("got %d %q, want %d", w.Code, w.Body.String(), tt.status)
			}
		})
	}
}
