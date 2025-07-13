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

// generateGraph creates an ASCII bar graph from book counts
func generateGraph(counts map[string]int, verses []Verse, searchPhrase string) string {
	if len(counts) == 0 {
		return "No results to graph"
	}
	
	// Find max count for scaling
	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}
	
	// Build the graph
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Count per Book for '%s':\n", searchPhrase))
	
	// Get book order from verses (maintains biblical order)
	var bookOrder []string
	seenBooks := make(map[string]bool)
	
	for _, verse := range verses {
		if !seenBooks[verse.Book] {
			bookOrder = append(bookOrder, verse.Book)
			seenBooks[verse.Book] = true
		}
	}
	
	// Display books in biblical order
	var overallCount int
	var hasOverall bool
	
	for _, book := range bookOrder {
		if count, exists := counts[book]; exists {
			// Scale bar length (max 50 characters)
			barLength := (count * 50) / maxCount
			if barLength == 0 && count > 0 {
				barLength = 1 // Ensure at least 1 character for non-zero counts
			}
			
			// Create the bar
			bar := strings.Repeat("█", barLength)
			result.WriteString(fmt.Sprintf("%-15s %3d %s\n", book, count, bar))
		}
	}
	
	// Check for "overall" in the counts
	for book, count := range counts {
		if strings.ToLower(book) == "overall" {
			overallCount = count
			hasOverall = true
			break
		}
	}
	
	// Append "overall" at the end if it exists
	if hasOverall {
		// Scale bar length (max 50 characters)
		barLength := (overallCount * 50) / maxCount
		if barLength == 0 && overallCount > 0 {
			barLength = 1 // Ensure at least 1 character for non-zero counts
		}
		
		// Create the bar
		bar := strings.Repeat("█", barLength)
		result.WriteString(fmt.Sprintf("%-15s %3d %s\n", "overall", overallCount, bar))
	}
	
	return result.String()
}

func main() {
	// Parse command line flags
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")
	output := flag.String("o", "json", "output format: json, inline, graph")
	graph := flag.Bool("g", false, "display ASCII graph of count per book")
	
	// Parse all arguments manually to handle flags anywhere
	var queryArgs []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-v" {
			*verbose = true
		} else if arg == "-h" {
			*help = true
		} else if arg == "-g" {
			*graph = true
		} else if arg == "-o" && i+1 < len(os.Args) {
			*output = os.Args[i+1]
			i++ // Skip the next argument as it's the value for -o
		} else if !strings.HasPrefix(arg, "-") {
			queryArgs = append(queryArgs, arg)
		}
	}

	// Configure logging
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Show help if requested
	if *help {
		fmt.Printf("Usage: %s [-v] [-h] [-o format] [-g] <query-string...>\n", os.Args[0])
		fmt.Println("  -v         verbose output")
		fmt.Println("  -h         show this help")
		fmt.Println("  -o format  output format: json (default), inline, graph")
		fmt.Println("  -g         display ASCII graph of count per book")
		os.Exit(0)
	}

	// Check if arguments are supplied
	if len(queryArgs) == 0 {
		fmt.Printf("Usage: %s [-v] [-h] [-o format] [-g] <query-string...>\n", os.Args[0])
		os.Exit(1)
	}

	// Join all arguments into a single string
	query := strings.Join(queryArgs, " ")
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

	// Output based on format (default: json)
	switch *output {
	case "json":
		jsonOutput, err := json.MarshalIndent(apiResponse, "", "  ")
		if err != nil {
			log.Printf("JSON marshaling failed: %v", err)
			fmt.Fprintf(os.Stderr, "Error creating JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	case "inline":
		for _, verse := range apiResponse.Verses {
			fmt.Printf("%s %d:%d - %s\n", verse.Book, verse.Chapter, verse.Verse, verse.Text)
		}
	case "graph":
		fmt.Print(generateGraph(apiResponse.Count, apiResponse.Verses, query))
	default:
		fmt.Fprintf(os.Stderr, "Unknown output format: %s\n", *output)
		os.Exit(1)
	}
	
	// If graph flag is set, also show the graph (in addition to regular output)
	if *graph && *output != "graph" {
		fmt.Print(generateGraph(apiResponse.Count, apiResponse.Verses, query))
	}
}
