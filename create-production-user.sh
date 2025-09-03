#!/bin/bash

# JobScanner Pro - Production User Creation Script
# Creates an admin user for the production deployment

set -e

echo "🔧 JobScanner Pro - Production User Setup"
echo ""

# Check if production config exists
if [ ! -f "config.production.direct.json" ]; then
    echo "❌ Error: config.production.direct.json not found"
    echo "Please ensure the production config file exists in the current directory"
    exit 1
fi

# Prompt for user details
echo "Creating admin user for production deployment..."
echo ""

read -p "👤 Username: " USERNAME
read -p "📧 Email: " EMAIL
read -s -p "🔒 Password: " PASSWORD
echo ""
read -p "👤 First Name (optional): " FIRSTNAME
read -p "👤 Last Name (optional): " LASTNAME

echo ""
echo "Creating user with production database..."

# Create user using production config
go run create_user.go \
    -config=config.production.direct.json \
    -username="$USERNAME" \
    -email="$EMAIL" \
    -password="$PASSWORD" \
    -firstname="$FIRSTNAME" \
    -lastname="$LASTNAME"

echo ""
echo "✅ Production admin user created successfully!"
echo ""
echo "🌐 You can now log in to the production application at:"
echo "   http://your-server:8080/login"
echo ""
echo "📝 Credentials:"
echo "   Username: $USERNAME"
echo "   Email: $EMAIL"
echo ""
echo "🔒 For security, please:"
echo "   1. Use a strong password"
echo "   2. Enable HTTPS in production"
echo "   3. Restrict access to the application"