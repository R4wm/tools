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
	"regexp"
	"strings"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Yellow    = "\033[33m"
	BoldYellow = "\033[1;33m"
)

// Verse represents a Bible verse from the API response
type Verse struct {
	Book    string `json:"Book"`
	Chapter int    `json:"Chapter"`
	Text    string `json:"Text"`
	Verse   int    `json:"Verse"`
}

// APIResponse represents the full API response structure
type APIResponse struct {
	Count       map[string]int `json:"Count"`
	GraphCount  interface{}    `json:"GraphCount"`
	SearchString string        `json:"SearchString"`
	Verses      []Verse        `json:"Verses"`
}

// highlightText highlights search terms in text using ANSI color codes
func highlightText(text, searchTerm string) string {
	// Create case-insensitive regex pattern
	pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(searchTerm))
	
	// Replace matches with highlighted version
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		return BoldYellow + match + Reset
	})
}

func main() {
	// Parse command line flags
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")
	color := flag.Bool("c", false, "highlight search terms in color")
	flag.Parse()

	// Configure logging
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Show help if requested
	if *help {
		fmt.Printf("Usage: %s [-v] [-h] [-c] <query-string...>\n", os.Args[0])
		fmt.Println("  -v    verbose output")
		fmt.Println("  -h    show this help")
		fmt.Println("  -c    highlight search terms in color")
		os.Exit(0)
	}

	// Check if arguments are supplied
	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Usage: %s [-v] [-h] [-c] <query-string...>\n", os.Args[0])
		os.Exit(1)
	}

	// Join all arguments into a single string
	query := strings.Join(args, " ")
	log.Printf("Search query: %s", query)

	// URL-encode the query string
	encodedQuery := url.QueryEscape(query)
	log.Printf("URL-encoded query: %s", encodedQuery)

	// Construct the URL
	apiURL := fmt.Sprintf("https://prsmusa.com/bible/search?q=%s&json=true", encodedQuery)
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

	// Parse JSON response into structured data
	log.Println("Parsing JSON response...")
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("JSON parsing failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Apply highlighting to verse text and output
	log.Println("Processing and highlighting results...")
	for _, verse := range apiResponse.Verses {
		// Conditionally highlight the search term in the verse text
		text := verse.Text
		if *color {
			text = highlightText(text, query)
		}
		
		// Format and print the verse
		fmt.Printf("%s %d:%d - %s\n", 
			verse.Book, 
			verse.Chapter, 
			verse.Verse, 
			text)
	}
}
