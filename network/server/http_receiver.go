package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type UploadResult struct {
	BytesReceived   int64   `json:"bytes_received"`
	DurationSeconds float64 `json:"duration_seconds"`
	SpeedMbps       float64 `json:"speed_mbps"`
	SpeedMBps       float64 `json:"speed_mbytes_per_sec"`
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Upload request from %s", r.RemoteAddr)

	start := time.Now()

	// Discard uploaded data efficiently
	bytesReceived, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start).Seconds()
	speedBytesPerSec := float64(bytesReceived) / duration
	speedMbps := (speedBytesPerSec * 8) / (1000 * 1000)
	speedMBps := speedBytesPerSec / (1024 * 1024)

	result := UploadResult{
		BytesReceived:   bytesReceived,
		DurationSeconds: duration,
		SpeedMbps:       speedMbps,
		SpeedMBps:       speedMBps,
	}

	log.Printf("Upload completed: %.2f MB in %.2f seconds (%.2f Mbps, %.2f MB/s)",
		float64(bytesReceived)/(1024*1024), duration, speedMbps, speedMBps)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get size from query parameter
	sizeStr := r.URL.Query().Get("size")
	if sizeStr == "" {
		sizeStr = "104857600" // Default 100MB
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid size parameter", http.StatusBadRequest)
		return
	}

	// Limit maximum size to 10GB for safety
	if size > 10*1024*1024*1024 {
		http.Error(w, "Size too large (max 10GB)", http.StatusBadRequest)
		return
	}

	log.Printf("Download request from %s for %d bytes (%.2f MB)",
		r.RemoteAddr, size, float64(size)/(1024*1024))

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.WriteHeader(http.StatusOK)

	// Stream random data
	written, err := io.CopyN(w, rand.Reader, size)
	if err != nil {
		log.Printf("Error sending data: %v (sent %d bytes)", err, written)
		return
	}

	log.Printf("Download completed: sent %.2f MB to %s",
		float64(written)/(1024*1024), r.RemoteAddr)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong\n"))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func main() {
	port := flag.Int("port", 8080, "Server port")
	flag.Parse()

	http.HandleFunc("/upload_test", handleUpload)
	http.HandleFunc("/download_test", handleDownload)
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("HTTP speed test server listening on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  POST /upload_test        - Upload speed test endpoint")
	log.Printf("  GET  /download_test?size - Download speed test endpoint")
	log.Printf("  HEAD /ping               - Latency test endpoint")
	log.Printf("  GET  /health             - Health check")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
