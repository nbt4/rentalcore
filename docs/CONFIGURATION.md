# RentalCore Configuration Guide

## Environment Variables (.env)

### Database Configuration
```bash
# Database Connection
DB_HOST=your-database-host.com
DB_PORT=3306
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=secure_password_here
DB_CHARSET=utf8mb4
DB_PARSE_TIME=true
DB_LOC=UTC

# Connection Pool Settings
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300
```

### Security Configuration
```bash
# Encryption and Security
ENCRYPTION_KEY=your-256-bit-encryption-key-here
SESSION_SECRET=your-session-secret-key
SESSION_TIMEOUT=3600
CORS_ALLOWED_ORIGINS=https://yourdomain.com
```

### Application Settings
```bash
# Server Configuration
PORT=8080
GIN_MODE=release
LOG_LEVEL=info
UPLOAD_PATH=/app/uploads
MAX_UPLOAD_SIZE=10485760

# Feature Flags
ENABLE_2FA=true
ENABLE_AUDIT_LOG=true
ENABLE_METRICS=true
```

### Email Configuration (Optional)
```bash
# SMTP Settings
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=email_password
SMTP_FROM=RentalCore <noreply@yourdomain.com>
```

## Application Configuration (config.json)

### UI Configuration
```json
{
  "ui": {
    "theme": "dark",
    "company_name": "Your Company",
    "company_logo": "/static/images/logo.png",
    "timezone": "Europe/Berlin",
    "currency": "EUR",
    "date_format": "DD.MM.YYYY"
  }
}
```

### Feature Configuration
```json
{
  "features": {
    "analytics": true,
    "qr_codes": true,
    "pdf_export": true,
    "bulk_operations": true,
    "device_packages": true
  }
}
```

### Invoice Configuration
```json
{
  "invoice": {
    "tax_rate": 19.0,
    "payment_terms": 30,
    "invoice_prefix": "INV",
    "footer_text": "Thank you for your business!"
  }
}
```

### Performance Settings
```json
{
  "performance": {
    "cache_timeout": 300,
    "max_results_per_page": 50,
    "analytics_cache_hours": 24,
    "session_cleanup_interval": 3600
  }
}
```

## Docker Compose Configuration

### Basic Configuration
```yaml
version: '3.8'

services:
  rentalcore:
    image: nbt4/rentalcore:latest
    container_name: rentalcore
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=your-db-host
      - DB_NAME=rentalcore
      - DB_USERNAME=rentalcore_user
      - DB_PASSWORD=secure_password
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
      - ./config.json:/app/config.json
    restart: unless-stopped
```

### Production Configuration with Proxy
```yaml
version: '3.8'

services:
  rentalcore:
    image: nbt4/rentalcore:latest
    container_name: rentalcore
    env_file:
      - .env
    volumes:
      - uploads_data:/app/uploads
      - logs_data:/app/logs
      - ./config.json:/app/config.json
      - ./keys:/app/keys
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.rentalcore.rule=Host(`rental.yourdomain.com`)"
      - "traefik.http.services.rentalcore.loadbalancer.server.port=8080"
      - "traefik.http.routers.rentalcore.tls.certresolver=letsencrypt"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  uploads_data:
  logs_data:
```

## Configuration Validation

### Environment Validation
RentalCore validates all environment variables on startup and will fail to start with clear error messages if required variables are missing.

### Configuration File Validation
The application validates the config.json file structure and provides default values for missing optional settings.

## Security Considerations

### Credential Management
- Never commit actual credentials to version control
- Use strong, randomly generated passwords
- Rotate encryption keys regularly
- Use environment-specific configurations

### File Permissions
- Ensure config files have appropriate permissions (600 or 644)
- Protect sensitive directories from public access
- Use dedicated service account for running the application