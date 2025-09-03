# RentalCore Docker Quick Start

## 🚀 Get Running in 5 Minutes

This guide gets RentalCore running quickly for development and testing. For production deployment, see the [Docker Deployment Guide](DOCKER_DEPLOYMENT.md).

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- External MySQL database (or use Docker MySQL for testing)

## Option 1: With External Database (Recommended)

### Step 1: Download Configuration
```bash
# Create project directory
mkdir rentalcore && cd rentalcore

# Download files
curl -O https://github.com/nbt4/rentalcore/raw/main/docker-compose.example.yml
curl -O https://github.com/nbt4/rentalcore/raw/main/.env.example
curl -O https://github.com/nbt4/rentalcore/raw/main/config.json.example

# Rename to active files
mv docker-compose.example.yml docker-compose.yml
mv .env.example .env
mv config.json.example config.json
```

### Step 2: Configure Database
Edit `.env` file:
```bash
nano .env
```

Update these values:
```bash
DB_HOST=your-database-host.com
DB_NAME=rentalcore
DB_USERNAME=your_db_user
DB_PASSWORD=your_secure_password
```

### Step 3: Setup Database
```bash
# Download and import database schema
curl -O https://github.com/nbt4/rentalcore/raw/main/database/rentalcore_setup.sql
mysql -h your-host -u your-user -p your-database < rentalcore_setup.sql
```

### Step 4: Launch Application
```bash
# Start RentalCore
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f rentalcore
```

### Step 5: Access Application
- **URL**: http://localhost:8080
- **Username**: admin
- **Password**: admin123
- **⚠️ Change password immediately!**

---

## Option 2: With Docker MySQL (Testing Only)

### Step 1: Complete Docker Setup
```bash
mkdir rentalcore && cd rentalcore

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: rentalcore-mysql
    restart: unless-stopped
    environment:
      MYSQL_DATABASE: rentalcore
      MYSQL_USER: rentalcore_user
      MYSQL_PASSWORD: demo_password
      MYSQL_ROOT_PASSWORD: root_password
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "127.0.0.1:3306:3306"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      timeout: 20s
      retries: 10

  rentalcore:
    image: nbt4/rentalcore:latest
    container_name: rentalcore
    restart: unless-stopped
    environment:
      - DB_HOST=mysql
      - DB_NAME=rentalcore
      - DB_USERNAME=rentalcore_user
      - DB_PASSWORD=demo_password
      - GIN_MODE=release
    ports:
      - "8080:8080"
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
    depends_on:
      mysql:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  mysql_data:
EOF
```

### Step 2: Launch Everything
```bash
# Create directories
mkdir -p uploads logs

# Start all services
docker-compose up -d

# Wait for services to be ready
sleep 30

# Import database schema
curl -o rentalcore_setup.sql https://github.com/nbt4/rentalcore/raw/main/database/rentalcore_setup.sql
docker exec -i rentalcore-mysql mysql -u rentalcore_user -pdemo_password rentalcore < rentalcore_setup.sql
```

### Step 3: Access Application
- **URL**: http://localhost:8080
- **Username**: admin
- **Password**: admin123

---

## ✅ Verification Steps

### 1. Health Check
```bash
curl http://localhost:8080/health
# Expected: {"status":"ok","database":"connected"}
```

### 2. Test Login
1. Open http://localhost:8080
2. Login with admin/admin123
3. Should see dashboard with sample data

### 3. Check Data
Navigate to:
- **Customers**: Should see 5 sample customers
- **Devices**: Should see 10 sample devices
- **Jobs**: Should see 5 sample jobs
- **Analytics**: Should show revenue and equipment data

---

## 🔧 Quick Commands

### View Logs
```bash
# All logs
docker-compose logs -f

# Just RentalCore
docker-compose logs -f rentalcore

# Just MySQL
docker-compose logs -f mysql
```

### Restart Services
```bash
# Restart everything
docker-compose restart

# Restart just RentalCore
docker-compose restart rentalcore
```

### Update RentalCore
```bash
# Pull latest version
docker pull nbt4/rentalcore:latest

# Recreate container with new image
docker-compose up -d rentalcore
```

### Stop Everything
```bash
# Stop services (data preserved)
docker-compose stop

# Stop and remove containers
docker-compose down

# Stop and remove everything including data
docker-compose down -v
```

---

## 🚨 Important Security Notes

### For Production Use:
1. **Change default password** immediately after first login
2. **Use external database** with secure credentials
3. **Enable HTTPS** with proper SSL certificates
4. **Configure firewall** to restrict access
5. **Set up regular backups**

### Default Credentials to Change:
- **Application**: admin/admin123
- **Database** (if using Docker MySQL): root/root_password

---

## 📁 Directory Structure After Setup
```
rentalcore/
├── docker-compose.yml    # Docker services configuration
├── .env                 # Environment variables (if using Option 1)
├── config.json          # Application configuration (if using Option 1)
├── uploads/             # File uploads (created automatically)
├── logs/               # Application logs (created automatically)
└── rentalcore_setup.sql # Database schema (downloaded)
```

---

## 🆘 Troubleshooting

### Container Won't Start
```bash
# Check logs for errors
docker-compose logs rentalcore

# Common fixes:
# 1. Ensure port 8080 is available
# 2. Check database connectivity
# 3. Verify environment variables
```

### Database Connection Failed
```bash
# Test database connectivity
docker exec rentalcore ping mysql  # For Docker MySQL
# OR
docker exec rentalcore ping your-external-db-host

# Check database credentials in logs
docker-compose logs rentalcore | grep -i database
```

### Can't Access Web Interface
```bash
# Check if service is running
docker-compose ps

# Check port mapping
netstat -tulpn | grep :8080

# Test direct connection
curl http://localhost:8080/health
```

### Need More Help?
- Check [Troubleshooting Guide](TROUBLESHOOTING.md)
- View [Full Deployment Guide](DOCKER_DEPLOYMENT.md)
- Open issue on [GitHub](https://github.com/nbt4/rentalcore/issues)

---

**🎉 You're ready to start managing equipment rentals with RentalCore!**