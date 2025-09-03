#!/bin/bash

# TS RentalCore - Development Restart Script
# Kills existing processes and starts with latest code

set -e

echo "🔄 Restarting TS RentalCore in Development Mode..."

# Kill any existing server processes
echo "🛑 Stopping existing server processes..."
pkill -f "go run cmd/server/main.go" || true
pkill -f "./server" || true
fuser -k 8080/tcp || true

# Give processes time to stop
sleep 2

# Build and start with latest code
echo "📦 Building and starting with latest code..."
go run cmd/server/main.go