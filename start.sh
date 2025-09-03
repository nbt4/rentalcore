#!/bin/bash

# JobScanner Pro - Go Web Application Startup Script

echo "🚀 Starting JobScanner Pro..."

# Set up Go environment
export PATH=$PATH:~/go/bin
export GOPATH=~/gopath

# Check if binary exists, build if not
if [ ! -f "./jobscanner" ]; then
    echo "📦 Building application..."
    go build -o jobscanner cmd/server/main.go
    if [ $? -ne 0 ]; then
        echo "❌ Build failed!"
        exit 1
    fi
    echo "✅ Build successful!"
fi

# Start the application
echo "🌐 Starting web server on http://10.0.0.100:8000"
echo "📊 Dashboard: http://10.0.0.100:8000/jobs"
echo "🔧 Devices: http://10.0.0.100:8000/devices"
echo "👥 Customers: http://10.0.0.100:8000/customers"
echo ""
echo "🌍 External access: http://10.0.0.100:8000"
echo "🏠 Local access: http://localhost:8000"
echo ""
echo "📖 Press Ctrl+C to stop the server"
echo ""

./jobscanner