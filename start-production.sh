#!/run/current-system/sw/bin/bash

# TS RentalCore - Production Startup Script
# Uses config.json for configuration

set -e

echo "Starting TS RentalCore in Production Mode..."

# Set Go to release mode for better performance
export GIN_MODE=release

# Create logs directory if it doesn't exist
mkdir -p logs

# Check if production config file exists
if [ ! -f "config.json" ]; then
    echo "Error: config.json not found"
    echo "Please ensure the production config file exists in the current directory"
    exit 1
fi

# Always build the latest binary to ensure code changes are included
echo "📦 Building latest binary..."
go build -o server ./cmd/server
echo "✅ Binary built successfully"

echo "Using configuration file: config.json"
echo "Server will start on port 8080"
echo "Logs will be written to: logs/production.log"

# Start the application with production config
exec ./server -config=config.json >> logs/production.log 2>&1
