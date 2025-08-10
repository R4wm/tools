#!/bin/bash

# Script to follow redirects and show final destination URL
# Usage: ./get-final-url.sh <URL> [-v|--verbose]

if [ $# -eq 0 ]; then
    echo "Usage: $0 <URL> [-v|--verbose]"
    echo "Example: $0 https://bit.ly/example"
    echo "Example: $0 -v https://bit.ly/example"
    exit 1
fi

# Parse arguments
URL=""
VERBOSE=false

for arg in "$@"; do
    case $arg in
        -v|--verbose)
            VERBOSE=true
            ;;
        http*|https*)
            URL="$arg"
            ;;
        *)
            if [ -z "$URL" ]; then
                URL="$arg"
            fi
            ;;
    esac
done

if [ -z "$URL" ]; then
    echo "Error: No URL provided"
    echo "Usage: $0 <URL> [-v|--verbose]"
    exit 1
fi

echo "Following redirects for: $URL"
echo

# Use curl to follow redirects and get final URL
FINAL_URL=$(curl -sLI -o /dev/null -w '%{url_effective}' "$URL" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$FINAL_URL" ]; then
    echo "Final destination URL: $FINAL_URL"
    
    # Show all redirect steps if verbose mode
    if [ "$VERBOSE" = true ]; then
        echo
        echo "Redirect chain:"
        curl -sLI "$URL" | grep -i "^location:" | nl
    fi
else
    echo "Error: Could not follow redirects for $URL"
    exit 1
fi