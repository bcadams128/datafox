#!/bin/bash

# Test ping
echo "=== Testing /ping ==="
curl -s http://localhost:8080/ping
echo

# Test logs with some sample data
echo "=== Testing /logs ==="
curl -s -X POST http://localhost:8080/logs \
  -d 'Jan 29 10:15:01 app server started
Jan 29 10:15:02 app connected to database
Jan 29 10:15:05 app request received from 192.168.1.1'

echo
echo "=== Done ==="
