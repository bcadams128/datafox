#!/bin/bash

# Generate fake Apache access logs using flog
# Writes two log files to ./logs/

LOGS_DIR="$(dirname "$0")/logs"
mkdir -p "$LOGS_DIR"

echo "Writing Apache logs to $LOGS_DIR..."

flog -f apache_common -t log -o "$LOGS_DIR/apache1.log" -n 10000 -w &
PID1=$!

flog -f apache_common -t log -o "$LOGS_DIR/apache2.log" -n 10000 -w &
PID2=$!

echo "Log generators running (PIDs: $PID1, $PID2). Press Ctrl+C to stop."

trap "kill $PID1 $PID2 2>/dev/null; echo 'Stopped.'" EXIT
wait
