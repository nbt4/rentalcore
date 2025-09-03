# Docker Deployment Guide

This guide explains how to deploy the Go Barcode Webapp using Docker Compose.

## Prerequisites

- Docker and Docker Compose installed on your system
- Access to the external MySQL database at tsunami-events.de

## Quick Start

1. **Clone the repository** (if not already done):
   ```bash
   git clone <repository-url>
   cd go-barcode-webapp
   ```

2. **Configure environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env with your specific configuration
   ```

3. **Build and start the application**:
   ```bash
   docker compose up -d
   ```

4. **Access the application**:
   - Open your browser and go to `http://localhost:8080`
   - The application will be available on the port specified in your `.env` file

## Configuration

### Environment Variables

The application is configured using environment variables in the `.env` file:

#### Database Configuration
- `DB_HOST`: Database host (default: tsunami-events.de)
- `DB_PORT`: Database port (default: 3306)
- `DB_NAME`: Database name (default: TS-Lager)
- `DB_USERNAME`: Database username
- `DB_PASSWORD`: Database password

#### Server Configuration
- `APP_PORT`: External port for the application (default: 8080)
- `SERVER_HOST`: Internal server host (default: 0.0.0.0)
- `SERVER_PORT`: Internal server port (default: 8080)
- `GIN_MODE`: Application mode (release/debug)

#### Security Configuration
- `ENCRYPTION_KEY`: Key for encryption operations
- `SESSION_TIMEOUT`: Session timeout in seconds

#### Email Configuration (Optional)
- `SMTP_HOST`: SMTP server host
- `SMTP_PORT`: SMTP server port
- `SMTP_USERNAME`: SMTP username
- `SMTP_PASSWORD`: SMTP password
- `FROM_EMAIL`: From email address
- `FROM_NAME`: From name
- `USE_TLS`: Enable TLS (true/false)

## Docker Commands

### Start the application
```bash
docker compose up -d
```

### Stop the application
```bash
docker compose down
```

### View logs
```bash
docker compose logs -f go-barcode-webapp
```

### Rebuild the application
```bash
docker compose build --no-cache
docker compose up -d
```

### Update the application
```bash
git pull
docker compose build
docker compose up -d
```

## Persistent Data

The following data is persisted using Docker volumes:
- `uploads/`: File uploads
- `logs/`: Application logs
- `archives/`: Archive files

## Health Checks

The application includes health checks:
- Container health check endpoint: `http://localhost:8080/health`
- Database connectivity check before application startup

## Troubleshooting

### Check container status
```bash
docker compose ps
```

### View application logs
```bash
docker compose logs go-barcode-webapp
```

### Check database connectivity
```bash
docker compose logs db-health-check
```

### Access container shell
```bash
docker compose exec go-barcode-webapp sh
```

### Verify environment variables
```bash
docker compose exec go-barcode-webapp env
```

## Production Considerations

1. **Security**:
   - Change the default `ENCRYPTION_KEY` in production
   - Use strong database passwords
   - Consider using Docker secrets for sensitive data

2. **Performance**:
   - Adjust resource limits in docker-compose.yml
   - Monitor container resource usage
   - Consider using a reverse proxy (nginx/traefik)

3. **Backups**:
   - The application data (uploads, logs) is stored in Docker volumes
   - Implement regular backups of these volumes
   - Database backups should be handled separately

4. **Updates**:
   - Test updates in a staging environment first
   - Use proper CI/CD pipelines for production deployments
   - Consider blue-green deployments for zero-downtime updates

## Network Configuration

The application uses a custom Docker network (`rental-network`) to isolate containers and ensure proper communication between services.

## Monitoring

Monitor the application using:
- Docker container logs
- Application health endpoint
- Container resource usage
- Database connection status

## Support

For issues or questions:
1. Check the application logs first
2. Verify environment variable configuration  
3. Ensure database connectivity
4. Check Docker container status