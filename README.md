# 🎯 RentalCore - Professional Equipment Rental Management System

A comprehensive, enterprise-grade equipment rental management system built with Go, featuring advanced analytics, device tracking, and customer management. Designed for exclusive Docker deployment with professional theming and modern web interface.

## ✨ Key Features

### 📊 **Advanced Analytics Dashboard**
- **Real-time Analytics**: Revenue trends, equipment utilization, customer metrics
- **Interactive Charts**: Chart.js integration with responsive visualizations  
- **Device Analytics**: Individual device performance with detailed insights modal
- **Time Period Filtering**: 7 days, 30 days, 90 days, 1 year analysis
- **Export Functionality**: PDF and CSV export with UTF-8 encoding
- **Performance Metrics**: Utilization rates, booking statistics, revenue analysis

### 🏢 **Equipment Management**
- **Device Inventory**: Complete equipment tracking with categories and products
- **Availability Tracking**: Real-time device status (available, checked out, maintenance)
- **QR Code & Barcode Generation**: Automated code generation for device identification
- **Bulk Operations**: Mass device assignment and status updates
- **Equipment Packages**: Predefined equipment bundles for common rentals
- **Revenue Tracking**: Per-device revenue analytics and performance insights

### 👥 **Customer & Job Management**
- **Customer Database**: Comprehensive customer information with rental history
- **Job Lifecycle**: Complete job management from creation to completion
- **Device Assignment**: Bulk scanning and assignment to rental jobs
- **Invoice Generation**: Professional invoice creation with customizable templates
- **Status Tracking**: Real-time job status updates with audit trails

### 💼 **Professional Features**
- **RentalCore Design System**: Professional dark theme with consistent branding
- **Responsive Interface**: Mobile-first design with tablet and desktop optimization
- **PWA Support**: Progressive Web App features for mobile deployment
- **Multi-language Support**: Internationalization ready
- **Document Management**: File upload, signature collection, document archival

### 🔐 **Security & Administration**  
- **User Management**: Role-based access control with security audit logs
- **2FA Authentication**: Two-factor authentication with WebAuthn support
- **Encryption**: Industry-standard data encryption and secure key management
- **GDPR Compliance**: Privacy controls and data retention management
- **Security Monitoring**: Real-time security event tracking and alerting

### 📈 **Business Intelligence**
- **Financial Dashboard**: Revenue tracking, payment monitoring, tax reporting
- **Performance Monitoring**: System metrics, error tracking, health checks
- **Audit Logging**: Comprehensive activity logging for compliance
- **Backup Management**: Automated data backup with retention policies

## 🚀 Quick Start (Docker Deployment)

### Prerequisites
- Docker Engine 20.10+
- Docker Compose 2.0+
- External MySQL/MariaDB database
- Domain with SSL certificate (production)

### 1. Get Configuration Files
```bash
git clone https://github.com/nbt4/RentalCore.git
cd RentalCore

# Or download configuration files directly
wget https://github.com/nbt4/RentalCore/raw/main/docker-compose.example.yml
wget https://github.com/nbt4/RentalCore/raw/main/.env.example
wget https://github.com/nbt4/RentalCore/raw/main/config.json.example
```

### 2. Configure Environment
```bash
# Copy and configure environment variables
cp .env.example .env
nano .env  # Edit with your database credentials

# Copy and configure application settings  
cp config.json.example config.json
nano config.json  # Customize application settings

# Copy and configure Docker Compose
cp docker-compose.example.yml docker-compose.yml
nano docker-compose.yml  # Adjust for your environment
```

### 3. Deploy
```bash
# Start the application
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f rentalcore

# Access the application
open http://localhost:8080
```

## 🏗️ Project Architecture

### 📁 **Directory Structure**
```
rentalcore/
├── cmd/server/              # Application entry point
├── internal/
│   ├── handlers/           # HTTP request handlers
│   ├── models/             # Database models and structures
│   ├── services/           # Business logic services
│   └── middleware/         # HTTP middleware
├── web/
│   ├── templates/          # HTML templates with modern design
│   └── static/            # CSS, JavaScript, assets
├── migrations/             # Database migration scripts
├── keys/                   # SSL certificates and keys
├── logs/                   # Application logs
├── uploads/               # User uploaded files
├── docker-compose.yml      # Docker deployment configuration
├── .env                   # Environment variables (not in repo)
├── config.json            # Application configuration (not in repo)
├── .gitignore             # Git ignore rules with credential protection
└── README.md              # This documentation
```

### 🛠️ **Technology Stack**
- **Backend**: Go 1.23+ with Gin web framework
- **Database**: MySQL 8.0+ with GORM ORM
- **Frontend**: HTML5, Bootstrap 5, Chart.js, vanilla JavaScript
- **Authentication**: WebAuthn, 2FA, session management
- **Deployment**: Docker with health checks and volume management
- **Monitoring**: Prometheus metrics, structured logging
- **Security**: TLS encryption, CORS protection, input validation

## 🔧 Configuration Management

### **Environment Variables (.env)**
```bash
# Database Configuration
DB_HOST=your-database-host.com
DB_PORT=3306
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=secure_password

# Security Settings
ENCRYPTION_KEY=your-256-bit-encryption-key
SESSION_TIMEOUT=3600
GIN_MODE=release

# Optional: Email Configuration
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=email_password
```

### **Application Configuration (config.json)**
- **UI Theming**: Professional dark theme with customizable colors
- **Feature Flags**: Enable/disable specific functionality
- **Performance Settings**: Cache timeouts, connection pooling
- **Invoice Configuration**: Tax rates, payment terms, currency settings
- **Security Policies**: Password requirements, session management

## 📊 API Documentation

### **Analytics Endpoints**
- `GET /analytics` - Main analytics dashboard
- `GET /analytics/devices/:deviceId` - Individual device analytics
- `GET /analytics/devices/all` - All device revenue data
- `GET /analytics/export` - Export analytics data (PDF/CSV)

### **Core Management APIs**
- `GET|POST|PUT|DELETE /api/v1/jobs` - Job management
- `GET|POST|PUT|DELETE /api/v1/devices` - Device inventory
- `GET|POST|PUT|DELETE /api/v1/customers` - Customer database
- `GET|POST|PUT|DELETE /api/v1/invoices` - Invoice management

### **Utility Endpoints**
- `GET /health` - Application health check
- `GET /barcodes/device/:id/qr` - Generate QR codes
- `POST /search/global` - Global search functionality
- `GET /monitoring/metrics` - Prometheus metrics

## 🔐 Security Features

### **Authentication & Authorization**
- Multi-factor authentication (2FA, WebAuthn)
- Role-based access control (RBAC)
- Session management with secure cookies
- Password policy enforcement

### **Data Protection**
- Industry-standard encryption (AES-256)
- HTTPS/TLS termination
- Input validation and sanitization
- SQL injection prevention
- CSRF protection

### **Compliance & Auditing**
- GDPR compliance features
- Comprehensive audit logging
- Data retention policies
- Security event monitoring

## 🚀 Production Deployment

### **Docker Hub Images**
```bash
# Latest stable release
docker pull nbt4/rentalcore:latest

# Specific version
docker pull nbt4/rentalcore:1.5.0
```

### **Reverse Proxy Integration**
```yaml
# Traefik labels example
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.rentalcore.rule=Host(`rental.yourdomain.com`)"
  - "traefik.http.services.rentalcore.loadbalancer.server.port=8080"
  - "traefik.http.routers.rentalcore.tls.certresolver=letsencrypt"
```

### **Monitoring & Maintenance**
```bash
# Health check
curl https://rental.yourdomain.com/health

# View logs
docker-compose logs -f --tail=100 rentalcore

# Update deployment
docker-compose pull && docker-compose up -d

# Backup data
docker run --rm -v rentalcore_uploads:/data -v $(pwd):/backup alpine tar czf /backup/backup.tar.gz /data
```

## 📈 Performance & Scaling

### **Optimization Features**
- Database connection pooling (50 connections default)
- Response caching for analytics data
- Lazy loading for large datasets
- Image optimization and compression
- Minified CSS/JavaScript assets

### **Monitoring Metrics**
- Application performance monitoring (APM)
- Database query performance
- Memory and CPU utilization
- Error rate and response time tracking
- User activity analytics

## 🛠️ Development

### **Local Development Setup**
```bash
# Clone repository
git clone https://github.com/nbt4/RentalCore.git
cd RentalCore

# Copy configuration examples
cp .env.example .env
cp config.json.example config.json

# Build and run
go mod tidy
go build -o server ./cmd/server
./server

# Or use Docker for development
docker-compose up -d --build
```

### **Contributing**
1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Documentation

- **[Docker Deployment Guide](DOCKER_DEPLOYMENT.md)** - Comprehensive deployment instructions
- **[Docker Quick Start](DOCKER_QUICK_START.md)** - Rapid deployment guide
- **[API Documentation](docs/api.md)** - Complete API reference
- **[Security Guide](docs/security.md)** - Security best practices

## 🏷️ Version History

### **v1.5.0** (Latest)
- ✅ Complete device analytics modal with detailed insights
- ✅ Enhanced Docker deployment with configuration examples
- ✅ Comprehensive .gitignore with credential protection
- ✅ UTF-8 PDF export fixes for proper currency formatting

### **v1.4.0**
- ✅ Device analytics modal implementation
- ✅ Enhanced analytics dashboard with working dropdowns
- ✅ Complete analytics functionality restoration

### **v1.3.0** 
- ✅ Analytics dashboard complete rewrite
- ✅ Fixed dropdown functionality and data display issues
- ✅ Professional RentalCore theming implementation

## 📧 Support & Contact

- **Issues**: [GitHub Issues](https://github.com/nbt4/RentalCore/issues)
- **Docker Hub**: [nbt4/rentalcore](https://hub.docker.com/r/nbt4/rentalcore)
- **Documentation**: [GitHub Repository](https://github.com/nbt4/RentalCore)

## 📄 License

This project is a professional equipment rental management system designed for enterprise use. All rights reserved.

---

**🎯 Ready for Production**: RentalCore is designed for professional equipment rental businesses requiring comprehensive analytics, robust security, and scalable Docker deployment.