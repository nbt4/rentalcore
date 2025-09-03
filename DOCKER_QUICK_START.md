# 🚀 RentalCore - Docker Quick Start

Deploy RentalCore in 5 minutes using Docker Hub!

## 📦 For Deployment (End Users)

### 1. Create Deployment Directory
```bash
mkdir rentalcore-app && cd rentalcore-app
```

### 2. Download Required Files
```bash
# Download docker-compose file
curl -O https://raw.githubusercontent.com/nbt4/rentalcore/main/docker-compose.prod.yml

# Download environment template
curl -O https://raw.githubusercontent.com/nbt4/rentalcore/main/.env.template
```

### 3. Configure Environment
```bash
# Create your configuration
cp .env.template .env

# Edit with your settings
nano .env
```

**Minimum required settings:**
```env
# Database
DB_HOST=your-mysql-host.com
DB_NAME=your_database_name
DB_USERNAME=your_db_user
DB_PASSWORD=your_secure_password

# Security (generate with: openssl rand -hex 16)
ENCRYPTION_KEY=your-32-character-encryption-key
```

### 4. Update Docker Image
Edit `docker-compose.prod.yml` and update the image name:
```yaml
services:
  rentalcore:
    image: nbt4/rentalcore:latest
```

### 5. Deploy
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### 6. Access Application
- Open: `http://your-server:8080`
- Health check: `http://your-server:8080/health`

---

## 🛠️ For Developers (Building & Publishing)

### 1. Set Your Docker Hub Username
```bash
./build-and-push.sh --set-username nbt4
```

### 2. Login to Docker Hub
```bash
docker login
```

### 3. Build and Push
```bash
./build-and-push.sh
```

This will:
- ✅ Build the Docker image
- ✅ Tag with `latest` and timestamp
- ✅ Push to Docker Hub
- ✅ Make it available for deployment anywhere

---

## 📋 Requirements

**For Deployment:**
- Docker & Docker Compose
- MySQL database (local or cloud)

**For Building:**
- Docker
- Docker Hub account
- Source code access

---

## 🔗 Links

- **Full Documentation**: [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- **Environment Template**: [.env.template](.env.template)
- **Production Compose**: [docker-compose.prod.yml](docker-compose.prod.yml)

---

## ⚡ One-Liner Deployment

```bash
mkdir rentalcore && cd rentalcore && curl -O https://raw.githubusercontent.com/nbt4/rentalcore/main/docker-compose.prod.yml && curl -O https://raw.githubusercontent.com/nbt4/rentalcore/main/.env.template && cp .env.template .env && echo "Edit .env file with your settings, then run: docker-compose -f docker-compose.prod.yml up -d"
```