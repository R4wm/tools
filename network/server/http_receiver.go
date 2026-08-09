package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultPort             = 8080
	defaultListenAddress    = "127.0.0.1"
	defaultDownloadDuration = 10 * time.Second
	defaultBufferSize       = 64 * 1024
	defaultMaxDownloads     = 20
)

type server struct {
	downloadDuration time.Duration
	downloadBuffer   []byte
	downloadSlots    chan struct{}
}

type uploadResult struct {
	BytesReceived   int64   `json:"bytes_received"`
	DurationSeconds float64 `json:"duration_seconds"`
	SpeedMbps       float64 `json:"speed_mbps"`
	SpeedMBps       float64 `json:"speed_mbytes_per_sec"`
}

func newServer(downloadDuration time.Duration, bufferSize, maxDownloads int) (*server, error) {
	if downloadDuration <= 0 {
		return nil, errors.New("download duration must be positive")
	}
	if bufferSize <= 0 {
		return nil, errors.New("download buffer size must be positive")
	}
	if maxDownloads <= 0 {
		return nil, errors.New("maximum concurrent downloads must be positive")
	}

	buffer := make([]byte, bufferSize)
	if _, err := rand.Read(buffer); err != nil {
		return nil, fmt.Errorf("create download buffer: %w", err)
	}

	return &server{
		downloadDuration: downloadDuration,
		downloadBuffer:   buffer,
		downloadSlots:    make(chan struct{}, maxDownloads),
	}, nil
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	bytesReceived, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		log.Printf("upload from %s failed: %v", r.RemoteAddr, err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start).Seconds()
	result := uploadResult{BytesReceived: bytesReceived, DurationSeconds: duration}
	if duration > 0 {
		result.SpeedMbps = (float64(bytesReceived) * 8) / duration / 1_000_000
		result.SpeedMBps = float64(bytesReceived) / duration / (1024 * 1024)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("write upload response to %s: %v", r.RemoteAddr, err)
	}
}

// handleDownload streams arbitrary bytes for the configured duration. It does
// not set Content-Length, so net/http uses a streaming response. A client must
// count the bytes it reads; query parameters are deliberately ignored.
func (s *server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case s.downloadSlots <- struct{}{}:
		defer func() { <-s.downloadSlots }()
	default:
		w.Header().Set("Retry-After", "1")
		http.Error(w, "download test capacity reached", http.StatusServiceUnavailable)
		return
	}

	// Keep intermediaries from caching, compressing, or buffering the test
	// payload. Compression would make the byte count unsuitable for a speed test.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "identity")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	deadline := time.Now().Add(s.downloadDuration)
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		log.Printf("set download write deadline for %s: %v", r.RemoteAddr, err)
	}

	var bytesWritten int64
	for time.Now().Before(deadline) {
		written, err := w.Write(s.downloadBuffer)
		bytesWritten += int64(written)
		if err != nil {
			if r.Context().Err() == nil {
				log.Printf("download to %s stopped after %d bytes: %v", r.RemoteAddr, bytesWritten, err)
			}
			return
		}
		if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
			log.Printf("flush download to %s: %v", r.RemoteAddr, err)
			return
		}
	}

	log.Printf("download to %s completed: %d bytes in %s", r.RemoteAddr, bytesWritten, s.downloadDuration)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, "pong\n")
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, "OK\n")
	}
}

func main() {
	port := flag.Int("port", defaultPort, "TCP port to listen on")
	listenAddress := flag.String("listen-address", defaultListenAddress, "IP address to listen on")
	downloadDuration := flag.Duration("download-duration", defaultDownloadDuration, "duration of each download stream")
	bufferSize := flag.Int("download-buffer-size", defaultBufferSize, "reused payload buffer size in bytes")
	maxDownloads := flag.Int("max-downloads", defaultMaxDownloads, "maximum simultaneous download streams")
	flag.Parse()

	s, err := newServer(*downloadDuration, *bufferSize, *maxDownloads)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload_test", s.handleUpload)
	mux.HandleFunc("GET /download_test", s.handleDownload)
	mux.HandleFunc("GET /ping", handlePing)
	mux.HandleFunc("HEAD /ping", handlePing)
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("HEAD /health", handleHealth)

	address := net.JoinHostPort(*listenAddress, strconv.Itoa(*port))
	log.Printf("speed test server listening on %s (download duration: %s, maximum streams: %d)", address, *downloadDuration, *maxDownloads)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
