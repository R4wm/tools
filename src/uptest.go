package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Constants
const (
	DefaultDataSize = 1 * 1024 * 1024 * 1024 // 1GB
	DefaultTimeout  = 5 * time.Minute
	DefaultServer   = "central.prsmusa.com"
	DefaultPort     = 443
)

// Config holds all configuration options
type Config struct {
	// Client identity
	ClientID string

	// Remote configuration
	RemoteConfigURL     string
	RemoteConfigEnabled bool
	ConfigCachePath     string

	// Test configuration
	Method   string // Deprecated: use Protocol
	Protocol string // "http", "tcp", or "both"
	Server   string
	Port     int

	// TCP-specific ports
	TCPUploadPort   int
	TCPDownloadPort int

	// Data size configuration
	DataSize         int64 // Deprecated: use UploadDataSize/DownloadDataSize
	UploadDataSize   int64
	DownloadDataSize int64

	Timeout time.Duration

	// Test type flags
	TestUpload   bool
	TestDownload bool
	TestLatency  bool
	TestAll      bool

	// Diagnostics configuration
	LatencySamples  int
	LatencyTimeout  time.Duration

	// Redis configuration
	RedisEnabled bool
	RedisAddr    string
	RedisDB      int
	RedisPrefix  string

	// GitHub configuration
	GitHubEnabled        bool
	GitHubRepoPath       string
	GitHubRepoURL        string
	GitHubBranch         string
	GitHubAutoPush       bool
	GitHubIncludeSummary bool
	GitHubUpdateREADME   bool

	// Output configuration
	OutputFormat string
	ShowProgress bool
	Verbose      bool
	Silent       bool

	// Daemon mode
	Daemon   bool
	Interval time.Duration
}

// ClientMetadata contains machine fingerprint information
type ClientMetadata struct {
	ClientID string
	IP       string
	Hostname string
	OS       string
}

// LatencyStats holds comprehensive latency measurements
type LatencyStats struct {
	Samples  []float64 `json:"samples"`
	Count    int       `json:"count"`
	MinMs    float64   `json:"min_ms"`
	MaxMs    float64   `json:"max_ms"`
	AvgMs    float64   `json:"avg_ms"`
	MedianMs float64   `json:"median_ms"`
	StdDevMs float64   `json:"stddev_ms"`
	JitterMs float64   `json:"jitter_ms"`
	P95Ms    float64   `json:"p95_ms"`
	P99Ms    float64   `json:"p99_ms"`
}

// PacketLossStats holds packet loss information
type PacketLossStats struct {
	TotalAttempts   int     `json:"total_attempts"`
	FailedAttempts  int     `json:"failed_attempts"`
	LossPercentage  float64 `json:"loss_percentage"`
	TimeoutCount    int     `json:"timeout_count"`
	ConnectionDrops int     `json:"connection_drops,omitempty"`
}

// TestResult stores the outcome of a speed test
type TestResult struct {
	TestID       string                 `json:"test_id"`
	Timestamp    time.Time              `json:"timestamp"`
	TestType     string                 `json:"test_type"`
	Server       string                 `json:"server"`
	Protocol     string                 `json:"protocol"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// Upload metrics
	UploadSpeedBytesPerSec float64 `json:"upload_speed_bytes_per_sec,omitempty"`
	UploadSpeedMbps        float64 `json:"upload_speed_mbps,omitempty"`
	UploadDataSizeBytes    int64   `json:"upload_data_size_bytes,omitempty"`
	UploadDurationSeconds  float64 `json:"upload_duration_seconds,omitempty"`

	// Download metrics
	DownloadSpeedBytesPerSec float64 `json:"download_speed_bytes_per_sec,omitempty"`
	DownloadSpeedMbps        float64 `json:"download_speed_mbps,omitempty"`
	DownloadDataSizeBytes    int64   `json:"download_data_size_bytes,omitempty"`
	DownloadDurationSeconds  float64 `json:"download_duration_seconds,omitempty"`

	// Diagnostics
	LatencyStats LatencyStats    `json:"latency_stats,omitempty"`
	PacketLoss   PacketLossStats `json:"packet_loss,omitempty"`

	// Deprecated fields (kept for backward compatibility)
	DataSizeBytes   int64   `json:"data_size_bytes,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	LatencyMs       float64 `json:"latency_ms,omitempty"`
}

// RemoteConfig represents configuration fetched from server
type RemoteConfig struct {
	Version    string `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	Uninstall  bool   `json:"uninstall"`
	ClientName string `json:"client_name"`
	Enabled    bool   `json:"enabled"`
	Method     string `json:"method"`
	Server     string `json:"server"`
	Port       int    `json:"port"`
	DataSize   int64  `json:"data_size"`
	Timeout    int    `json:"timeout"`

	Redis struct {
		Enabled bool   `json:"enabled"`
		Addr    string `json:"addr"`
		DB      int    `json:"db"`
		Prefix  string `json:"prefix"`
	} `json:"redis"`

	GitHub struct {
		Enabled        bool   `json:"enabled"`
		RepoURL        string `json:"repo_url"`
		Branch         string `json:"branch"`
		AutoPush       bool   `json:"auto_push"`
		IncludeSummary bool   `json:"include_summary"`
		UpdateREADME   bool   `json:"update_readme"`
	} `json:"github"`

	Daemon struct {
		Interval int `json:"interval"`
	} `json:"daemon"`
}

// RemoteConfigFetcher handles fetching configuration from server
type RemoteConfigFetcher struct {
	url       string
	cachePath string
	timeout   time.Duration
}

// generateClientID creates a deterministic client ID from machine fingerprint
func generateClientID() (*ClientMetadata, error) {
	metadata := &ClientMetadata{}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	metadata.Hostname = hostname

	// Get primary IP address
	ip, err := getOutboundIP()
	if err != nil {
		ip = "unknown"
	}
	metadata.IP = ip

	// Get OS
	metadata.OS = runtime.GOOS

	// Generate deterministic client ID from machine fingerprint
	fingerprint := fmt.Sprintf("%s|%s|%s", ip, hostname, metadata.OS)
	hash := sha256.Sum256([]byte(fingerprint))
	metadata.ClientID = fmt.Sprintf("%x", hash[:16])

	if !flag.Lookup("silent").Value.(flag.Getter).Get().(bool) {
		log.Printf("Client ID: %s (IP: %s, Hostname: %s, OS: %s)",
			metadata.ClientID, metadata.IP, metadata.Hostname, metadata.OS)
	}

	return metadata, nil
}

// getOutboundIP returns the IP address used for internet connections
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// NewRemoteConfigFetcher creates a new config fetcher
func NewRemoteConfigFetcher(url string, cachePath string) *RemoteConfigFetcher {
	return &RemoteConfigFetcher{
		url:       url,
		cachePath: cachePath,
		timeout:   10 * time.Second,
	}
}

// Fetch retrieves configuration from remote server
func (r *RemoteConfigFetcher) Fetch(metadata *ClientMetadata) (*RemoteConfig, error) {
	// Build URL with client ID parameter
	url := fmt.Sprintf("%s?client_id=%s", r.url, metadata.ClientID)

	// Create HTTP request with metadata headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return r.loadFromCache()
	}

	// Add client metadata as headers
	req.Header.Set("X-Client-IP", metadata.IP)
	req.Header.Set("X-Client-Hostname", metadata.Hostname)
	req.Header.Set("X-Client-OS", metadata.OS)
	req.Header.Set("User-Agent", fmt.Sprintf("uptest/1.0 (%s)", metadata.OS))

	// HTTP GET to remote config URL
	client := &http.Client{Timeout: r.timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Remote config fetch failed: %v, using cached config", err)
		return r.loadFromCache()
	}
	defer resp.Body.Close()

	// Handle 404 (no client-specific config)
	if resp.StatusCode == 404 {
		log.Printf("No config found for client %s, using local config", metadata.ClientID)
		return nil, fmt.Errorf("no remote config for client")
	}

	// Parse JSON response
	var config RemoteConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		log.Printf("Failed to parse remote config: %v", err)
		return r.loadFromCache()
	}

	// Validate config version
	if config.Version != "1.0" {
		log.Printf("Warning: Config version mismatch (got %s, expected 1.0)", config.Version)
	}

	// Cache the config locally
	r.saveToCache(&config)

	return &config, nil
}

// loadFromCache loads configuration from local cache
func (r *RemoteConfigFetcher) loadFromCache() (*RemoteConfig, error) {
	data, err := os.ReadFile(r.cachePath)
	if err != nil {
		return nil, err
	}

	var config RemoteConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}

// saveToCache saves configuration to local cache
func (r *RemoteConfigFetcher) saveToCache(config *RemoteConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(r.cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.cachePath, data, 0644)
}

// MergeRemote merges remote config into local config (remote takes precedence)
func (c *Config) MergeRemote(remote *RemoteConfig) {
	if remote.Server != "" {
		c.Server = remote.Server
	}
	if remote.Port > 0 {
		c.Port = remote.Port
	}
	if remote.DataSize > 0 {
		c.DataSize = remote.DataSize
	}
	if remote.Timeout > 0 {
		c.Timeout = time.Duration(remote.Timeout) * time.Second
	}

	c.RedisEnabled = remote.Redis.Enabled
	if remote.Redis.Addr != "" {
		c.RedisAddr = remote.Redis.Addr
	}
	c.RedisDB = remote.Redis.DB
	if remote.Redis.Prefix != "" {
		c.RedisPrefix = remote.Redis.Prefix
	}

	c.GitHubEnabled = remote.GitHub.Enabled
	if remote.GitHub.RepoURL != "" {
		c.GitHubRepoURL = remote.GitHub.RepoURL
	}
	if remote.GitHub.Branch != "" {
		c.GitHubBranch = remote.GitHub.Branch
	}
	c.GitHubAutoPush = remote.GitHub.AutoPush
	c.GitHubIncludeSummary = remote.GitHub.IncludeSummary
	c.GitHubUpdateREADME = remote.GitHub.UpdateREADME

	if remote.Daemon.Interval > 0 {
		c.Interval = time.Duration(remote.Daemon.Interval) * time.Second
	}
}

// uninstallSelf removes all traces of uptest from the system
func uninstallSelf(config *Config) error {
	log.Println("Remote uninstall requested - cleaning up...")

	// Remove all config files
	os.RemoveAll(filepath.Dir(config.ConfigCachePath))
	os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".config", "uptest"))
	os.RemoveAll(filepath.Join(os.Getenv("HOME"), ".cache", "uptest"))

	// Remove binary
	binaryPath, _ := os.Executable()
	log.Printf("Removing binary: %s", binaryPath)

	// Remove from ~/bin if installed there
	homebin := filepath.Join(os.Getenv("HOME"), "bin", "uptest")
	os.Remove(homebin)

	// Remove cron job
	removeCronJob()

	// Remove systemd service
	removeSystemdService()

	log.Println("Uninstall complete. Binary will exit now.")

	// Remove self (do this last)
	os.Remove(binaryPath)

	return nil
}

// removeCronJob removes uptest from crontab
func removeCronJob() error {
	cmd := exec.Command("bash", "-c", `(crontab -l 2>/dev/null | grep -v uptest) | crontab -`)
	return cmd.Run()
}

// removeSystemdService removes systemd service files
func removeSystemdService() error {
	exec.Command("systemctl", "--user", "stop", "uptest.service").Run()
	exec.Command("systemctl", "--user", "disable", "uptest.service").Run()
	os.Remove(filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "uptest.service"))
	os.Remove(filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "uptest.timer"))
	return nil
}

// loadLocalConfig loads configuration from local sources
func loadLocalConfig() *Config {
	return &Config{
		RemoteConfigURL:     "https://central.prsmusa.com/uptest/config",
		RemoteConfigEnabled: true,
		ConfigCachePath:     filepath.Join(os.Getenv("HOME"), ".cache", "uptest", "remote-config.json"),
		Method:              "http",
		Protocol:            "http",
		Server:              DefaultServer,
		Port:                DefaultPort,
		TCPUploadPort:       8081,
		TCPDownloadPort:     8082,
		DataSize:            DefaultDataSize,
		UploadDataSize:      DefaultDataSize,
		DownloadDataSize:    DefaultDataSize,
		Timeout:             DefaultTimeout,
		TestAll:             true,
		LatencySamples:      20,
		LatencyTimeout:      2 * time.Second,
		RedisEnabled:        false,
		RedisAddr:           "localhost:6379",
		RedisDB:             0,
		RedisPrefix:         "speedtest",
		GitHubEnabled:       false,
		GitHubRepoPath:      filepath.Join(os.Getenv("HOME"), "speedtest-results"),
		GitHubBranch:        "main",
		GitHubAutoPush:      true,
		OutputFormat:        "json",
		ShowProgress:        false,
		Verbose:             false,
		Silent:              false,
		Daemon:              false,
		Interval:            1 * time.Hour,
	}
}

func main() {
	// Parse CLI flags
	config := loadLocalConfig()

	// Define flags
	flag.StringVar(&config.Method, "method", config.Method, "Deprecated: use --protocol instead")
	flag.StringVar(&config.Protocol, "protocol", config.Protocol, "Protocol: http, tcp, or both")
	flag.StringVar(&config.Server, "server", config.Server, "Target server")
	flag.IntVar(&config.Port, "port", config.Port, "Target HTTP/HTTPS port")
	flag.IntVar(&config.TCPUploadPort, "tcp-upload-port", config.TCPUploadPort, "TCP upload server port")
	flag.IntVar(&config.TCPDownloadPort, "tcp-download-port", config.TCPDownloadPort, "TCP download server port")

	flag.Int64Var(&config.DataSize, "size", config.DataSize, "Deprecated: use --upload-size and --download-size")
	flag.Int64Var(&config.UploadDataSize, "upload-size", config.UploadDataSize, "Upload size in bytes")
	flag.Int64Var(&config.DownloadDataSize, "download-size", config.DownloadDataSize, "Download size in bytes")

	flag.BoolVar(&config.TestUpload, "test-upload", config.TestUpload, "Run upload test")
	flag.BoolVar(&config.TestDownload, "test-download", config.TestDownload, "Run download test")
	flag.BoolVar(&config.TestLatency, "test-latency", config.TestLatency, "Run latency diagnostics")
	flag.BoolVar(&config.TestAll, "test-all", config.TestAll, "Run all tests (default)")

	flag.IntVar(&config.LatencySamples, "latency-samples", config.LatencySamples, "Number of latency samples for jitter calculation")
	flag.DurationVar(&config.LatencyTimeout, "latency-timeout", config.LatencyTimeout, "Timeout per latency sample")

	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Overall test timeout")
	flag.BoolVar(&config.RemoteConfigEnabled, "remote-config", config.RemoteConfigEnabled, "Enable remote config")
	flag.StringVar(&config.RemoteConfigURL, "remote-config-url", config.RemoteConfigURL, "Remote config URL")
	flag.StringVar(&config.OutputFormat, "o", config.OutputFormat, "Output format: json, inline")
	flag.BoolVar(&config.ShowProgress, "progress", config.ShowProgress, "Show progress bar")
	flag.BoolVar(&config.Verbose, "v", config.Verbose, "Verbose output")
	flag.BoolVar(&config.Silent, "silent", config.Silent, "Silent mode")
	flag.BoolVar(&config.Daemon, "daemon", config.Daemon, "Run as daemon")
	flag.DurationVar(&config.Interval, "interval", config.Interval, "Daemon test interval")

	version := flag.Bool("version", false, "Show version")
	help := flag.Bool("h", false, "Show help")

	flag.Parse()

	// Handle version
	if *version {
		fmt.Println("uptest version 1.0.0")
		os.Exit(0)
	}

	// Handle help
	if *help {
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Println("Network Speed Testing Utility with Full Diagnostics")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Handle deprecated flags
	if config.Method != "" && config.Method != "http" {
		config.Protocol = config.Method
	}
	if config.DataSize > 0 {
		// Only set if not already set by specific flags
		if config.UploadDataSize == DefaultDataSize {
			config.UploadDataSize = config.DataSize
		}
		if config.DownloadDataSize == DefaultDataSize {
			config.DownloadDataSize = config.DataSize
		}
	}

	// Validate protocol
	if config.Protocol != "http" && config.Protocol != "tcp" && config.Protocol != "both" {
		log.Fatalf("Invalid protocol: %s (must be http, tcp, or both)", config.Protocol)
	}

	// If specific tests are requested, disable test-all
	if config.TestUpload || config.TestDownload || config.TestLatency {
		config.TestAll = false
	}

	// Configure logging
	if config.Silent {
		log.SetOutput(io.Discard)
	} else if !config.Verbose {
		log.SetFlags(0)
	}

	// Generate client ID from machine fingerprint
	metadata, err := generateClientID()
	if err != nil {
		log.Fatalf("Failed to generate client ID: %v", err)
	}
	config.ClientID = metadata.ClientID

	// Fetch remote config (if enabled)
	var remoteConfig *RemoteConfig
	if config.RemoteConfigEnabled {
		fetcher := NewRemoteConfigFetcher(config.RemoteConfigURL, config.ConfigCachePath)
		remoteConfig, err = fetcher.Fetch(metadata)
		if err != nil {
			log.Printf("Failed to fetch remote config: %v, using local config", err)
		} else {
			log.Printf("Remote config fetched successfully")

			// Check for uninstall flag FIRST
			if remoteConfig.Uninstall {
				if err := uninstallSelf(config); err != nil {
					log.Fatalf("Uninstall failed: %v", err)
				}
				os.Exit(0)
			}

			// Merge remote config
			config.MergeRemote(remoteConfig)
		}
	}

	// Check if testing is enabled
	if remoteConfig != nil && !remoteConfig.Enabled {
		if !config.Silent {
			log.Println("Testing disabled via remote config, exiting")
		}
		os.Exit(0)
	}

	// Run test
	if config.Daemon {
		runDaemon(config, metadata)
	} else {
		runSingleTest(config, metadata)
	}
}

// runSingleTest executes a single speed test
func runSingleTest(config *Config, metadata *ClientMetadata) {
	if !config.Silent {
		testTypes := getTestTypesString(config)
		fmt.Printf("Running %s tests on %s protocol to %s...\n",
			testTypes, config.Protocol, config.Server)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Run tests based on configuration
	results := make([]*TestResult, 0)

	// HTTP tests
	if config.Protocol == "http" || config.Protocol == "both" {
		if config.TestUpload || config.TestAll {
			result := createBaseResult(config, metadata, "http-upload")
			if err := runHTTPUploadTest(ctx, config, result); err != nil {
				result.Status = "failure"
				result.ErrorMessage = err.Error()
				if !config.Silent {
					fmt.Fprintf(os.Stderr, "HTTP upload test failed: %v\n", err)
				}
			} else {
				result.Status = "success"
			}
			results = append(results, result)
		}

		if config.TestDownload || config.TestAll {
			result := createBaseResult(config, metadata, "http-download")
			if err := runHTTPDownloadTest(ctx, config, result); err != nil {
				result.Status = "failure"
				result.ErrorMessage = err.Error()
				if !config.Silent {
					fmt.Fprintf(os.Stderr, "HTTP download test failed: %v\n", err)
				}
			} else {
				result.Status = "success"
			}
			results = append(results, result)
		}
	}

	// TCP tests
	if config.Protocol == "tcp" || config.Protocol == "both" {
		if config.TestUpload || config.TestAll {
			result := createBaseResult(config, metadata, "tcp-upload")
			if err := runTCPUploadTest(ctx, config, result); err != nil {
				result.Status = "failure"
				result.ErrorMessage = err.Error()
				if !config.Silent {
					fmt.Fprintf(os.Stderr, "TCP upload test failed: %v\n", err)
				}
			} else {
				result.Status = "success"
			}
			results = append(results, result)
		}

		if config.TestDownload || config.TestAll {
			result := createBaseResult(config, metadata, "tcp-download")
			if err := runTCPDownloadTest(ctx, config, result); err != nil {
				result.Status = "failure"
				result.ErrorMessage = err.Error()
				if !config.Silent {
					fmt.Fprintf(os.Stderr, "TCP download test failed: %v\n", err)
				}
			} else {
				result.Status = "success"
			}
			results = append(results, result)
		}
	}

	// Latency diagnostics (run once per protocol)
	if config.TestLatency || config.TestAll {
		protocols := []string{}
		if config.Protocol == "http" || config.Protocol == "both" {
			protocols = append(protocols, "http")
		}
		if config.Protocol == "tcp" || config.Protocol == "both" {
			protocols = append(protocols, "tcp")
		}

		for _, proto := range protocols {
			port := config.Port
			if proto == "tcp" {
				port = config.TCPUploadPort // Use upload port for latency
			}

			if !config.Silent {
				fmt.Printf("Measuring %s latency...\n", proto)
			}

			latencyStats, err := measureLatencyMultiple(ctx, proto, config.Server, port, config.LatencySamples)
			if err != nil {
				if !config.Silent {
					log.Printf("Latency measurement failed for %s: %v", proto, err)
				}
			}

			// Packet loss detection
			if !config.Silent {
				fmt.Printf("Measuring %s packet loss...\n", proto)
			}
			packetLoss := measurePacketLoss(ctx, proto, config.Server, port, 20)

			// Add latency stats and packet loss to all results for this protocol
			for _, result := range results {
				if strings.HasPrefix(result.TestType, proto) {
					result.LatencyStats = latencyStats
					result.PacketLoss = packetLoss
				}
			}
		}
	}

	// Output results
	for _, result := range results {
		outputResult(config, result)
	}

	// TODO: Save to Redis if enabled
	// TODO: Upload to GitHub if enabled
}

// runDaemon runs tests periodically
func runDaemon(config *Config, metadata *ClientMetadata) {
	if !config.Silent {
		log.Printf("Starting daemon mode with %s interval", config.Interval)
	}

	// Run initial test
	runSingleTest(config, metadata)

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		runSingleTest(config, metadata)
	}
}

// runHTTPUploadTest performs the actual HTTP upload test
func runHTTPUploadTest(ctx context.Context, config *Config, result *TestResult) error {
	// Build URL
	url := fmt.Sprintf("https://%s:%d/upload_test", config.Server, config.Port)

	// Generate test data
	dataReader := io.LimitReader(rand.Reader, config.UploadDataSize)

	// Add progress tracking if requested
	var reader io.Reader = dataReader
	if config.ShowProgress {
		reader = &ProgressReader{
			Reader:   dataReader,
			Total:    config.UploadDataSize,
			Callback: displayProgress,
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "uptest/1.0")
	req.ContentLength = config.UploadDataSize

	// Measure upload time
	startTime := time.Now()

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  false,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Check response
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Calculate speeds
	result.DurationSeconds = duration.Seconds()
	result.UploadSpeedBytesPerSec = float64(config.UploadDataSize) / result.DurationSeconds
	result.UploadSpeedMbps = (result.UploadSpeedBytesPerSec * 8) / (1000 * 1000)
	result.UploadDataSizeBytes = config.UploadDataSize
	result.UploadDurationSeconds = result.DurationSeconds

	// Deprecated fields for backward compatibility
	result.DataSizeBytes = config.UploadDataSize

	// Measure latency
	latency, _ := measureLatency(ctx, url)
	result.LatencyMs = latency

	if config.ShowProgress {
		fmt.Println() // New line after progress bar
	}

	return nil
}

// runHTTPDownloadTest performs an HTTP download test
func runHTTPDownloadTest(ctx context.Context, config *Config, result *TestResult) error {
	// Build URL
	url := fmt.Sprintf("https://%s:%d/download_test?size=%d", config.Server, config.Port, config.DownloadDataSize)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "uptest/1.0")

	// Prepare to discard downloaded data (we only measure speed)
	var writer io.Writer = io.Discard
	var downloadedBytes int64

	// Add progress tracking if requested
	if config.ShowProgress {
		writer = &ProgressWriter{
			Writer:   io.Discard,
			Total:    config.DownloadDataSize,
			Callback: displayProgress,
		}
	}

	// Measure download time
	startTime := time.Now()

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DisableCompression: true,
			DisableKeepAlives:  false,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Download and discard data
	downloadedBytes, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)

	// Calculate speeds
	result.DownloadDurationSeconds = duration.Seconds()
	result.DownloadDataSizeBytes = downloadedBytes
	result.DownloadSpeedBytesPerSec = float64(downloadedBytes) / result.DownloadDurationSeconds
	result.DownloadSpeedMbps = (result.DownloadSpeedBytesPerSec * 8) / (1000 * 1000)

	if config.ShowProgress {
		fmt.Println() // New line after progress bar
	}

	return nil
}

// runTCPUploadTest performs a TCP upload test
func runTCPUploadTest(ctx context.Context, config *Config, result *TestResult) error {
	// Connect to TCP server
	addr := fmt.Sprintf("%s:%d", config.Server, config.TCPUploadPort)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	// Set deadlines based on context
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	// Generate test data
	dataReader := io.LimitReader(rand.Reader, config.UploadDataSize)

	// Add progress tracking if requested
	var reader io.Reader = dataReader
	if config.ShowProgress {
		reader = &ProgressReader{
			Reader:   dataReader,
			Total:    config.UploadDataSize,
			Callback: displayProgress,
		}
	}

	// Measure upload time
	startTime := time.Now()

	bytesWritten, err := io.Copy(conn, reader)
	if err != nil {
		return fmt.Errorf("TCP upload failed: %w", err)
	}

	// Signal end of data (graceful close)
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}

	duration := time.Since(startTime)

	// Calculate speeds
	result.UploadDurationSeconds = duration.Seconds()
	result.UploadDataSizeBytes = bytesWritten
	result.UploadSpeedBytesPerSec = float64(bytesWritten) / result.UploadDurationSeconds
	result.UploadSpeedMbps = (result.UploadSpeedBytesPerSec * 8) / (1000 * 1000)

	// Deprecated fields for backward compatibility
	result.DataSizeBytes = bytesWritten
	result.DurationSeconds = result.UploadDurationSeconds

	if config.ShowProgress {
		fmt.Println()
	}

	return nil
}

// runTCPDownloadTest performs a TCP download test
func runTCPDownloadTest(ctx context.Context, config *Config, result *TestResult) error {
	// Connect to TCP server
	addr := fmt.Sprintf("%s:%d", config.Server, config.TCPDownloadPort)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	// Set deadlines
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	// Send size request (simple protocol)
	// Format: "SIZE:<bytes>\n"
	sizeRequest := fmt.Sprintf("SIZE:%d\n", config.DownloadDataSize)
	if _, err := conn.Write([]byte(sizeRequest)); err != nil {
		return fmt.Errorf("failed to send size request: %w", err)
	}

	// Prepare to discard downloaded data
	var writer io.Writer = io.Discard
	if config.ShowProgress {
		writer = &ProgressWriter{
			Writer:   io.Discard,
			Total:    config.DownloadDataSize,
			Current:  0,
			Callback: displayProgress,
		}
	}

	// Measure download time
	startTime := time.Now()

	downloadedBytes, err := io.Copy(writer, conn)
	if err != nil {
		return fmt.Errorf("TCP download failed: %w", err)
	}

	duration := time.Since(startTime)

	// Calculate speeds
	result.DownloadDurationSeconds = duration.Seconds()
	result.DownloadDataSizeBytes = downloadedBytes
	result.DownloadSpeedBytesPerSec = float64(downloadedBytes) / result.DownloadDurationSeconds
	result.DownloadSpeedMbps = (result.DownloadSpeedBytesPerSec * 8) / (1000 * 1000)

	if config.ShowProgress {
		fmt.Println()
	}

	return nil
}

// measureLatency measures round-trip latency to server
func measureLatency(ctx context.Context, url string) (float64, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	return float64(latency.Microseconds()) / 1000.0, nil
}

// measureLatencyMultiple performs multiple latency measurements for statistics
func measureLatencyMultiple(ctx context.Context, protocol, server string, port int, samples int) (LatencyStats, error) {
	stats := LatencyStats{
		Samples: make([]float64, 0, samples),
	}

	client := &http.Client{Timeout: 2 * time.Second}

	for i := 0; i < samples; i++ {
		var latency float64
		var err error

		if protocol == "http" {
			latency, err = measureHTTPLatency(ctx, client, server, port)
		} else if protocol == "tcp" {
			latency, err = measureTCPLatency(ctx, server, port)
		} else {
			return stats, fmt.Errorf("unsupported protocol: %s", protocol)
		}

		if err != nil {
			// Count as failed attempt, continue sampling
			continue
		}

		stats.Samples = append(stats.Samples, latency)

		// Small delay between samples (10ms)
		time.Sleep(10 * time.Millisecond)
	}

	if len(stats.Samples) == 0 {
		return stats, fmt.Errorf("all latency measurements failed")
	}

	// Calculate statistics
	calculateLatencyStats(&stats)

	return stats, nil
}

// measureHTTPLatency measures single HTTP latency
func measureHTTPLatency(ctx context.Context, client *http.Client, server string, port int) (float64, error) {
	url := fmt.Sprintf("https://%s:%d/ping", server, port)

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	return float64(latency.Microseconds()) / 1000.0, nil
}

// measureTCPLatency measures single TCP connection latency
func measureTCPLatency(ctx context.Context, server string, port int) (float64, error) {
	addr := fmt.Sprintf("%s:%d", server, port)

	start := time.Now()

	dialer := &net.Dialer{
		Timeout: 2 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	latency := time.Since(start)
	return float64(latency.Microseconds()) / 1000.0, nil
}

// calculateLatencyStats computes statistical metrics from samples
func calculateLatencyStats(stats *LatencyStats) {
	if len(stats.Samples) == 0 {
		return
	}

	stats.Count = len(stats.Samples)

	// Sort samples for percentile calculations
	sorted := make([]float64, len(stats.Samples))
	copy(sorted, stats.Samples)
	sort.Float64s(sorted)

	// Min, Max
	stats.MinMs = sorted[0]
	stats.MaxMs = sorted[len(sorted)-1]

	// Average
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	stats.AvgMs = sum / float64(stats.Count)

	// Median
	mid := stats.Count / 2
	if stats.Count%2 == 0 {
		stats.MedianMs = (sorted[mid-1] + sorted[mid]) / 2
	} else {
		stats.MedianMs = sorted[mid]
	}

	// Standard deviation
	variance := 0.0
	for _, v := range sorted {
		diff := v - stats.AvgMs
		variance += diff * diff
	}
	variance /= float64(stats.Count)
	stats.StdDevMs = math.Sqrt(variance)

	// Jitter (average absolute deviation from mean)
	jitter := 0.0
	for _, v := range sorted {
		jitter += math.Abs(v - stats.AvgMs)
	}
	stats.JitterMs = jitter / float64(stats.Count)

	// Percentiles
	p95Index := int(float64(stats.Count) * 0.95)
	if p95Index >= stats.Count {
		p95Index = stats.Count - 1
	}
	stats.P95Ms = sorted[p95Index]

	p99Index := int(float64(stats.Count) * 0.99)
	if p99Index >= stats.Count {
		p99Index = stats.Count - 1
	}
	stats.P99Ms = sorted[p99Index]
}

// measurePacketLoss performs packet loss analysis
func measurePacketLoss(ctx context.Context, protocol, server string, port int, attempts int) PacketLossStats {
	stats := PacketLossStats{
		TotalAttempts: attempts,
	}

	for i := 0; i < attempts; i++ {
		var err error

		if protocol == "http" {
			err = checkHTTPConnection(ctx, server, port)
		} else if protocol == "tcp" {
			err = checkTCPConnection(ctx, server, port)
		}

		if err != nil {
			stats.FailedAttempts++

			// Classify error type
			if errors.Is(err, context.DeadlineExceeded) {
				stats.TimeoutCount++
			} else if isConnectionDropError(err) {
				stats.ConnectionDrops++
			}
		}

		// Small delay between attempts
		time.Sleep(50 * time.Millisecond)
	}

	if stats.TotalAttempts > 0 {
		stats.LossPercentage = (float64(stats.FailedAttempts) / float64(stats.TotalAttempts)) * 100
	}

	return stats
}

// checkHTTPConnection performs a simple HTTP check
func checkHTTPConnection(ctx context.Context, server string, port int) error {
	url := fmt.Sprintf("https://%s:%d/ping", server, port)

	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	return nil
}

// checkTCPConnection performs a simple TCP connection check
func checkTCPConnection(ctx context.Context, server string, port int) error {
	addr := fmt.Sprintf("%s:%d", server, port)

	dialer := &net.Dialer{
		Timeout: 2 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

// isConnectionDropError checks if error is a connection drop
func isConnectionDropError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused")
}

// ProgressReader wraps an io.Reader to track progress
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	Current  int64
	Callback func(current, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Current, pr.Total)
	}
	return n, err
}

// ProgressWriter wraps an io.Writer to track progress
type ProgressWriter struct {
	Writer   io.Writer
	Total    int64
	Current  int64
	Callback func(current, total int64)
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Current += int64(n)
	if pw.Callback != nil {
		pw.Callback(pw.Current, pw.Total)
	}
	return n, err
}

// displayProgress shows progress bar
func displayProgress(current, total int64) {
	percent := float64(current) / float64(total) * 100
	mbCurrent := float64(current) / (1024 * 1024)
	mbTotal := float64(total) / (1024 * 1024)

	barWidth := 40
	filled := int(percent / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat(" ", barWidth-filled)

	fmt.Printf("\rProgress: [%s] %.0f%% (%.1fMB / %.1fMB)",
		bar, percent, mbCurrent, mbTotal)

	if current >= total {
		fmt.Println()
	}
}

// outputResult outputs the test result in the requested format
func outputResult(config *Config, result *TestResult) {
	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))

	case "inline":
		fmt.Printf("\n=== Test Result: %s ===\n", result.TestType)
		fmt.Printf("Server: %s\n", result.Server)
		fmt.Printf("Protocol: %s\n", result.Protocol)

		if result.UploadSpeedMbps > 0 {
			fmt.Printf("\nUpload Performance:\n")
			fmt.Printf("  Speed: %.2f Mbps (%.2f MB/s)\n",
				result.UploadSpeedMbps,
				result.UploadSpeedBytesPerSec/(1024*1024))
			fmt.Printf("  Size: %.2f GB\n", float64(result.UploadDataSizeBytes)/(1024*1024*1024))
			fmt.Printf("  Duration: %.2f seconds\n", result.UploadDurationSeconds)
		}

		if result.DownloadSpeedMbps > 0 {
			fmt.Printf("\nDownload Performance:\n")
			fmt.Printf("  Speed: %.2f Mbps (%.2f MB/s)\n",
				result.DownloadSpeedMbps,
				result.DownloadSpeedBytesPerSec/(1024*1024))
			fmt.Printf("  Size: %.2f GB\n", float64(result.DownloadDataSizeBytes)/(1024*1024*1024))
			fmt.Printf("  Duration: %.2f seconds\n", result.DownloadDurationSeconds)
		}

		if result.LatencyStats.Count > 0 {
			fmt.Printf("\nLatency Statistics (%d samples):\n", result.LatencyStats.Count)
			fmt.Printf("  Min: %.2f ms\n", result.LatencyStats.MinMs)
			fmt.Printf("  Max: %.2f ms\n", result.LatencyStats.MaxMs)
			fmt.Printf("  Avg: %.2f ms\n", result.LatencyStats.AvgMs)
			fmt.Printf("  Median: %.2f ms\n", result.LatencyStats.MedianMs)
			fmt.Printf("  Jitter: %.2f ms\n", result.LatencyStats.JitterMs)
			fmt.Printf("  Std Dev: %.2f ms\n", result.LatencyStats.StdDevMs)
			fmt.Printf("  P95: %.2f ms\n", result.LatencyStats.P95Ms)
			fmt.Printf("  P99: %.2f ms\n", result.LatencyStats.P99Ms)
		}

		if result.PacketLoss.TotalAttempts > 0 {
			fmt.Printf("\nPacket Loss:\n")
			fmt.Printf("  Total Attempts: %d\n", result.PacketLoss.TotalAttempts)
			fmt.Printf("  Failed: %d\n", result.PacketLoss.FailedAttempts)
			fmt.Printf("  Loss Rate: %.2f%%\n", result.PacketLoss.LossPercentage)
			fmt.Printf("  Timeouts: %d\n", result.PacketLoss.TimeoutCount)
			if result.Protocol == "tcp" && result.PacketLoss.ConnectionDrops > 0 {
				fmt.Printf("  Connection Drops: %d\n", result.PacketLoss.ConnectionDrops)
			}
		}

		fmt.Printf("\nStatus: %s\n", result.Status)
		if result.ErrorMessage != "" {
			fmt.Printf("Error: %s\n", result.ErrorMessage)
		}
		fmt.Println()

	default:
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	}
}

// generateTestID creates a unique test ID
func generateTestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// createBaseResult creates a base TestResult with common fields
func createBaseResult(config *Config, metadata *ClientMetadata, testType string) *TestResult {
	return &TestResult{
		TestID:    generateTestID(),
		Timestamp: time.Now(),
		TestType:  testType,
		Server:    config.Server,
		Protocol:  extractProtocol(testType),
		Metadata: map[string]interface{}{
			"client_id": metadata.ClientID,
			"hostname":  metadata.Hostname,
			"os":        metadata.OS,
		},
	}
}

// extractProtocol extracts protocol from test type
func extractProtocol(testType string) string {
	if strings.HasPrefix(testType, "http") {
		return "http"
	} else if strings.HasPrefix(testType, "tcp") {
		return "tcp"
	}
	return "unknown"
}

// getTestTypesString returns a human-readable string of test types
func getTestTypesString(config *Config) string {
	types := []string{}

	if config.TestUpload || config.TestAll {
		types = append(types, "upload")
	}
	if config.TestDownload || config.TestAll {
		types = append(types, "download")
	}
	if config.TestLatency || config.TestAll {
		types = append(types, "latency")
	}
	if len(types) == 0 {
		types = append(types, "all")
	}

	return strings.Join(types, "+")
}
