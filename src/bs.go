package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	// Parse command line flags
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")
	flag.Parse()

	// Configure logging
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Show help if requested
	if *help {
		fmt.Printf("Usage: %s [-v] [-h] <query-string...>\n", os.Args[0])
		fmt.Println("  -v    verbose output")
		fmt.Println("  -h    show this help")
		os.Exit(0)
	}

	// Check if arguments are supplied
	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Usage: %s [-v] [-h] <query-string...>\n", os.Args[0])
		os.Exit(1)
	}

	// Join all arguments into a single string
	query := strings.Join(args, " ")
	log.Printf("Search query: %s", query)

	// URL-encode the query string
	encodedQuery := url.QueryEscape(query)
	log.Printf("URL-encoded query: %s", encodedQuery)

	// Construct the URL
	apiURL := fmt.Sprintf("https://web.prsmusa.com/bible/search?q=%s&json=true", encodedQuery)
	log.Printf("API URL: %s", apiURL)

	if *verbose {
		fmt.Println(apiURL)
	}

	// Make HTTP request
	log.Println("Making HTTP request...")
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	log.Printf("HTTP response status: %s", resp.Status)

	// Read response body
	log.Println("Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Response body length: %d bytes", len(body))

	// Pretty print JSON (equivalent to jq .)
	log.Println("Parsing JSON response...")
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		log.Printf("JSON parsing failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	log.Println("Formatting JSON output...")
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Printf("JSON formatting failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		os.Exit(1)
	}

	log.Println("Outputting formatted JSON")
	fmt.Println(string(prettyJSON))
}
