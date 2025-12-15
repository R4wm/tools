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
echo "[2/4] Uploading to prsmusa.com:/tmp..."
echo "Starting upload..."
START_TIME=$(date +%s.%N)

scp "$TEST_FILE" r4wm@prsmusa.com:/tmp/speedtest_1gb.bin

END_TIME=$(date +%s.%N)
echo ""

# Calculate speed
DURATION=$(awk "BEGIN {printf \"%.2f\", $END_TIME - $START_TIME}")
SPEED_MBPS=$(awk "BEGIN {printf \"%.2f\", ($FILE_SIZE * 8) / ($DURATION * 1000000)}")
SPEED_MBps=$(awk "BEGIN {printf \"%.2f\", $FILE_SIZE / ($DURATION * 1024 * 1024)}")

echo "✓ Upload complete!"
echo ""
echo "========================================"
echo "Results:"
echo "========================================"
echo "File size:     ${FILE_SIZE_MB} MB"
echo "Duration:      ${DURATION} seconds"
echo "Upload speed:  ${SPEED_MBPS} Mbps"
echo "Upload speed:  ${SPEED_MBps} MB/s"
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
