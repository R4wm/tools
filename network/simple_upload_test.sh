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

# Upload with time measurement and speed tracking
echo "[2/4] Uploading to prsmusa.com:/tmp via SCP..."
echo "Method: SCP (Secure Copy Protocol over SSH)"
echo ""

# Arrays to store speed samples
SPEED_SAMPLES=()
TIMESTAMPS=()

START_TIME=$(date +%s.%N)

# Start monitoring in background
REMOTE_FILE="/tmp/speedtest_1gb.bin"
MONITOR_LOG="/tmp/upload_monitor_$$.log"
> "$MONITOR_LOG"

(
    LAST_SIZE=0
    SAMPLE_INTERVAL=5
    while true; do
        sleep $SAMPLE_INTERVAL

        # Check remote file size
        CURRENT_SIZE=$(ssh r4wm@prsmusa.com "stat -c%s $REMOTE_FILE 2>/dev/null || echo 0")

        if [ "$CURRENT_SIZE" = "0" ]; then
            continue
        fi

        # Calculate speed for this interval
        BYTES_DIFF=$((CURRENT_SIZE - LAST_SIZE))
        SPEED_MBps=$(awk "BEGIN {printf \"%.2f\", $BYTES_DIFF / ($SAMPLE_INTERVAL * 1024 * 1024)}")
        ELAPSED=$(awk "BEGIN {printf \"%.1f\", $(date +%s.%N) - $START_TIME}")

        # Log the sample
        echo "$ELAPSED:$SPEED_MBps" >> "$MONITOR_LOG"

        LAST_SIZE=$CURRENT_SIZE

        # Exit if file transfer is complete
        if [ "$CURRENT_SIZE" -ge "$FILE_SIZE" ]; then
            break
        fi
    done
) &

MONITOR_PID=$!

# Upload with real-time progress
scp "$TEST_FILE" r4wm@prsmusa.com:/tmp/speedtest_1gb.bin

END_TIME=$(date +%s.%N)

# Stop monitoring
kill $MONITOR_PID 2>/dev/null
wait $MONITOR_PID 2>/dev/null

echo ""

# Read speed samples
if [ -f "$MONITOR_LOG" ]; then
    while IFS=: read -r timestamp speed; do
        TIMESTAMPS+=("$timestamp")
        SPEED_SAMPLES+=("$speed")
    done < "$MONITOR_LOG"
    rm -f "$MONITOR_LOG"
fi

# Calculate average speed based on total time
DURATION=$(awk "BEGIN {printf \"%.2f\", $END_TIME - $START_TIME}")
AVG_SPEED_MBPS=$(awk "BEGIN {printf \"%.2f\", ($FILE_SIZE * 8) / ($DURATION * 1000000)}")
AVG_SPEED_MBps=$(awk "BEGIN {printf \"%.2f\", $FILE_SIZE / ($DURATION * 1024 * 1024)}")

# Calculate peak speed from samples
PEAK_SPEED_MBps=0
for speed in "${SPEED_SAMPLES[@]}"; do
    PEAK_SPEED_MBps=$(awk "BEGIN {if ($speed > $PEAK_SPEED_MBps) print $speed; else print $PEAK_SPEED_MBps}")
done

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
echo ""

# Show peak speed if we have samples
if [ "${#SPEED_SAMPLES[@]}" -gt 0 ] && [ "$(echo "$PEAK_SPEED_MBps > 0" | bc)" -eq 1 ]; then
    PEAK_SPEED_MBPS=$(awk "BEGIN {printf \"%.2f\", $PEAK_SPEED_MBps * 8}")
    echo "Peak Speed (sampled):"
    echo "  ${PEAK_SPEED_MBPS} Mbps"
    echo "  ${PEAK_SPEED_MBps} MB/s"
    echo ""
fi

# Display speed samples over time
if [ "${#SPEED_SAMPLES[@]}" -gt 0 ]; then
    echo "========================================"
    echo "Speed Samples (every 5 seconds):"
    echo "========================================"
    echo "Time (s) | Speed (MB/s) | Graph"
    echo "---------|--------------|----------------------------------------"

    for i in "${!TIMESTAMPS[@]}"; do
        timestamp="${TIMESTAMPS[$i]}"
        speed="${SPEED_SAMPLES[$i]}"

        # Create simple ASCII bar graph
        bar_length=$(awk "BEGIN {printf \"%.0f\", ($speed / $PEAK_SPEED_MBps) * 30}")
        bar=$(printf '%*s' "$bar_length" | tr ' ' '█')

        printf "%8s | %12s | %s\n" "$timestamp" "$speed" "$bar"
    done
    echo ""

    # Save data to CSV for external graphing
    CSV_FILE="/tmp/upload_speed_data.csv"
    echo "timestamp,speed_mbps" > "$CSV_FILE"
    for i in "${!TIMESTAMPS[@]}"; do
        timestamp="${TIMESTAMPS[$i]}"
        speed="${SPEED_SAMPLES[$i]}"
        echo "${timestamp},${speed}" >> "$CSV_FILE"
    done
    echo "Speed data saved to: $CSV_FILE"
    echo "Use this file to create graphs with gnuplot, Excel, or Python"
    echo ""
fi


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
