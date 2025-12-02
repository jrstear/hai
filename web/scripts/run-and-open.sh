#!/bin/bash
# Script to run web server and open in Chrome

set -euo pipefail

# Get port and API URL from environment (set by Taskfile) or use defaults
PORT=${PORT:-3030}
API_URL=${API_URL:-http://localhost:8080}
BINARY_PATH=${1:-../bin/hai-web}

echo "Starting web server on port $PORT"
echo "API server: $API_URL"

# Start server in background
"$BINARY_PATH" --port "$PORT" --api-url "$API_URL" &
SERVER_PID=$!

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping server (PID: $SERVER_PID)..."
    set +e  # Disable exit on error for cleanup
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    set -e
    exit 0
}

# Trap signals to cleanup
trap cleanup EXIT INT TERM

# Wait a moment for server to start
sleep 2

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "Error: Server failed to start"
    exit 1
fi

# Open Chrome (macOS)
if command -v open > /dev/null 2>&1; then
    open -a "Google Chrome" "http://localhost:$PORT" 2>/dev/null || open "http://localhost:$PORT"
# Linux
elif command -v xdg-open > /dev/null 2>&1; then
    xdg-open "http://localhost:$PORT"
else
    echo "Please open http://localhost:$PORT in your browser"
fi

echo "Server running on http://localhost:$PORT (PID: $SERVER_PID)"
echo "Press Ctrl+C to stop"

# Wait for server process
wait $SERVER_PID

