// cloudflare-ddns: keep a Cloudflare DNS A record pointed at this machine's
// current public IP, even when behind a router/NAT.
//
// Public IP is discovered without scraping a single point of failure: it first
// asks OpenDNS over plain DNS (the same trick `dig myip.opendns.com
// @resolver1.opendns.com` uses), then falls back to a couple of HTTPS echo
// services. Cloudflare's own record is treated as the source of truth, so the
// daemon self-heals if the record drifts.
//
// Build:   make cloudflare-ddns      (or: go build -o bin/cloudflare-ddns src/cloudflare_ddns.go)
// Run one shot (cron/timer):  cloudflare-ddns -once
// Run as a daemon:            cloudflare-ddns -interval 5m
//
// Configuration comes from env vars, or an env-style file passed with -config:
//
//	CF_API_TOKEN   Cloudflare API token, scoped Zone:DNS:Edit + Zone:Read  (required)
//	CF_DOMAIN      Zone / apex domain, e.g. example.com                    (required)
//	CF_RECORD      Record name to update, e.g. example.com or home.example.com
//	               (defaults to CF_DOMAIN)
//	CF_TTL         Record TTL in seconds; 1 = "automatic"  (default 300)
//	CF_PROXIED     true to proxy through Cloudflare, false for DNS-only    (default false)
//
// For a home server where you port-forward arbitrary ports, keep CF_PROXIED=false
// (DNS-only) so traffic hits your IP directly instead of Cloudflare's proxy.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const cfAPI = "https://api.cloudflare.com/client/v4"

type config struct {
	Token   string
	Domain  string
	Record  string
	TTL     int
	Proxied bool
}

func main() {
	var (
		configPath string
		once       bool
		interval   time.Duration
		verbose    bool
	)
	flag.StringVar(&configPath, "config", "", "path to env-style config file (optional; env vars also work)")
	flag.BoolVar(&once, "once", false, "run a single update and exit (for cron / systemd timers)")
	flag.DurationVar(&interval, "interval", 5*time.Minute, "daemon poll interval when not using -once")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.Parse()

	log.SetFlags(log.LstdFlags)

	if configPath != "" {
		if err := loadEnvFile(configPath); err != nil {
			log.Fatalf("config: %v", err)
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	vlog := func(format string, a ...any) {
		if verbose {
			log.Printf(format, a...)
		}
	}

	c := &client{http: &http.Client{Timeout: 15 * time.Second}, token: cfg.Token}

	// Resolve the zone once up front; it doesn't change while we run.
	zoneID, err := c.zoneID(cfg.Domain)
	if err != nil {
		log.Fatalf("lookup zone %q: %v", cfg.Domain, err)
	}
	vlog("zone %s -> %s", cfg.Domain, zoneID)

	runOnce := func() {
		ip, src, err := publicIP()
		if err != nil {
			log.Printf("ERROR: detect public IP: %v", err)
			return
		}
		vlog("public IP %s (via %s)", ip, src)

		if err := c.sync(zoneID, cfg, ip, vlog); err != nil {
			log.Printf("ERROR: sync %s: %v", cfg.Record, err)
		}
	}

	if once {
		runOnce()
		return
	}

	log.Printf("cloudflare-ddns started: %s -> (public IP), every %s", cfg.Record, interval)
	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		runOnce()
	}
}

// loadConfig reads settings from the environment and validates them.
func loadConfig() (config, error) {
	cfg := config{
		Token:  strings.TrimSpace(os.Getenv("CF_API_TOKEN")),
		Domain: strings.TrimSpace(os.Getenv("CF_DOMAIN")),
		Record: strings.TrimSpace(os.Getenv("CF_RECORD")),
		TTL:    300,
	}
	if cfg.Token == "" {
		return cfg, fmt.Errorf("CF_API_TOKEN is required")
	}
	if cfg.Domain == "" {
		return cfg, fmt.Errorf("CF_DOMAIN is required")
	}
	if cfg.Record == "" {
		cfg.Record = cfg.Domain
	}
	if v := strings.TrimSpace(os.Getenv("CF_TTL")); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &cfg.TTL); err != nil {
			return cfg, fmt.Errorf("CF_TTL %q is not a number", v)
		}
	}
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("CF_PROXIED"))); v == "true" || v == "1" || v == "yes" {
		cfg.Proxied = true
	}
	return cfg, nil
}

// loadEnvFile loads KEY=VALUE lines into the process environment. Existing env
// vars win, so you can override the file on the command line. Supports comments,
// blank lines, optional "export ", and quoted values.
func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return s.Err()
}

// ---- public IP detection ----------------------------------------------------

// publicIP returns this machine's public IPv4, working from behind NAT. It tries
// DNS (OpenDNS) first, then HTTPS echo services, returning the source for logs.
func publicIP() (ip, source string, err error) {
	if ip, err := ipViaOpenDNS(); err == nil {
		return ip, "opendns", nil
	}
	for _, u := range []string{
		"https://1.1.1.1/cdn-cgi/trace", // Cloudflare; parse "ip=" line
		"https://api.ipify.org",
		"https://icanhazip.com",
	} {
		if ip, err := ipViaHTTP(u); err == nil {
			return ip, u, nil
		}
	}
	return "", "", fmt.Errorf("all public-IP methods failed")
}

// ipViaOpenDNS resolves the special name myip.opendns.com against OpenDNS's
// resolvers, which answer with the querying client's public IP.
func ipViaOpenDNS() (string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Force IPv4 transport so OpenDNS sees our v4 source and answers
			// with the v4 address we want for an A record.
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp4", "resolver1.opendns.com:53")
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	addrs, err := r.LookupHost(ctx, "myip.opendns.com")
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		if v4 := net.ParseIP(a).To4(); v4 != nil {
			return v4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 in OpenDNS answer")
}

// ipv4HTTP is an HTTP client pinned to IPv4 so echo services report our public
// IPv4 even on dual-stack networks.
var ipv4HTTP = &http.Client{
	Timeout: 8 * time.Second,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 8 * time.Second}
			return d.DialContext(ctx, "tcp4", addr)
		},
	},
}

// ipViaHTTP fetches a public-IP echo endpoint and extracts the first IPv4 it finds.
func ipViaHTTP(url string) (string, error) {
	resp, err := ipv4HTTP.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	text := string(body)
	// Cloudflare's trace endpoint returns key=value lines.
	if strings.Contains(text, "ip=") {
		for _, line := range strings.Split(text, "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "ip="); ok {
				text = rest
				break
			}
		}
	}
	ipStr := strings.TrimSpace(text)
	if v4 := net.ParseIP(ipStr).To4(); v4 != nil {
		return v4.String(), nil
	}
	return "", fmt.Errorf("%s: no IPv4 in response", url)
}

// ---- Cloudflare API ---------------------------------------------------------

type client struct {
	http  *http.Client
	token string
}

type cfResponse struct {
	Success bool            `json:"success"`
	Errors  []cfMessage     `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type cfMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type dnsRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

func (c *client) do(method, url string, body any) (*cfResponse, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var out cfResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if !out.Success {
		return &out, fmt.Errorf("http %d: %s", resp.StatusCode, cfErr(out.Errors))
	}
	return &out, nil
}

func cfErr(msgs []cfMessage) string {
	if len(msgs) == 0 {
		return "unknown error"
	}
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		parts = append(parts, fmt.Sprintf("[%d] %s", m.Code, m.Message))
	}
	return strings.Join(parts, "; ")
}

func (c *client) zoneID(domain string) (string, error) {
	resp, err := c.do(http.MethodGet, cfAPI+"/zones?name="+domain, nil)
	if err != nil {
		return "", err
	}
	var zones []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Result, &zones); err != nil {
		return "", err
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("no zone named %q on this account (is the domain added to Cloudflare?)", domain)
	}
	return zones[0].ID, nil
}

func (c *client) findRecord(zoneID, name string) (*dnsRecord, error) {
	resp, err := c.do(http.MethodGet, fmt.Sprintf("%s/zones/%s/dns_records?type=A&name=%s", cfAPI, zoneID, name), nil)
	if err != nil {
		return nil, err
	}
	var recs []dnsRecord
	if err := json.Unmarshal(resp.Result, &recs); err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return &recs[0], nil
}

// sync makes the A record match ip, creating it if absent and skipping the API
// call when nothing changed.
func (c *client) sync(zoneID string, cfg config, ip string, vlog func(string, ...any)) error {
	rec, err := c.findRecord(zoneID, cfg.Record)
	if err != nil {
		return err
	}

	payload := dnsRecord{
		Type:    "A",
		Name:    cfg.Record,
		Content: ip,
		TTL:     cfg.TTL,
		Proxied: cfg.Proxied,
	}

	if rec == nil {
		_, err := c.do(http.MethodPost, fmt.Sprintf("%s/zones/%s/dns_records", cfAPI, zoneID), payload)
		if err != nil {
			return err
		}
		log.Printf("created A %s -> %s", cfg.Record, ip)
		return nil
	}

	if rec.Content == ip && rec.Proxied == cfg.Proxied && rec.TTL == cfg.TTL {
		vlog("no change: %s already %s", cfg.Record, ip)
		return nil
	}

	_, err = c.do(http.MethodPut, fmt.Sprintf("%s/zones/%s/dns_records/%s", cfAPI, zoneID, rec.ID), payload)
	if err != nil {
		return err
	}
	log.Printf("updated A %s: %s -> %s", cfg.Record, rec.Content, ip)
	return nil
}
