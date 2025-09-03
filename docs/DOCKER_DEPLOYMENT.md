# RentalCore Docker Deployment Guide

## Overview
This comprehensive guide covers all aspects of deploying RentalCore using Docker in production environments.

## Prerequisites

### System Requirements
- **Operating System**: Linux (Ubuntu 20.04+, CentOS 8+, Debian 10+)
- **Docker Engine**: 20.10 or later
- **Docker Compose**: 2.0 or later
- **Memory**: Minimum 2GB RAM, recommended 4GB+
- **Storage**: Minimum 10GB free space
- **CPU**: 2 cores recommended

### External Dependencies
- **Database**: MySQL 8.0+ or MariaDB 10.5+
- **Reverse Proxy**: Traefik, Nginx, or Apache (for SSL termination)
- **DNS**: Properly configured domain name for production

## Quick Start

### 1. Download Configuration Files
```bash
# Create deployment directory
mkdir -p /opt/rentalcore
cd /opt/rentalcore

# Download configuration templates
wget https://github.com/nbt4/rentalcore/raw/main/docker-compose.example.yml
wget https://github.com/nbt4/rentalcore/raw/main/.env.example
wget https://github.com/nbt4/rentalcore/raw/main/config.json.example

# Rename to production files
mv docker-compose.example.yml docker-compose.yml
mv .env.example .env
mv config.json.example config.json
```

### 2. Configure Environment
```bash
# Edit environment variables
nano .env

# Edit application configuration
nano config.json

# Edit Docker Compose settings
nano docker-compose.yml
```

### 3. Deploy
```bash
# Create necessary directories
mkdir -p uploads logs keys

# Start the application
docker-compose up -d

# Verify deployment
docker-compose ps
docker-compose logs -f rentalcore
```

## Production Configuration

### Environment Variables (.env)
```bash
# Database Configuration
DB_HOST=your-database-host.example.com
DB_PORT=3306
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=your_secure_database_password

# Security Settings
ENCRYPTION_KEY=your-256-bit-encryption-key-here
SESSION_SECRET=your-session-secret-key-here
SESSION_TIMEOUT=3600

# Application Settings
GIN_MODE=release
LOG_LEVEL=info
PORT=8080
MAX_UPLOAD_SIZE=10485760

# Email Configuration (Optional)
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=your_smtp_password
```

### Docker Compose Configuration
```yaml
version: '3.8'

services:
  rentalcore:
    image: nbt4/rentalcore:latest
    container_name: rentalcore
    restart: unless-stopped
    
    # Environment
    env_file:
      - .env
    
    # Ports (use reverse proxy in production)
    ports:
      - "127.0.0.1:8080:8080"
    
    # Volumes
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
      - ./config.json:/app/config.json
      - ./keys:/app/keys
    
    # Health Check
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    
    # Resource Limits
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '0.5'
        reservations:
          memory: 512M
          cpus: '0.25'
    
    # Logging
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Reverse Proxy Configuration

### Traefik (Recommended)
```yaml
version: '3.8'

services:
  rentalcore:
    image: nbt4/rentalcore:latest
    container_name: rentalcore
    restart: unless-stopped
    env_file:
      - .env
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
      - ./config.json:/app/config.json
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.rentalcore.rule=Host(`rental.yourdomain.com`)"
      - "traefik.http.services.rentalcore.loadbalancer.server.port=8080"
      - "traefik.http.routers.rentalcore.tls.certresolver=letsencrypt"
      - "traefik.http.routers.rentalcore.middlewares=secure-headers"
      - "traefik.http.middlewares.secure-headers.headers.customRequestHeaders.X-Forwarded-Proto=https"
    networks:
      - traefik
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  traefik:
    external: true
```

### Nginx
```nginx
server {
    listen 80;
    server_name rental.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name rental.yourdomain.com;
    
    # SSL Configuration
    ssl_certificate /etc/ssl/certs/rental.yourdomain.com.crt;
    ssl_certificate_key /etc/ssl/private/rental.yourdomain.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    
    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    
    # Proxy Configuration
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }
    
    # File Upload Configuration
    client_max_body_size 10M;
}
```

## Database Setup

### External MySQL Database
```sql
-- Create database and user
CREATE DATABASE rentalcore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'rentalcore_user'@'%' IDENTIFIED BY 'secure_password_here';
GRANT ALL PRIVILEGES ON rentalcore.* TO 'rentalcore_user'@'%';
FLUSH PRIVILEGES;

-- Import schema (download from GitHub)
mysql -u rentalcore_user -p rentalcore < rentalcore_setup.sql
```

### Docker MySQL (Development Only)
```yaml
# Add to docker-compose.yml for development
services:
  mysql:
    image: mysql:8.0
    container_name: rentalcore-mysql
    restart: unless-stopped
    environment:
      MYSQL_DATABASE: rentalcore
      MYSQL_USER: rentalcore_user
      MYSQL_PASSWORD: secure_password
      MYSQL_ROOT_PASSWORD: root_password
    volumes:
      - mysql_data:/var/lib/mysql
      - ./database/rentalcore_setup.sql:/docker-entrypoint-initdb.d/setup.sql
    ports:
      - "127.0.0.1:3306:3306"

volumes:
  mysql_data:
```

## SSL/TLS Configuration

### Let's Encrypt with Traefik
Traefik automatically handles Let's Encrypt certificates:
```yaml
# traefik.yml
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@yourdomain.com
      storage: /certificates/acme.json
      httpChallenge:
        entryPoint: web
```

### Manual SSL Certificates
```bash
# Create keys directory
mkdir -p keys

# Copy your certificates
cp yourdomain.crt keys/
cp yourdomain.key keys/

# Set proper permissions
chmod 600 keys/*
```

## Monitoring and Logging

### Application Monitoring
```bash
# Health check endpoint
curl https://rental.yourdomain.com/health

# Container status
docker-compose ps

# Resource usage
docker stats rentalcore
```

### Log Management
```bash
# View real-time logs
docker-compose logs -f rentalcore

# View last 100 lines
docker-compose logs --tail=100 rentalcore

# Application logs
tail -f logs/application.log
tail -f logs/error.log
```

### Log Rotation
```bash
# Configure logrotate
sudo nano /etc/logrotate.d/rentalcore

# Content:
/opt/rentalcore/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    notifempty
    create 644 root root
    postrotate
        docker-compose restart rentalcore
    endscript
}
```

## Backup and Recovery

### Automated Backup Script
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/backups/rentalcore"
DATE=$(date +%Y%m%d_%H%M%S)
CONTAINER_NAME="rentalcore"

# Create backup directory
mkdir -p $BACKUP_DIR

# Database backup
docker exec mysql-container mysqldump -u root -p$MYSQL_ROOT_PASSWORD rentalcore > $BACKUP_DIR/database_$DATE.sql

# Files backup
tar -czf $BACKUP_DIR/uploads_$DATE.tar.gz -C /opt/rentalcore uploads/
tar -czf $BACKUP_DIR/logs_$DATE.tar.gz -C /opt/rentalcore logs/

# Configuration backup
cp /opt/rentalcore/.env $BACKUP_DIR/env_$DATE
cp /opt/rentalcore/config.json $BACKUP_DIR/config_$DATE.json

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "*" -mtime +30 -delete

echo "Backup completed: $DATE"
```

### Recovery Procedures
```bash
# Database recovery
mysql -u rentalcore_user -p rentalcore < database_backup.sql

# Files recovery
tar -xzf uploads_backup.tar.gz -C /opt/rentalcore/
tar -xzf logs_backup.tar.gz -C /opt/rentalcore/

# Restart application
docker-compose restart rentalcore
```

## Security Best Practices

### Container Security
```bash
# Run security scan
docker scout cves nbt4/rentalcore:latest

# Update to latest image
docker pull nbt4/rentalcore:latest
docker-compose up -d
```

### File Permissions
```bash
# Set proper ownership
chown -R 1000:1000 /opt/rentalcore/uploads
chown -R 1000:1000 /opt/rentalcore/logs

# Set proper permissions
chmod 600 .env
chmod 600 config.json
chmod 700 keys/
```

### Firewall Configuration
```bash
# UFW example
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 8080/tcp  # Block direct access
ufw enable
```

## Troubleshooting

### Common Issues
1. **Container won't start**: Check logs with `docker-compose logs`
2. **Database connection failed**: Verify credentials and network connectivity
3. **SSL issues**: Check certificate validity and reverse proxy configuration
4. **Performance issues**: Monitor resources with `docker stats`

### Diagnostic Commands
```bash
# Container health
docker-compose ps
docker inspect rentalcore

# Network connectivity
docker exec rentalcore ping database-host
docker exec rentalcore curl -f http://localhost:8080/health

# Resource usage
docker stats --no-stream
df -h
free -h
```

## Maintenance

### Regular Updates
```bash
# Update application
docker pull nbt4/rentalcore:latest
docker-compose up -d

# Verify update
docker-compose logs -f rentalcore
curl https://rental.yourdomain.com/health
```

### Performance Optimization
```bash
# Database optimization
docker exec mysql-container mysql -u root -p -e "OPTIMIZE TABLE rentalcore.*;"

# Clean up unused images
docker image prune -f

# Monitor disk usage
du -sh /opt/rentalcore/*
```

This deployment guide provides comprehensive instructions for production deployment of RentalCore using Docker with proper security, monitoring, and maintenance procedures.