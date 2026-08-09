package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type discardResponseWriter struct {
	header http.Header
	status int
	bytes  int64
}

func (w *discardResponseWriter) Header() http.Header { return w.header }

func (w *discardResponseWriter) WriteHeader(status int) { w.status = status }

func (w *discardResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.bytes += int64(len(data))
	return len(data), nil
}

func (w *discardResponseWriter) Flush() {}

func TestDownloadStreamsForConfiguredDuration(t *testing.T) {
	s, err := newServer(25*time.Millisecond, 1024, 1)
	if err != nil {
		t.Fatal(err)
	}

	w := &discardResponseWriter{header: make(http.Header)}
	request := httptest.NewRequest(http.MethodGet, "/download_test", nil)
	start := time.Now()
	s.handleDownload(w, request)
	elapsed := time.Since(start)

	if w.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.status, http.StatusOK)
	}
	if w.bytes == 0 {
		t.Fatal("download did not stream any bytes")
	}
	if elapsed < 20*time.Millisecond || elapsed > time.Second {
		t.Fatalf("download duration = %s, want approximately 25ms", elapsed)
	}
	if got := w.Header().Get("Content-Encoding"); got != "identity" {
		t.Fatalf("Content-Encoding = %q, want identity", got)
	}
}

func TestDownloadRejectsNonGet(t *testing.T) {
	s, err := newServer(time.Second, 1024, 1)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.handleDownload(w, httptest.NewRequest(http.MethodPost, "/download_test", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
