#!/bin/bash

# Script to switch audio source with user selection
# Usage: ./switch-audio-source.sh

echo "Current audio source:"
echo "$(pactl get-default-source)"
echo

echo "Available audio sources:"
pactl list short sources | nl -v0

echo
read -p "Enter the number of the source you want to switch to: " choice

# Get the selected source
SOURCE=$(pactl list short sources | sed -n "$((choice+1))p" | cut -f2)

if [ -z "$SOURCE" ]; then
    echo "Invalid selection."
    exit 1
fi

echo "Switching to audio source: $SOURCE"
pactl set-default-source "$SOURCE"

# Verify the change
CURRENT_SOURCE=$(pactl get-default-source)
if [ "$CURRENT_SOURCE" = "$SOURCE" ]; then
    echo "Successfully switched to: $SOURCE"
else
    echo "Failed to switch audio source."
    exit 1
fi