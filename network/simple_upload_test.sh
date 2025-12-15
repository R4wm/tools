#!/bin/bash

echo "========================================"
echo "Simple Upload Speed Test to prsmusa.com"
echo "========================================"
echo ""

# Generate 1GB test file
echo "[1/4] Generating 1GB test file..."
TEST_FILE="/tmp/speedtest_1gb.bin"
dd if=/dev/urandom of="$TEST_FILE" bs=1M count=1024 status=progress 2>&1
echo ""
echo "✓ Generated $TEST_FILE (1GB)"
echo ""

# Get file size
FILE_SIZE=$(stat -c%s "$TEST_FILE")
FILE_SIZE_MB=$(awk "BEGIN {printf \"%.2f\", $FILE_SIZE / 1024 / 1024}")
echo "File size: ${FILE_SIZE_MB} MB"
echo ""

# Upload with time measurement
echo "[2/4] Uploading to prsmusa.com:/tmp via SCP..."
echo "Method: SCP (Secure Copy Protocol over SSH)"
echo "Starting upload..."
echo ""
START_TIME=$(date +%s.%N)

# Capture SCP output to parse speed
SCP_OUTPUT=$(scp "$TEST_FILE" r4wm@prsmusa.com:/tmp/speedtest_1gb.bin 2>&1)

END_TIME=$(date +%s.%N)
echo ""

# Calculate average speed based on total time
DURATION=$(awk "BEGIN {printf \"%.2f\", $END_TIME - $START_TIME}")
AVG_SPEED_MBPS=$(awk "BEGIN {printf \"%.2f\", ($FILE_SIZE * 8) / ($DURATION * 1000000)}")
AVG_SPEED_MBps=$(awk "BEGIN {printf \"%.2f\", $FILE_SIZE / ($DURATION * 1024 * 1024)}")

# Try to extract peak speed from SCP output (if available)
PEAK_SPEED=$(echo "$SCP_OUTPUT" | grep -oP '\d+\.\d+MB/s' | head -1 || echo "")

echo "✓ Upload complete!"
echo ""
echo "========================================"
echo "Upload Performance Results"
echo "========================================"
echo "Method:              SCP over SSH"
echo "File size:           ${FILE_SIZE_MB} MB (1024 MB)"
echo "Total duration:      ${DURATION} seconds"
echo ""
echo "Average Speed:"
echo "  ${AVG_SPEED_MBPS} Mbps (megabits per second)"
echo "  ${AVG_SPEED_MBps} MB/s (megabytes per second)"
if [ -n "$PEAK_SPEED" ]; then
echo ""
echo "Peak Speed:          $PEAK_SPEED (as reported by SCP)"
fi
echo ""

# Delete local file
echo "[3/4] Deleting local test file..."
rm -f "$TEST_FILE"
echo "✓ Deleted $TEST_FILE"
echo ""

# Delete remote file
echo "[4/4] Deleting remote test file..."
ssh r4wm@prsmusa.com "rm -f /tmp/speedtest_1gb.bin"
echo "✓ Deleted remote file"
echo ""

echo "========================================"
echo "Test Complete!"
echo "========================================"
