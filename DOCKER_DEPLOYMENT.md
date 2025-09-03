# 🐳 RentalCore Docker Hub Deployment Guide

This guide explains how to build, push to Docker Hub, and deploy RentalCore on any system using Docker.

## 📋 Table of Contents
1. [Prerequisites](#prerequisites)
2. [Building and Pushing to Docker Hub](#building-and-pushing-to-docker-hub)
3. [Deploying from Docker Hub](#deploying-from-docker-hub)
4. [Configuration](#configuration)
5. [Production Deployment](#production-deployment)
6. [Troubleshooting](#troubleshooting)

---

## 🔧 Prerequisites

### For Building & Pushing:
- Docker installed and running
- Docker Hub account
- Access to this source code

### For Deployment Only:
- Docker and Docker Compose installed
- Access to your MySQL database
- The deployment files (docker-compose.prod.yml and .env)

---

## 🏗️ Building and Pushing to Docker Hub

### Step 1: Login to Docker Hub
```bash
docker login
```
Enter your Docker Hub username and password.

### Step 2: Build the Image
Replace `nbt4` with your actual Docker Hub username:

```bash
# Build the image
docker build -t nbt4/rentalcore:latest .

# Optional: Tag with version number
docker build -t nbt4/rentalcore:v1.0.0 .
```

### Step 3: Push to Docker Hub
```bash
# Push latest tag
docker push nbt4/rentalcore:latest

# Push version tag (if created)
docker push nbt4/rentalcore:v1.0.0
```

### Step 4: Verify Upload
Visit `https://hub.docker.com/r/nbt4/rentalcore` to confirm your image is uploaded.

---

## 🚀 Deploying from Docker Hub

### Step 1: Download Deployment Files
On your target server, create a new directory and download these files:
- `docker-compose.prod.yml`
- `.env.template`

```bash
mkdir rentalcore-deployment
cd rentalcore-deployment

# Download files (replace with your actual URLs)
wget https://raw.githubusercontent.com/nbt4/rentalcore/main/docker-compose.prod.yml
wget https://raw.githubusercontent.com/nbt4/rentalcore/main/.env.template
```

### Step 2: Configure Environment
```bash
# Copy template to create your configuration
cp .env.template .env

# Edit the configuration
nano .env
```

**Important**: Update the Docker image name in `docker-compose.prod.yml`:
```yaml
services:
  rentalcore:
    image: nbt4/rentalcore:latest
```

### Step 3: Configure Your Database
Edit `.env` file with your database settings:
```env
# Database Configuration
DB_HOST=your-mysql-host.com
DB_PORT=3306
DB_NAME=your-database-name
DB_USERNAME=your-db-username
DB_PASSWORD=your-secure-password

# Security (REQUIRED)
ENCRYPTION_KEY=your-32-character-encryption-key-here
```

### Step 4: Deploy
```bash
# Start the application
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f rentalcore
```

---

## ⚙️ Configuration

### Required Environment Variables
```env
# Database (Required)
DB_HOST=your-database-host
DB_NAME=your-database-name
DB_USERNAME=your-db-user
DB_PASSWORD=your-db-password

# Security (Required)
ENCRYPTION_KEY=generate-a-32-character-key
```

### Optional Environment Variables
```env
# Application
APP_PORT=8080
GIN_MODE=release

# Email (for notifications and invoices)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# Invoice Settings
DEFAULT_TAX_RATE=19.0
CURRENCY_SYMBOL=€
CURRENCY_CODE=EUR
```

### Generating Encryption Key
```bash
# Generate a secure 32-character key
openssl rand -hex 16
```

---

## 🏭 Production Deployment

### With Reverse Proxy (Recommended)
Create a reverse proxy setup with nginx or Traefik:

**nginx example:**
```nginx
server {
    listen 80;
    server_name rentalcore.yourdomain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### SSL/HTTPS Setup
Use Let's Encrypt with certbot:
```bash
sudo certbot --nginx -d rentalcore.yourdomain.com
```

### Backup Volumes
Your data is stored in Docker volumes. To backup:
```bash
# Create backup directory
mkdir backups

# Backup uploads
docker run --rm -v rentalcore-deployment_rentalcore_uploads:/data -v $(pwd)/backups:/backup alpine tar czf /backup/uploads.tar.gz -C /data .

# Backup logs  
docker run --rm -v rentalcore-deployment_rentalcore_logs:/data -v $(pwd)/backups:/backup alpine tar czf /backup/logs.tar.gz -C /data .
```

---

## 🔍 Troubleshooting

### Health Check
Test if the application is running:
```bash
curl http://localhost:8080/health
# Should return: {"service":"RentalCore","status":"ok"}
```

### Database Connection Issues
```bash
# Test database connectivity
docker-compose -f docker-compose.prod.yml exec rentalcore nc -z $DB_HOST $DB_PORT
```

### View Application Logs
```bash
# Real-time logs
docker-compose -f docker-compose.prod.yml logs -f rentalcore

# Last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100 rentalcore
```

### Common Issues

**1. Database Connection Failed**
- Check DB credentials in `.env`
- Ensure database server is accessible from Docker container
- Verify firewall settings

**2. Permission Denied**
- Check file permissions on volumes
- Ensure Docker has permission to access volumes

**3. Out of Memory**
- Increase Docker memory limits
- Monitor resource usage: `docker stats`

### Restart Services
```bash
# Restart application only
docker-compose -f docker-compose.prod.yml restart rentalcore

# Restart all services
docker-compose -f docker-compose.prod.yml restart

# Full stop and start
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d
```

---

## 📁 File Structure for Deployment

Your deployment directory should look like this:
```
rentalcore-deployment/
├── docker-compose.prod.yml
├── .env
├── .env.template
└── backups/  (optional)
```

---

## 🔄 Updates

To update to a new version:
```bash
# Pull the latest image
docker-compose -f docker-compose.prod.yml pull

# Restart with new image
docker-compose -f docker-compose.prod.yml up -d
```

---

## 📞 Support

- **Documentation**: Check this file for common issues
- **Logs**: Always check application logs first
- **Health Check**: Use `/health` endpoint to verify service status
- **Database**: Verify database connectivity separately

**Remember**: Always backup your data before updates!