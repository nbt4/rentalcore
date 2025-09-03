#!/bin/bash

# JobScanner Pro - Production Deployment Script

set -e

echo "🚀 Deploying JobScanner Pro to Production..."

# Build the application for production
echo "📦 Building application..."
go build -o server ./cmd/server

# Create logs directory
mkdir -p logs

# Check if production config exists
if [ ! -f "config.production.direct.json" ]; then
    echo "⚠️  Production config file not found!"
    echo "Please ensure config.production.direct.json exists with your production settings"
    echo "You can copy and modify config.production.json as a template"
fi

# Install systemd service (requires root)
if [ "$EUID" -eq 0 ]; then
    echo "🔧 Installing systemd service..."
    cp jobscanner.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable jobscanner
    echo "✅ Systemd service installed and enabled"
    
    echo "📝 To start the service, run:"
    echo "   sudo systemctl start jobscanner"
    echo "   sudo systemctl status jobscanner"
else
    echo "⚠️  Run as root to install systemd service:"
    echo "   sudo ./deploy-production.sh"
fi

echo ""
echo "✅ Configuration:"
echo "   📄 Using config file: config.production.direct.json"
echo "   🌐 Server will run on: http://0.0.0.0:8080"
echo "   📝 Logs location: logs/production.log"
echo ""
echo "👤 Create admin user for production:"
echo "   ./create-production-user.sh"
echo ""
echo "🚀 To start manually:"
echo "   ./start-production.sh"
echo ""
echo "📋 User Management Access:"
echo "   1. Start the application"
echo "   2. Log in with admin credentials"
echo "   3. Navigate to: http://your-server:8080/users"
echo "   4. Click 'Create New User' to add users"
echo ""
echo "✅ Deployment complete!"