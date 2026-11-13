#!/bin/bash

# Generate fake Apache access logs using flog
# Writes two log files to ./logs/

echo "Writing Apache logs to $LOGS_DIR..."

flog -t log -f apache_combined -o logs/apache.log -b 104857600 -p 10485760 -w

flog -t log -f json -o logs/json.log -b 104857600 -p 10485760 -w

