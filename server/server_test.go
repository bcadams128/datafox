package main

import (
	"bytes"
	"datafox/server/pkg/disk"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func newTestServer() *Server {
	ch := make(chan disk.LogBatch, 10)
	return &Server{ingestChan: ch}
}

func TestHandlePing(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	srv.handlePing(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != "Pong" {
		t.Errorf("expected body Pong, got %q", body)
	}
}

func TestHandleLogs(t *testing.T) {
	batch := disk.LogBatch{
		Lines:    []string{"line one", "line two"},
		Metadata: map[string]string{"hostname": "myhost", "source": "var/log/app.log"},
	}

	t.Run("Happy path", func(t *testing.T) {
		srv := newTestServer()

		var buf bytes.Buffer
		msgpack.NewEncoder(&buf).Encode(batch)

		req := httptest.NewRequest(http.MethodPost, "/logs", &buf)
		rec := httptest.NewRecorder()
		srv.handleLogs(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d", rec.Code)
		}
	})
	t.Run("bad payload returns 400", func(t *testing.T) {
		//Payload is not GZIPPed
		//Partial Bytes
		//JSON payload
		//Content-type is not gzip
		cases := []struct {
			name string
			body string
		}{
			{"random text", "not msgpack"},
			{"json", `{"lines": ["foo"]}`},
		}
		srv := newTestServer()

		for _, tests := range cases {
			t.Run(tests.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader(tests.body))
				rec := httptest.NewRecorder()
				srv.handleLogs(rec, req)
				fmt.Printf("case code returned %d \n", rec.Code)
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d", rec.Code)
				}
			})
		}
	})
}
