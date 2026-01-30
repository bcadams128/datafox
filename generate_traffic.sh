#!/bin/bash

# Script to generate HTTP traffic for nginx log testing

NGINX_URL="http://localhost:8080"
ITERATIONS=20
SLEEP_BETWEEN=0.5  # seconds

echo "Generating traffic to $NGINX_URL..."

for i in $(seq 1 $ITERATIONS); do
    # Valid requests to generate access logs
    curl -s -o /dev/null "$NGINX_URL/" &
    curl -s -o /dev/null "$NGINX_URL/index.html" &

    # 404s to generate error logs
    curl -s -o /dev/null "$NGINX_URL/missing-page-$(date +%s)" &
    curl -s -o /dev/null "$NGINX_URL/api/users/$i" &

    # Random user agents
    curl -s -o /dev/null -A "Mozilla/5.0 (Test Bot $i)" "$NGINX_URL/" &

    echo "Iteration $i/$ITERATIONS"
    sleep $SLEEP_BETWEEN
done

wait
echo "Traffic generation complete!"
