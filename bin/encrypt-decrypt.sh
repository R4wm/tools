#!/bin/bash

# Function to display usage
usage() {
    echo "Usage: $0 [-e|-d] <input_file> [output_file]"
    echo "  -e  Encrypt file"
    echo "  -d  Decrypt file"
    echo ""
    echo "Environment variable:"
    echo "  openssl_encrypt_password - Password for encryption/decryption"
    exit 1
}

# Check if at least 2 arguments are provided
if [ $# -lt 2 ]; then
    usage
fi

# Parse arguments
MODE=$1
INPUT_FILE=$2
OUTPUT_FILE=$3

# Validate mode
if [ "$MODE" != "-e" ] && [ "$MODE" != "-d" ]; then
    echo "Error: Invalid mode. Use -e for encrypt or -d for decrypt."
    usage
fi

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file '$INPUT_FILE' does not exist."
    exit 1
fi

# Set default output file if not provided
if [ -z "$OUTPUT_FILE" ]; then
    if [ "$MODE" = "-e" ]; then
        OUTPUT_FILE="${INPUT_FILE}.enc"
    else
        OUTPUT_FILE="${INPUT_FILE%.enc}"
        # If file doesn't end with .enc, append .dec
        if [ "$OUTPUT_FILE" = "$INPUT_FILE" ]; then
            OUTPUT_FILE="${INPUT_FILE}.dec"
        fi
    fi
fi

# Get password
if [ -z "$openssl_encrypt_password" ]; then
    echo -n "Enter password: "
    read -s PASSWORD
    echo
    echo -n "Confirm password: "
    read -s PASSWORD_CONFIRM
    echo
    
    if [ "$PASSWORD" != "$PASSWORD_CONFIRM" ]; then
        echo "Error: Passwords do not match."
        exit 1
    fi
else
    PASSWORD="$openssl_encrypt_password"
fi

# Perform encryption or decryption
if [ "$MODE" = "-e" ]; then
    # Encrypt
    openssl enc -aes-256-cbc -salt -pbkdf2 -in "$INPUT_FILE" -out "$OUTPUT_FILE" -pass pass:"$PASSWORD"
    if [ $? -eq 0 ]; then
        echo "File encrypted successfully: $OUTPUT_FILE"
    else
        echo "Error: Encryption failed."
        exit 1
    fi
else
    # Decrypt
    openssl enc -aes-256-cbc -d -pbkdf2 -in "$INPUT_FILE" -out "$OUTPUT_FILE" -pass pass:"$PASSWORD"
    if [ $? -eq 0 ]; then
        echo "File decrypted successfully: $OUTPUT_FILE"
    else
        echo "Error: Decryption failed. Wrong password or corrupted file."
        exit 1
    fi
fi