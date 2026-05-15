# 🎯 RentalCore - Professional Equipment Rental Management System

A comprehensive, enterprise-grade equipment rental management system built with Go, featuring advanced analytics, device tracking, and customer management. Designed for exclusive Docker deployment with professional theming and modern web interface.

## 📑 Table of Contents

- [✨ Key Features](#-key-features)
- [🚀 Quick Start](#-quick-start-docker-deployment)
- [🏗️ Project Architecture](#️-project-architecture)
- [🔧 Configuration](#-configuration-management)
- [📊 API Documentation](#-api-documentation)
- [🔐 Security](#-security-features)
- [🚀 Deployment](#-production-deployment)
- [📈 Performance](#-performance--scaling)
- [🛠️ Development](#️-development)
- [📝 Documentation](#-documentation)
- [📱 Responsive Design](#-responsive-design)
- [📷 Demo Images](#-demo-images)
- [🏷️ Version History](#️-version-history)
- [📧 Support](#-support--contact)

## ✨ Key Features

### 📊 **Advanced Analytics Dashboard**
- **Real-time Analytics**: Revenue trends, equipment utilization, customer metrics
- **Interactive Charts**: Chart.js integration with responsive visualizations  
- **Device Analytics**: Individual device performance with detailed insights modal
- **Time Period Filtering**: 7 days, 30 days, 90 days, 1 year analysis
- **Export Functionality**: PDF and CSV export with UTF-8 encoding
- **Performance Metrics**: Utilization rates, booking statistics, revenue analysis

### 🏢 **Equipment Management**
- **Product Catalog (WarehouseCore)**: Manage product master data exclusively in WarehouseCore (`/admin/products`); RentalCore consumes the catalog read-only for jobs, invoices, and device assignments
- **Device Management (WarehouseCore)**: Full device CRUD operations now managed exclusively in WarehouseCore (`/admin/devices`); RentalCore provides read-only device access for job assignments, invoices, and analytics
- **Cable Management (WarehouseCore)**: Complete cable CRUD operations now managed exclusively in WarehouseCore (`/admin/cables`); includes cable types, connectors, and length specifications
- **Case Management (WarehouseCore)**: Complete case CRUD operations now managed exclusively in WarehouseCore (`/admin/cases`); includes device mapping, labels, and case tracking
- **Availability Tracking**: Real-time device status integration from WarehouseCore (available, checked out, maintenance)
- **QR Code & Barcode Generation**: Device identification codes now managed in WarehouseCore
- **Bulk Operations**: Mass device assignment and status updates
- **🆕 Package Booking to Jobs**: Assign entire packages to jobs as single line items with automatic device reservation
- **Revenue Tracking**: Per-device revenue analytics and performance insights
- **🆕 Rental Equipment System**: External equipment rental tracking with supplier management
- **🆕 Manual Entry & Selection**: Add external rentals directly to jobs or select from catalog
- **🆕 Rental Analytics**: Dedicated analytics for external equipment usage and costs
- **🆕 Accessories & Consumables System**: Complete inventory tracking for accessories and consumables
  - **Inventory Dashboard**: Real-time stock monitoring with low stock alerts
  - **Generic Barcodes**: Single barcode per product type (not individual units)
  - **Measurement Units**: Flexible count types (pieces, kg, liters, etc.)
  - **Job Integration**: Accessories/consumables displayed under devices in job views
  - **Scan Tracking**: Real-time scanning status (scanned out/in quantities)
  - **Stock Management**: Manual adjustments with reason tracking and audit trail
  - **Auto-Calculation**: Automatic stock updates when items are scanned

### 📄 **PDF Processing & OCR**
- **📤 PDF Upload**: Upload invoices, quotes, and delivery notes for automatic processing
- **🗂️ File Pool Integration**: All uploaded PDFs stored in File Pool as single source of truth - no file duplication
- **🔍 OCR Text Extraction**: Automatic text extraction from PDF documents
- **🧠 Intelligent Parsing**: Smart detection of customer info, document numbers, dates, line items, prices
- **📊 Extraction Review**: Visual review screen showing all extracted data with confidence scores
- **🔗 Auto-Mapping**: Fuzzy matching algorithm automatically maps PDF products to database products
- **📦 Package Suggestions**: WarehouseCore package alias map powers bundle-level recommendations (configure `WAREHOUSECORE_ALIAS_MAP_URL` or reuse `WAREHOUSECORE_DOMAIN`)
- **💡 Smart Suggestions**: AI-powered product suggestions with confidence scoring (>80% auto-accept)
- **🎯 Manual Mapping**: Searchable product database for manual mapping when needed
- **📚 Learning System**: Saves both product and package mappings to improve future auto-mapping accuracy
- **🔄 Package Mapping**: Package mappings persist to database and are suggested in future PDFs
- **🎯 Alias Support**: Multiple different PDF texts can map to the same product/package (one-to-many)
- **✅ Package Preservation**: Items mapped to packages are added as packages (not expanded to products)
- **💰 Price Extraction**: Line totals and unit prices from PDFs applied as custom pricing
- **🔄 Workflow Integration**: Complete workflow from upload → review → mapping → job creation with mixed product/package support
- **📂 Pool Document OCR**: Process existing File Pool documents for OCR without re-uploading

### 👥 **Customer & Job Management**
- **Customer Database**: Comprehensive customer information with rental history
- **Job Lifecycle**: Complete job management from creation to completion
- **Automatic Job Codes**: Consistent `JOB000123` style identifiers generated for every job
- **Enhanced Job Modals**: Revenue and device count display with detailed overview
- **Product-Based Assignment**: Build job manifests by selecting products with live availability checks; RentalCore resolves device allocations automatically
- **Device Price Management**: Real-time price adjustment per job with API integration
- **Categorized Device Overview**: Devices grouped by Sound, Light, Effect, Stage, Other
- **Invoice Generation**: Professional invoice creation with customizable templates
- **Status Tracking**: Real-time job status updates with audit trails

### 💼 **Professional Features**
- **RentalCore Design System**: Professional dark theme with consistent branding
- **🆕 Fully Responsive Design**: Complete mobile-first responsive implementation
  - **Mobile Navigation**: Drawer-style navigation with backdrop and touch optimization
  - **Tablet Interface**: Icon rail navigation with compact layouts
  - **Desktop Experience**: Full sidebar with comprehensive layouts
  - **Responsive Tables**: Card transformation for mobile, horizontal scroll with sticky columns
  - **Adaptive Forms**: Single-column mobile, multi-column desktop with responsive grids
  - **Touch-Optimized**: 44px minimum touch targets, enhanced focus states
- **🆕 Personalized Dashboard Widgets**: User-level control over homepage metrics with quick links to core modules
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
- External PostgreSQL database
- Domain with SSL certificate (production)

### 1. Get Configuration Files
```bash
git clone https://git.server-nt.de/ntielmann/rentalcore.git
cd RentalCore

# Or download configuration files directly
wget https://git.server-nt.de/ntielmann/rentalcore/raw/main/docker-compose.example.yml
wget https://git.server-nt.de/ntielmann/rentalcore/raw/main/.env.example
wget https://git.server-nt.de/ntielmann/rentalcore/raw/main/config.json.example
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

**🎉 On a fresh system, the database is automatically initialized!**

The `RentalCore.sql` file is automatically imported when you start the stack for the first time. No manual database setup required!

```bash
# Start the application (database auto-initializes on first run)
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f rentalcore

# Wait for PostgreSQL to initialize (first run takes ~30 seconds)
# Watch the logs to see when initialization is complete
docker-compose logs -f postgres

# Access the application
open http://localhost:8081
```

**📝 How Auto-Initialization Works:**
- On **first start**, PostgreSQL automatically imports `/opt/dev/cores/RentalCore.sql`
- The database schema and default admin user are created automatically
- This only happens when the `postgres-data` volume is empty (fresh install)

**🔄 To Reset Database (Fresh Install):**
```bash
# Stop and remove volumes (⚠️ DELETES ALL DATA!)
docker-compose down -v

# Start again (triggers fresh database init)
docker-compose up -d
```

**⚠️ Common Issues on Fresh System:**

1. **Restart Loop on First Start?**
   - **Normal!** PostgreSQL needs 30-60 seconds to import the database
   - Wait and monitor: `docker-compose logs -f postgres`
   - Once you see "database system is ready to accept connections", services will start automatically

2. **Can't Login with admin/admin?**
   - You likely have an existing PostgreSQL volume from a previous install
   - **Solution:** `docker-compose down -v` then `docker-compose up -d`
   - This will delete all data and reinitialize the database

3. **Services Won't Start?**
   - Check logs: `docker-compose logs --tail=100`
   - Pull images: `docker-compose pull`
   - Check ports: `sudo lsof -i :8081 :5432`

**📖 Detailed Troubleshooting:** See [DEPLOYMENT_GUIDE.md](../DEPLOYMENT_GUIDE.md) for complete deployment instructions and troubleshooting.

### 🔑 Default Login Credentials

The admin user is **automatically created** on first database initialization:

- **Username:** `admin`
- **Password:** `admin`
- **Email:** `admin@localhost`
- **Roles:** `super_admin`, `admin`, `warehouse_admin`

**⚠️ IMPORTANT:** The default admin is now forced to change the password on the very first login before the dashboard can be accessed.

**Default roles provisioned by the schema:**

| Role | Scope | Description |
|------|-------|-------------|
| `super_admin` | Global | Full access across RentalCore + WarehouseCore |
| `admin` | RentalCore | Complete RentalCore administration |
| `manager` | RentalCore | Jobs, customers, devices management |
| `operator` | RentalCore | Operational flows incl. scanning |
| `viewer` | RentalCore | Read-only |
| `warehouse_admin` | Warehouse | Full warehouse administration |
| `warehouse_manager` | Warehouse | Warehouse operations + reporting |
| `warehouse_worker` | Warehouse | Day-to-day warehouse scanning/tasks |
| `warehouse_viewer` | Warehouse | Read-only warehouse access |

### 🔄 Integrated Deployment with WarehouseCore

For integrated deployment of both RentalCore and WarehouseCore together, use the root docker-compose configuration:

```bash
# Navigate to the parent directory (NOT a git repo)
cd /opt/dev/lager_weidelbach

# Pull latest images from Docker Hub
docker compose pull

# Start both services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f rentalcore
docker compose logs -f warehousecore
```

**Access the applications:**
- **RentalCore**: http://localhost:8081
- **WarehouseCore**: http://localhost:8082

**Cross-navigation:**
Both applications feature navbar links to seamlessly switch between RentalCore and WarehouseCore with a single click.

**Note:** The images use `:latest` tags. Pull periodically to get the newest versions:
```bash
docker compose pull && docker compose up -d
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
- **Database**: PostgreSQL 17+ with GORM ORM
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
DB_PORT=5432
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=secure_password
DB_SSLMODE=disable

# Security Settings
ENCRYPTION_KEY=your-256-bit-encryption-key
SESSION_TIMEOUT=3600
GIN_MODE=release

# Nextcloud File Pool (WebDAV)
NEXTCLOUD_WEBDAV_URL=https://cloud.example.com/remote.php/dav/files/username
NEXTCLOUD_WEBDAV_USER=your-nextcloud-user
NEXTCLOUD_WEBDAV_PASSWORD=your-nextcloud-app-password
NEXTCLOUD_WEBDAV_BASE_PATH=rentalcore-filepool
FILEPOOL_ASSIGNED_ROOT=assigned
FILEPOOL_UNASSIGNED_ROOT=unassigned
# Disable automatic migration of existing local uploads by setting to false
# NEXTCLOUD_BACKFILL_ON_START=false

# Optional: Email Configuration
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=email_password
```
`FILEPOOL_ASSIGNED_ROOT` and `FILEPOOL_UNASSIGNED_ROOT` control which WebDAV folders appear in the Assigned vs. Unassigned cards—the pool UI only shows files that actually live inside those directories.

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
- `GET /api/v1/devices` - Device inventory (read-only)
- `GET /api/v1/devices/:id` - Device details (read-only)
- `GET /api/v1/devices/available` - Available devices for assignment
- `GET /api/v1/devices/available/job/:jobId` - Availability scoped to a job
- `GET /api/v1/devices/tree/availability` - Hierarchical availability view
- `POST|PUT|DELETE /api/v1/devices` - **410 Gone** (use WarehouseCore for write operations)
- `GET|POST|PUT|DELETE /api/v1/customers` - Customer database
- `GET|POST|PUT|DELETE /api/v1/invoices` - Invoice management

### **Utility Endpoints**
- `GET /health` - Application health check
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
docker pull nbt4/rentalcore:1.4
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
git clone https://git.server-nt.de/ntielmann/rentalcore
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

All documentation is organized in the `docs/` folder for easy access:

### Core Documentation
- **[Database Setup Guide](docs/DATABASE_SETUP.md)** - Complete database installation and configuration guide

### Deployment Guides
- **[Docker Deployment](docs/DOCKER_DEPLOYMENT.md)** - Comprehensive deployment instructions
- **[Quick Start Guide](docs/DOCKER_QUICK_START.md)** - Rapid deployment guide
- **[Configuration Examples](docs/CONFIGURATION.md)** - Environment and config examples

### Technical Documentation
- **[API Reference](docs/API.md)** - Complete API documentation
- **[Security Guide](docs/SECURITY.md)** - Security best practices and features
- **[Architecture Guide](docs/ARCHITECTURE.md)** - System architecture and design patterns

### User Guides
- **[User Manual](docs/USER_GUIDE.md)** - Complete user documentation
- **[Admin Guide](docs/ADMIN_GUIDE.md)** - Administrator documentation
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions

## 🏷️ Version History

### **v5.3.50** - fix: POST /employees 400-Fehler behoben — date_of_birth als String geparst (nicht direkt time.Time), Adressfelder in separate Felder (street, house_number, zip, city) aufgeteilt

### **v5.3.49** - fix: M365 Kalender-Endpunkt von /users/ auf /groups/ umgestellt (events@tsunami-events.de ist eine M365-Gruppe, kein User-Postfach)

### **v5.3.48** - feat: Mitarbeiterverwaltung, Skills-Verwaltung und M365-Kalender-Integration — Jobs erzeugen/aktualisieren/löschen Kalendertermine in events@tsunami-events.de; Bearbeiter werden als Attendees eingetragen

### **v5.3.47** - fix: GAL bulk-push 409-Konflikte (E-Mail gehört echtem Exchange-Postfach) werden als gesynct markiert statt endlos retryed

### **v5.3.46** - feat: GAL-Sync via Exchange Online admin API — Kontakte erscheinen in OWA-Autocomplete und Outlook auf allen Plattformen; gal_contact_id-Spalte, BulkPushToGAL beim Start, E-Mail-Änderungen korrekt behandelt

### **v5.3.45** - feat: M365 Bulk-Push beim Start — alle bestehenden Kunden ohne m365_id werden einmalig nach M365 gepusht; Beispieldateien um M365-Konfiguration ergänzt

### **v5.3.44** - feat: Bidirektionaler M365-Kontakt-Sync (Shared Mailbox, Delta-Poll alle 5 Min, Last-Write-Wins, M365-Löschungen archivieren Kunden)

### **v5.3.43** - fix: parseItemIntelligently fallback parser — qty-as-price (2-number case) and pos-as-qty (4-number case) bugs corrected; switch-based number assignment with qty×price≈total verification

### **v5.3.42** - feat: multiply_by_days + prices_include_tax flags for job price calculation; UpdatePriceSettings API endpoint; toggle UI in JobPositionsPanel; fix GORM column-tag casing for Product model (500 on position update); UpdateFields method in job_repository

### **v5.3.41** - fix: Pluck("deviceid") PostgreSQL case-sensitive identifier fix; error handling for device ID collection on job delete; add rental/package position types and rental_equipment_id/tax_rate to JobPosition interface

### **v5.3.40** - feat: OCR-Finalize erstellt job_positions; tax_rate + Mietprodukt-Sektion in JobPositionsPanel; editierbare Datum-Inputs im MappingModal

### **v5.3.39** - feat: job positions API (quote-style line items with degressive pricing)

### **v5.3.38** - fix: load existing product requirements when editing a job

### **v5.3.37** - feat: service items for non-physical cost positions (Fahrtkosten, Personal etc.) with OCR auto-mapping

### **v5.3.36** - feat: rental equipment auto-mapping in OCR — saved mappings, fuzzy match, proper type tracking

### **v5.3.35** - fix: OCR parser price/qty plausibility checks; auto-mapping re-evaluates all non-confirmed items on re-upload

### **v5.3.34** - refactor: remove Kontakte page from RentalCore — now managed globally in Cores Dashboard

### **v5.3.33** - feat: rename "Kunden" to "Kontakte" in navigation and all UI texts

### **v5.3.32** - feat: supplier autocomplete picker in rental equipment form; contact role badges, filter tabs, and role checkboxes in Kontakte UI; IsCustomer/IsSupplier to Customer model; rental equipment in OCR inline search; rental-equipment search endpoint for OCR mapping

### **v5.3.31** - Fix: OCR finalize 400-Fehler — Kundenpicker in MappingModal; normalized_text Schema-Fix

### **v5.3.30** - Fix: OCR mapping product-quick-create 500 error; full create modal with Produkt/Paket/Mietprodukt tabs

### **v5.3.29** - Fix: OCR parser now correctly handles multi-page invoices — footer lines (Seite, IBAN, company name) no longer interrupt item parsing; position numbers up to 9999 supported

### **v5.3.28** - Feature: two-stage device availability — job form saves requirements, job detail assigns specific devices
### **v5.3.25** (Latest) - Fix: job delete 500 (employeejob table missing in live DB)
### **v5.3.24** - Fix: job create FK violation on statusid; default to status 1 (Planung) when not provided
### **v5.3.23** - Fix: job detail 404 (job_packages table missing in live DB); bad OCR dates silently discarded
### **v5.3.22** - Fix: show backend error message on job save; validate dates required when products selected
### **v5.3.21** - Fix: double /api/v1 prefix on job save/delete/device-tree requests (404 errors)
### **v5.3.22** - Fix: device availability query — remove table alias to prevent GORM quoting issues; fix CAST(CHAR) truncation bug; remove status filter from conflict check (align with tree picker); add diagnostic logging
### **v5.3.21** - Fix: product_packages PK mapped to column:id (was package_id); create missing DB tables job_packages, job_package_reservations, job_attachments, job_edit_sessions, job_history
### **v5.3.20** - Fix: PDF date (DocumentDate) now seeds start_date in metadata; formatted as YYYY-MM-DD before response
### **v5.3.19** - Fix: remove global RC CSS link; scope rc-* classes to index.css to prevent app-wide style breakage
### **v5.3.18** - Feature: PDF import fills customer/dates into job form; RC color theme in MappingModal; show matched product name; create product/package inline
### **v5.3.17** - Fix: populate legacy filename column in pdf_uploads (NOT NULL violation on upload)
### **v5.3.16** - Feature: Interactive PDF mapping modal with OCR review, confidence bars, 3-tab mapping management
### **v5.3.15** - Fix: PDF mapping upsert broken on PostgreSQL (ON DUPLICATE KEY UPDATE is MySQL-only)
### **v5.3.14** - Fix: move Produkt-Mappings to Tools dropdown in sidebar
### **v5.3.13** - Fix: standalone mapping_management template, base.html nav link
### **v5.3.12** - Fix: add web/templates to Docker image (resolves nil HTMLRender panic)
### **v5.3.11** - feat: add product mappings nav link, clear search input on cancel

- Added "Produkt-Mappings" link in the user dropdown navigation menu
- Fixed UX issue in mapping management: search input now clears when cancelling an edit

---

### **v5.3.10** - Fix: PDF upload "Failed to save upload record" & PostgreSQL sequence sync

- **Bug Fix**: PDF uploads failed with "Failed to save upload record" due to PostgreSQL sequences being out of sync with actual table data (sequences reset to 1 after volume recreation)
- Fixed all `pdf_*` table sequences by aligning them with current MAX ids
- Fixed `ensurePackageMappingColumn/Index/FK` functions: replaced MySQL-specific `AFTER` column syntax and `information_schema.statistics` with PostgreSQL equivalents
- Added proper error logging to the upload handler for easier future debugging

---

### **v5.3.9** - Fix: Existing devices visible in job edit form

- **Bug Fix**: Opening an existing job in the edit (Bearbeiten) window now shows the devices already assigned to that job
- Each device row has a remove button (×) to unassign it directly from the edit form
- Device list refreshes automatically after removal

---

### **v5.3.8** - Feature: Product selection & device management in job form

- **Product Picker**: Collapsible category tree with availability counts and quantity input per product
- **Selected Products Summary**: Editable quantity inputs, remove button per selection
- Products are sent as `selected_products` to the backend on save
- DeviceList now supports removal of individual devices via DELETE API

---

### **v5.3.7** - Feature: PDF/OCR inline import & editable product quantities

- **PDF/OCR from job form**: Uploading a PDF now auto-maps products inline and populates customer, dates and quantities directly into the new job form — no redirect to the review page
- Unmapped items show a link to open the review in a new tab
- **Selected Products table**: Quantities are now editable inputs; each row has a remove button
- Both the summary table and the product tree stay in sync

---

### **v5.3.6** - Fix: DigitalSignature table name and column mapping

- **Bug Fix**: `GET /api/v1/documents` returned HTTP 500 due to GORM querying non-existent table `digital_signature` (singular)
- Added `TableName()` method to `DigitalSignature` struct returning `"digital_signatures"` (plural)
- Fixed GORM column tags on `SignatureID` (`signatureid`) and `DocumentID` (`documentid`) to match actual DB columns
- Fixed raw SQL JOIN in `GetDocumentStats()` to use lowercase `documentid` column name
- Fixed `SignatureData` field type tag from `longtext` (MySQL) to `text` (PostgreSQL)

---

### **v5.2.0** - React SPA Frontend

- 🎨 **Complete UI Rebuild**: Replaced Go-Template frontend with React 19 + Vite + Tailwind v4 SPA
- 🔗 **Unified Design System**: Identical dark theme and glass morphism effects as WarehouseCore
- ⚡ **JSON Auth API**: Added `/api/v1/auth/login|logout|me|change-password` endpoints
- 📱 **Responsive Sidebar**: Collapsible navigation with cross-navigation to WarehouseCore
- 📊 **Analytics Dashboard**: Monthly revenue chart + status distribution
- 🔒 **Role Guards**: Protected admin routes with role-based access control

---

### **v5.1.5** - Critical Database Sequence Fix

**🔧 DATABASE INTEGRITY FIX**

- 🐛 **PostgreSQL Sequence Synchronization**:
  - **Root Cause**: Auto-increment sequences were out of sync with actual table data, causing "duplicate key value" errors
  - **Products Table**: Sequence was at 3, but max productID was 1,000,008 (causing CreateProductPackage failures)
  - **System-Wide**: Fixed all 88 sequences across the database (jobs, customers, cables, cases, users, invoices, etc.)
  - **Migration**: Added `029_fix_sequences.up.sql` to automatically synchronize all sequences on deployment
  - **Prevention**: Migration script detects and fixes sequence drift automatically

- 📊 **Affected Tables Fixed**:
  - `products`: 1,000,008 records synced
  - `jobs`: 1,118 records synced
  - `cables`: 1,123 records synced
  - `cases`: 1,006 records synced
  - `customers`: 67 records synced
  - `users`: 20 records synced
  - `invoices`: 30 records synced
  - Plus 81 additional sequences verified

- ✅ **Testing Verified**:
  - Product package creation now works correctly
  - All auto-increment columns ready for new inserts
  - No data loss, only sequence metadata updated

### **v4.1.38** - Complete Accessories & Consumables System

**🎉 FULL IMPLEMENTATION - ALL FEATURES COMPLETE**

- 📦 **Complete Product Management**:
  - **Product Form Template**: Full product CRUD with accessory/consumable support (`product_form.html`)
  - **Type Selection**: Checkboxes to mark products as accessories or consumables
  - **Inventory Fields**: Count type, stock quantity, minimum stock, generic barcode, price per unit
  - **Conditional Display**: Fields shown only when accessory/consumable is selected
  - **Accessories Management Tab**: Add/remove accessories linked to products with real-time stock display
  - **Consumables Management Tab**: Add/remove consumables linked to products
  - **API Integration**: Full CRUD operations with all backend endpoints

- 🎯 **Job Assignment Modals**:
  - **Accessories Selection Modal**: Checkbox-based selection with quantity inputs
  - **Required vs Optional**: Required accessories pre-selected and locked, optional items user-selectable
  - **Availability Display**: Shows current stock for each item during selection
  - **Consumables Input Modal**: Quantity prompt for consumable items with unit display
  - **Unit Display**: Shows measurement units (kg, liters, pieces) next to quantities
  - **Skip Option**: Users can skip accessories/consumables if not needed
  - **Auto-Submission**: Automatic API calls to assign items to jobs

- 📚 **WarehouseCore Integration Guide** (`WAREHOUSECORE_INTEGRATION_GUIDE.md`):
  - **Step-by-Step Instructions**: Complete implementation guide for product forms, scanning, admin features
  - **Code Examples**: 30+ copy-paste ready code snippets (JavaScript, HTML, API calls)
  - **Scanning Workflows**: Detailed flows for accessories (scan per piece) and consumables (quantity prompt)
  - **UI/UX Guidelines**: Visual design specs, color coding, notifications, sound feedback
  - **Testing Scenarios**: 4 complete test scenarios with expected outcomes
  - **Completion Criteria**: 8-point checklist for verifying implementation
  - **API Reference**: Complete endpoint documentation with request/response examples

- 🔧 **Technical Implementation**:
  - `product_form.html`: 800+ lines - Complete product management interface
  - `job_form.html`: Enhanced with 200+ lines of modal and selection code
  - Modal system with Promise-based async workflows
  - Real-time validation and stock availability checks
  - Responsive design matching RentalCore theme

- ✅ **100% Feature Complete - Issue #37**:
  - Backend: 50+ API endpoints operational ✅
  - Database: 6 tables, 5 views deployed ✅
  - Frontend (RentalCore): All UI components implemented ✅
  - Product Management: Full CRUD with inventory tracking ✅
  - Job Assignment: Modal system for accessories/consumables ✅
  - Documentation: Complete integration guide for WarehouseCore ✅
  - Testing: All scenarios documented and testable ✅

### **v4.1.37** - Accessories & Consumables Frontend Implementation
- 📦 **Inventory Management Dashboard**: New `/inventory` page with comprehensive stock tracking
  - **Low Stock Alerts**: Real-time monitoring of items below minimum threshold with visual indicators
  - **Accessories List**: Complete list of all accessory products with current stock levels and prices
  - **Consumables List**: Tracking of consumable items with measurement units (kg, L, etc.)
  - **Manual Stock Adjustment**: Modal interface for stock adjustments with reason tracking
  - **Transaction History**: Full audit trail of all inventory movements with filtering
- 🔧 **Job Detail View Enhancements**: Accessories and consumables now displayed under devices
  - **Visual Hierarchy**: Items shown indented under parent devices with distinctive icons
  - **Scan Status**: Real-time display of scanned quantities (e.g., `[4/0 scanned]`)
  - **Auto-Loading**: JavaScript automatically fetches and displays items on page load
  - **Icon Differentiation**: Accessories (📎) and consumables (💧) clearly distinguished
- 🎯 **Navigation Integration**: Added Inventory link to main sidebar navigation
- 📚 **Comprehensive Documentation**: New FRONTEND_STATUS.md with implementation details
  - Complete feature breakdown (completed vs pending)
  - WarehouseCore integration requirements
  - Testing checklist and API endpoint documentation
  - Code locations and file structure reference
- 🔌 **API Integration**: Full integration with backend accessories/consumables system
  - `/api/jobs/:id/accessories` - Fetch job accessories
  - `/api/jobs/:id/consumables` - Fetch job consumables
  - `/api/inventory/low-stock` - Low stock monitoring
  - `/api/inventory/adjust` - Manual stock adjustments
  - `/api/inventory/transactions` - Transaction history
- 🎨 **Visual Design**: Consistent with RentalCore design system using CSS variables
- ⚙️ **Backend Foundation**: Complete accessories/consumables system (v4.1.36)

### **v4.1.35** - Advanced Financial Dashboard & Reports Redesign
- 🎨 **Complete Premium UI Redesign**: Financial dashboard and reports pages rebuilt with enterprise SaaS aesthetics
- ✨ **Enhanced Glass Morphism**: Advanced backdrop-blur effects with semi-transparent layered card backgrounds and radial gradient overlays
- 🌈 **Dynamic Animated Backgrounds**: Gradient backgrounds with accent color highlights and smooth animations
- 📊 **6-Card Hero Metrics Grid**: Expanded to include Revenue, Profit, Pending, Overdue, Monthly, and Financial Health Score
  - **Visual Health Ring**: Conic gradient health score visualization with dynamic percentage display
  - **Profit Analytics**: New profit card with growth trends and margin analysis
  - **Smart Trends**: Context-aware trend indicators with color-coded performance metrics
  - **Interactive Detail Modals**: Click any metric to see comprehensive breakdown with detailed analytics
- 📈 **Advanced Chart Visualizations**:
  - Revenue trend with multi-dataset (revenue + expenses) line charts
  - Payment status doughnut chart with smooth hover animations
  - Transaction type distribution charts with color-coded segments
  - Top customers bar chart with currency formatting
- 🎯 **Smart Period Selection**: Elegant pill-button design for time period selection with active state animations
- 📋 **Modern Responsive Tables**:
  - Redesigned with hover effects, status badges, and type icons
  - Automatic mobile transformation to card layout
  - Sticky headers with smooth scrolling
- ⚡ **Quick Actions Grid**: 4+ action cards for common tasks with color-coded icons and descriptions
- 🔢 **Sophisticated Number Animations**: Count-up animations with cubic-bezier easing functions
- ⌨️ **Keyboard Shortcuts**: R (refresh), N (new), Escape (close), with smart context awareness
- 🔄 **Auto-Refresh**: Charts and transactions automatically refresh every 5 minutes with smooth updates
- 📱 **Fully Responsive Design**: Mobile-first, tablet, and desktop layouts with touch optimization
- 🎯 **JetBrains Mono Typography**: Monospace font for precise financial number display
- 📑 **Enhanced Reports Page**:
  - Gradient filter bar with icon labels and visual hierarchy
  - 6 predefined report types (Revenue, Payments, Customers, Jobs, Tax, Overdue)
  - Dynamic content grid with summary stats
  - Advanced export modal with format selection (CSV, PDF, Excel, JSON)
- 🖨️ **Professional Export Options**: Multiple format support with date range and chart inclusion
- 🎭 **Visual Polish**: Subtle shadows, smooth transitions, and micro-interactions throughout

### **v4.1.27** - WarehouseCore Rental Equipment Integration
- 🚛 **Rental Equipment in Jobs**: Fetch and display rental equipment from WarehouseCore API in job creation/edit forms
- 📊 **Supplier Grouping**: Rental equipment displayed sorted and grouped by supplier name for easy navigation
- 🎨 **Visual Distinction**: Purple/violet accent color differentiates rental equipment from owned products (red accent)
- 💰 **Cost Tracking**: Shows both rental price (internal cost) and customer price (revenue) per item
- 📦 **Job Integration**: Full CRUD operations for job_rental_equipment assignments with quantity and days tracking
- 🔢 **Real-time Summary**: Dynamic summary table shows selected rental equipment with cost/revenue totals
- 📝 **Notes Support**: Add notes per rental equipment item for special instructions or requirements
- 🔄 **WarehouseCore Client**: New HTTP client with caching for efficient rental equipment API calls

### **v4.1.26** - Nextcloud Live Sync
- 🔄 **Automatic Nextcloud Sync**: Files added directly to Nextcloud (via Explorer or Nextcloud UI) are automatically detected and tracked in database
- 📂 **Auto-Sync on Access**: ListFilePool and GetPoolDocumentsForOCR now sync from Nextcloud before returning data
- 🔌 **Manual Sync Endpoint**: New GET `/api/v1/documents/pool/sync` endpoint for manual sync triggers
- ✅ **Fixed API Response**: GetPoolDocumentsForOCR now returns proper `success: true` field
- 🧠 **Intelligent Detection**: Smart MIME type and document type detection for synced files
- 📁 **Folder Scanning**: Scans both unassigned and assigned Nextcloud folders recursively
- 🔍 **Duplicate Prevention**: Only creates database entries for files not already tracked
- 📊 **Sync Reporting**: Returns count of newly synced files for monitoring

### **v4.1.25** - File Pool Job Creation UI
- 🎨 **Tab-Based PDF Modal**: Job creation PDF modal now features tabbed interface with "Upload New" and "From Pool" options
- 📂 **Pool Document Selection**: Browse and select existing File Pool documents directly from the job creation modal
- 🔍 **Live Pool List**: Dynamic loading of unassigned PDF documents with filename, size, and upload date
- ✅ **Visual Selection**: Radio-style selection UI with clear selected state and pool item highlighting
- 🎯 **Unified Processing**: Single "Process" button handles both upload and pool document OCR workflows
- 🎨 **Consistent Styling**: Tab toggle uses theme red accent color matching File Pool page design

### **v4.1.24** - File Pool as Single Source of Truth
- 🗂️ **Unified File Storage**: File Pool (documents table) is now the single source of truth for all files
- 🔗 **Document Assignment API**: New methods to assign/unassign documents to jobs without file duplication
- 📤 **Pool Document OCR**: New endpoint `/api/v1/pdf/from-pool/:documentID` to process existing pool documents for OCR
- 📋 **Pool Documents List**: New endpoint `/api/v1/pdf/pool-documents` to list unassigned PDF documents for OCR
- 🔄 **Hybrid Attachments**: Job attachments now query both legacy attachments AND documents tables
- 📂 **Nextcloud Move Support**: Added WebDAV MOVE method for relocating files within Nextcloud storage
- 🗄️ **Database Schema Update**: Added `document_id` foreign key to `pdf_uploads` table for File Pool integration
- 🔌 **PDF Handler Integration**: DocumentHandler now integrated into PDF processing workflow for seamless file management

### **v4.1.23** - Toolbar Separation
- 🎨 **Pool Table Toolbar**: Separated toolbar from table for better visual hierarchy

### **v4.1.22** - File Pool Redesign
- 🎨 **Slim Professional Theme**: Complete theming overhaul of the File Pool page with modern, minimalist design
- 📑 **Tab Navigation**: Clean tab-based navigation replacing bulky card layouts
- 📤 **Streamlined Upload**: Compact upload form with improved file type display and styling
- 📋 **Clean File List**: Modern file list with hover effects, better spacing, and visual hierarchy
- 🎯 **Visual Consistency**: Unified design language with CSS custom properties for colors and spacing
- ⚡ **Improved UX**: Better empty states, refined typography, and polished interactions

### **v4.1.8** - Nextcloud Backfill
- 🌐 **Nextcloud import**: File Pool scan holt WebDAV-Dateien, die noch nicht in der DB sind (system/unassigned), damit auch manuell hochgeladene Files sichtbar werden
- 🔧 **Env-Handling**: WebDAV-Variablen werden entquotet/getrimmt, damit `.env`-Werte mit Anführungszeichen korrekt funktionieren

### **v4.1.4** - File Pool Shell Fix
- 🎨 **File Pool layout**: Page now uses the shared base shell (sidebar/header/scripts) to avoid audit-page redirect quirks and dropped assets
- 🖼️ **Manifest icons**: Missing PWA icon sizes restored to remove 404s in the browser console

### **v4.1.3** - File Pool Shell Fix
- 🎨 **File Pool layout**: Page now uses the shared base shell (sidebar/header/scripts) to avoid audit-page redirect quirks and dropped assets
- 🖼️ **Manifest icons**: Missing PWA icon sizes restored to remove 404s in the browser console

### **v4.1.2** - File Pool UI Integration
- 🎨 **File Pool in App Shell**: File Pool now uses the main RentalCore UI (navbar/theme) and sits under Tools > File Pool
- 🧭 **Navigation**: Documents dropdown links to the pool; Tools dropdown highlights when active

### **v4.1.1** - Nextcloud File Pool
- 📂 **Nextcloud-backed File Pool**: All document uploads now stored in Nextcloud via WebDAV (no local disk copies)
- 🔒 **Role-gated Pool API**: `/documents/pool` (admin/manager) groups files into Assigned vs. Unused (system/unassigned)
- 🔄 **Backfill**: Existing local uploads are migrated to Nextcloud on startup (disable with `NEXTCLOUD_BACKFILL_ON_START=false`)
- ⚙️ **Config via ENV**: `NEXTCLOUD_WEBDAV_URL`, `NEXTCLOUD_WEBDAV_USER`, `NEXTCLOUD_WEBDAV_PASSWORD`, optional `NEXTCLOUD_WEBDAV_BASE_PATH`

### **v4.0.1** - Remove Legacy Equipment Packages & Invoices
- 🔒 Removed legacy Equipment Packages UI/API routes; requests now return HTTP 410 (Gone)
- 🧾 Removed legacy Invoices and Invoice Templates UI/API routes; navigation entries hidden
- 🧭 Navbar Tools dropdown now only lists Analytics and Financial

### **v4.0** - Package System Redesign
- 🎯 **MAJOR ARCHITECTURAL CHANGE**: Complete package system redesign for simplicity
- 📦 **Manual Package Devices**: Packages now treated exactly like products - manually created in WarehouseCore (e.g., PKG_SOUNDM_001, PKG_SOUNDM_002)
- 🔄 **Automatic Real Device Reservation**: When package device assigned to job, backend automatically reserves real devices with discounted prices
- 💰 **Smart Discount Calculation**: Calculates discount percentage from (regular prices - package price) and applies to all items
- 🏷️ **Package Item Flags**: Real devices marked with `is_package_item=true` and excluded from revenue calculation
- ✅ **Revenue Accuracy**: Only package device price counts; real devices don't double-count revenue
- 🗑️ **Removed Virtual Device Creation**: No more automatic PKG_* device generation - user creates package devices manually
- 🔧 **Simplified AssignPackageToJob**: OCR compatibility layer finds real package devices and triggers full assignment logic
- 🎨 **UI Enhancement**: Package items shown in "Assigned Devices" with special marker (📦)
- ⚡ **Performance**: Reduced complexity, cleaner transaction handling, fewer database operations
- 🏗️ **New Functions**: `handlePackageDeviceAssignment()` and `findAvailableDevicesForProduct()` in job_repository.go
- 📊 **Updated Revenue Calculation**: `CalculateAndUpdateRevenue()` now skips `is_package_item=true` devices

### **v3.83** - Debug: RowsAffected Logging
- 🔍 **Added RowsAffected Logging**: Track how many rows GORM actually inserts
- 🐛 **Debug Silent Failures**: Investigate why JobDevice INSERT statements aren't generated
- 📊 **Enhanced Logging**: Log RowsAffected for both virtual and real JobDevice creation

### **v3.82** - Critical Fix: JobDevice Type Mismatch
- 🔴 **CRITICAL BUG FIX**: Fixed JobDevice model type mismatch causing GORM silent failures
- 🔧 **Type Alignment**: Changed JobDevice.JobID from `uint` to `int` to match database schema
- ✅ **Package JobDevices Now Created**: Virtual and real package devices now correctly inserted into jobdevices table
- 📊 **Database Compatibility**: Resolved composite primary key issues with type mismatch (int in DB vs uint in model)
- 🎯 **Root Cause**: GORM silently failed to execute INSERT statements due to type incompatibility
- ⚡ **Impact**: Packages will now appear in "Selected Products" and count towards revenue

### **v3.81** - Package JobDevice Debug Logging
- 🔍 **Comprehensive Debug Logging**: Added detailed logging throughout AssignPackageToJob to track package assignment flow
- 📊 **Transaction Tracking**: Log every step from transaction start to commit with clear markers ([TX], [VERIFY], [JOB_PACKAGE], etc.)
- 🐛 **Troubleshooting Aid**: Helps diagnose why JobDevices aren't being created for packages
- ⚡ **Idempotency Logging**: Clear logging for recovery mode when packages exist but JobDevices don't
- 🎯 **Virtual Device Tracking**: Detailed logs for virtual product, virtual device, and virtual JobDevice creation
- 📦 **Real Device Tracking**: Logs for each real device added to JobDevices with package markers
- ✅ **Success/Failure Indicators**: Clear ERROR markers for failures, SUCCESS for completion

### **v3.70** - Database Collation Fix
- ✅ **Fixed Collation Errors**: Resolved "Illegal mix of collations" errors in package availability queries
- 🔧 **COLLATE Clauses Added**: All JOIN and WHERE conditions now specify utf8mb4_unicode_ci collation
- 📦 **Package Assignment Works**: Packages can now be assigned without collation mismatch errors
- 🎯 **Affected Queries**: Fixed countAvailableDevicesByProduct and findAvailableDevicesByProduct
- 🚀 **Cross-Table Comparisons**: deviceID and jobID comparisons now use consistent collation

### **v3.69** - Package Mapping Auto-Suggestions Fix
- ✅ **Package Auto-Mapping Now Works**: RunAutoMapping now checks saved package mappings for auto-suggestions
- 🎯 **Priority Checking**: PackageMapper.LookupSavedMapping() is checked first before falling back to ProductMapper
- 💡 **Improved Learning**: Previously saved package mappings now automatically suggest in new PDF extractions
- 🔧 **Exported Method**: LookupSavedMapping is now publicly accessible for handler integration
- 🚀 **Complete Workflow**: Package mapping learning system now works end-to-end (save → retrieve → auto-suggest)

### **v3.68** - Complete Column Name Fix
- ✅ **Fixed ALL SQL Errors**: Corrected ALL column names in both countAvailableDevicesByProduct and findAvailableDevicesByProduct
- 🔧 **Complete Fix**: Both availability check functions now use startDate/endDate correctly
- 📦 **Package Assignment Works**: Packages can now be properly assigned to jobs without any SQL errors

### **v3.67** - Partial Column Name Fix (incomplete)
- ⚠️ **Partial Fix**: Only fixed findAvailableDevicesByProduct, missed countAvailableDevicesByProduct
- 🔧 **Package Assignment**: Still had SQL errors in availability counting

### **v3.66** - WarehouseCore Package Integration
- 🔧 **Product-Based Package System**: Job package assignment now uses WarehouseCore's `product_packages` and `product_package_items` tables
- 🗄️ **Database Migration**: Switched from RentalCore's `equipment_packages` to WarehouseCore's `product_packages` for unified package management
- 📦 **Product-Type Reservations**: Device reservations now query by `productID` instead of `deviceID` pattern matching
- 🎯 **Improved Availability**: countAvailableDevicesByProduct() queries devices by product type for accurate availability checking
- ⚡ **Enhanced Performance**: Cleaner queries using exact product IDs instead of LIKE patterns
- 🔄 **Model Update**: JobPackage.Package now references ProductPackage instead of EquipmentPackage
- 🛠️ **Repository Refactor**: Complete rewrite of JobPackageRepository to use WarehouseCore package structure
- ✅ **SQL Error Fix**: Resolved "Error 1146: Table 'RentalCore.equipment_packages' doesn't exist" issue

### **v3.65** - PDF Package Assignment Fix
- 📦 **Package Preservation**: PDF items mapped to packages now stay as packages instead of expanding to individual products
- 🔄 **Smart Assignment**: Packages are added to `job_packages` table with automatic device reservations
- 💰 **Custom Pricing**: PDF line totals and unit prices are extracted and applied as custom package prices
- 🔧 **Mixed Support**: Single PDF can contain both products (assigned individually) and packages (assigned as bundles)
- ⚡ **Improved Workflow**: assignProductsToJob() now separates package handling from product handling
- 🎯 **Price Aggregation**: Multiple PDF items mapping to same package have quantities summed and prices averaged
- ✅ **Graceful Degradation**: Package assignment failures logged as warnings but don't block job creation
- 🛠️ **Repository Integration**: PDFHandler now uses JobPackageRepository for direct package assignment

### **v3.64** - PDF Package Mapping Persistence & One-to-Many Support
- 📦 **Package Mapping Persistence**: Package mappings from PDF extraction are now saved to `pdf_package_mappings` table
- 🔄 **Package Learning System**: User-confirmed package mappings are automatically saved for future auto-suggestions
- 🗄️ **New Database Table**: Added `pdf_package_mappings` table with same structure as product mappings
- 🔧 **Enhanced Mapping Events**: `pdf_mapping_events` now tracks both product_id and package_id for complete audit trail
- 💡 **PackageMapper Service**: New dedicated service for managing package mapping operations
- ✅ **One-to-Many Mappings**: Multiple PDF text strings can now map to the same product/package (alias support)
- 🎯 **Improved Auto-Mapping**: Package suggestions from previous extractions now persist and improve accuracy over time
- 📊 **Event Tracking**: All manual and auto package mappings are logged for analytics and quality control

### **v3.63** - Job Package Booking System
- 📦 **Package Assignment to Jobs**: Packages can now be assigned to jobs as single line items (like products)
- 🔒 **Automatic Device Reservations**: When a package is assigned, underlying devices are automatically reserved in the background
- 💰 **Package Pricing**: Use package prices (not sum of individual products), with override support per job
- 📊 **Availability Checking**: Smart availability checking considers both individual device assignments and package reservations
- 🗄️ **Database Schema**: New tables `job_packages` and `job_package_reservations` for tracking assignments
- 🔌 **API Endpoints**: Full CRUD API for managing packages on jobs (assign, update price/quantity, remove, view reservations)
- 🖥️ **UI Integration**: Job detail page now displays assigned packages with expandable device lists
- 🔄 **Backward Compatible**: Existing jobs with individual devices continue to work unchanged
- 📝 **Migration Support**: SQL migration file included for database updates

### **v3.62** - PDF Error Handling Fix
- ✅ **Fixed 500 Error**: Resolved "device not found in job" error during PDF extraction finalization
- 🛠️ **Deployed Error Handling**: The graceful error handling from v3.61 is now properly deployed to production
- 📦 **Package Mapping Support**: Database migration applied to support package mapping in PDF extractions
- 🔧 **Production Ready**: All recent PDF processing improvements are now live in Docker image

### **v3.61** - Graceful PDF Device Assignment
- ✅ **Resilient Job Creation**: PDF extraction finalization now succeeds even when some devices can't be assigned or priced
- 🛠️ **Warning-Based Errors**: Device assignment and price override failures are logged as warnings instead of fatal errors
- 📊 **Partial Success Handling**: Jobs are created successfully with available devices even if some assignments fail
- 🔍 **Detailed Logging**: Failed device assignments and price updates are logged with specific device IDs for debugging

### **v3.60** - Warehouse Package OCR (Issue #19)
- 📦 **Package Suggestions**: RentalCore now pulls WarehouseCore’s alias map to recognize product bundles directly in the PDF mapping flow.
- 🔁 **Auto-Mapping Upgrade**: Alias hits preempt product fuzzies, so matching “Audio (XLR 3P - XLR 3P)” immediately offers the cable package instead of 37 duplicate rows.
- 🧮 **Job Assignment Expansion**: When a PDF item maps to a package, the assigner now unfolds it into the correct per-product counts before reserving inventory or pricing overrides.
- 🖥️ **UI Enhancements**: Mapping cards highlight package matches with dedicated chips, show the gender/connector metadata, and allow manual searches across both products and packages in a single panel.
- ⚙️ **Config Surface**: Set `WAREHOUSECORE_ALIAS_MAP_URL` (or rely on `WAREHOUSECORE_DOMAIN`) so RentalCore can refresh the alias cache periodically without restarts.

### **v3.37** - Split-Line Discount Support
- ✅ **Descriptor Pairing**: OCR now links multi-line item descriptions with their numeric rows, so products that list text on one line and the `Menge/Einzel/Rabatt/Gesamt` block on the next keep their discounts intact.
- 🛠️ **Fallback Parsing**: When the PDF jams quantity + unit together (e.g., `1Stück250,00`), the parser still splits out the percentage column before computing the discounted line total.

### **v3.36** - Line Discount Awareness
- ✅ **Column Parser Upgrade**: OCR now understands the `Rabatt %` column (even when exporters jam the numbers together), so per-product rebates persist as real totals instead of collapsing into faux line totals.
- 🆕 **Review Discount Column**: The extraction review table highlights the exact euro amount removed from each item, making it obvious which lines were discounted before mapping or job creation.

### **v3.35** - Forced Percent Discounts
- ✅ **Column-Aware Discounts**: Line-item discount columns are now always interpreted as percentages even when the PDF omits the `%` symbol, so individual product rebates no longer get misread as absolute totals.
- 🛠️ **Safer Overrides**: Percent math executes before trusting exported totals, preventing the discount cell from being mistaken for the final line total when vendors shuffle columns.

### **v3.34** - Discount Parsing Reliability
- ✅ **Per-Line Discounts Applied**: OCR now detects signed discount columns and forces recalculated device totals whenever the exported total still equals the pre-discount amount, so custom prices always inherit the actual quoted rates.
- 🛠️ **More Forgiving Regex**: Handles negative discount cells and stray percent markers, ensuring future PDF layouts keep lowering device prices without manual tweaks.

### **v3.33** - OCR Duplicate Safeguards & Discounts (Issue #30)
- ✅ **Duplicate Detection**: PDF mapping screen now warns when an identical job (same customer + product mix) already exists, with live re-checks and a public API hook for future automations.
- 🛠️ **Ready-to-Check Counts**: Server-side mapping summaries now feed the duplicate detector immediately, so alerts appear as soon as products are mapped without reloading the page.
- ✅ **Accurate Shortage Errors**: Device assignment failures mention the actual product name instead of opaque IDs, making shortages actionable.
- ✅ **Discount-Aware Pricing**: Line items that include per-row discounts now feed the discounted totals into custom device prices, so the resulting job reflects the real quoted rates.
- ✅ **Smarter Customer Auto-Match**: OCR names are normalized against company/legal suffixes and fuzzy-matched with a lower threshold, so existing customers auto-link without manual mapping.

### **v3.32** - Presence Messaging Polish
- ✅ **Clearer Alerts**: Editors now see “started to edit” when someone joins mid-session, keeping the order of events obvious.

### **v3.31** - Job Presence Schema Fix
- ✅ **Migration Compatibility**: Aligns the new `job_edit_sessions` table with existing `jobs`/`users` column types so PostgreSQL accepts the foreign keys without manual tweaks.
- ✅ **Init SQL Updated**: Fresh installations now ship with the corrected schema.
- ✅ **Applied in Production**: The running PostgreSQL instance is already migrated via `026_add_job_edit_sessions.up.sql`.

### **v3.30** - Live Job Editing Presence (Issue #21)
- ✅ **Real-Time Presence Warnings**: Users now see immediate slide-in alerts when someone else starts editing the same job, preventing conflicting updates.
- ✅ **Session Tracking**: Backed by a dedicated `job_edit_sessions` table with automatic expiry, so stale editors disappear within minutes.
- ✅ **Automatic Cleanup Hooks**: Edit modal opens start a heartbeat, and closing/refreshing the modal clears the session to keep signals accurate.

### **v3.29** - Job Attachment Delete Control (Issue #28)
- ✅ **Inline Delete Action**: Added dedicated delete buttons next to preview/download controls inside the Job Details modal attachments list.
- ✅ **Two-Way Sync**: When a file is removed, the view-only modal and edit modal attachment lists (plus counters) refresh automatically to stay in sync.
- ✅ **Safer Preview Handling**: Any open preview closes automatically if its backing file is deleted, preventing stale downloads or 404s.

### **v2.83** - PDF Invoice/Offer Processing (Phase 1 - Issue #24)
- ✅ **PDF Upload Interface**: New "Upload PDF" button in job creation form
- ✅ **Database Schema**: 4 new tables (pdf_uploads, pdf_extractions, pdf_extraction_items, pdf_product_mappings)
- ✅ **Text Extraction**: Automated PDF text parsing with regex-based data extraction
- ✅ **Product Mapping**: Fuzzy string matching with Levenshtein distance algorithm
- ✅ **Smart Suggestions**: Automatic product mapping with confidence scores (70%+ threshold)
- ✅ **Data Parser**: Extracts customer names, invoice numbers, dates, amounts, discounts, line items
- ✅ **Mapping Persistence**: Saved mappings improve accuracy over time
- ✅ **5 API Endpoints**:
  - POST /api/pdf/upload - Upload and process PDFs
  - GET /api/pdf/extraction/:id - Retrieve extraction results
  - POST /api/pdf/mapping - Save product mappings
  - GET /api/pdf/suggestions - Get mapping suggestions
  - PUT /api/pdf/items/:id/mapping - Update item mappings
- ✅ **Async Processing**: Background processing with status polling
- ✅ **Modal UI**: Clean upload interface with extraction preview
- 📝 **Phase 2 Planned**: Full PDF library integration, review UI, auto-population

### **v2.82** - Fixed Revenue Calculation Logic
- ✅ **Correct Price Structure**: Fixed revenue calculation to use proper nested device structure
- ✅ **Custom Price Support**: Properly checks custom_price before falling back to product price
- ✅ **Multi-Property Handling**: Handles both itemcostperday and itemCostPerDay properties
- ✅ **Debug Logging**: Added console logging to trace price calculations
- ✅ **Accurate Totals**: Revenue now correctly sums all device prices (e.g., 30.00 instead of 0.00)

### **v2.81** - Live Revenue Recalculation
- ✅ **Real-Time Revenue Updates**: Revenue automatically recalculates when devices are removed
- ✅ **Price Change Sync**: Revenue updates instantly when device prices are edited
- ✅ **Auto-Summation**: Automatically sums all device prices for accurate totals
- ✅ **Form Integration**: Updates revenue input field in real-time during editing
- ✅ **Verified Styling**: Confirmed Edit and New Job modals have identical product tree designs

### **v2.80** - Smart Device Removal with Product Tree Sync
- ✅ **Auto-Decrement Quantity**: Removing assigned devices decreases product tree quantity automatically
- ✅ **Prevents Re-Assignment**: No more accidental device re-adds when removing from list
- ✅ **Product Tree Sync**: Input fields update in real-time when devices removed
- ✅ **Helper Function**: Added product finder to traverse tree structure efficiently
- ✅ **Consistent Design**: Both Edit and New Job modals share identical product tree styling

### **v2.79** - Right-Aligned Input Controls
- ✅ **Improved Layout**: Input and availability now right-aligned next to product name
- ✅ **Better Visual Flow**: Product name on left, controls grouped on right
- ✅ **Cleaner Spacing**: Natural gap between product name and input controls
- ✅ **Consistent Alignment**: Same layout across modal and standalone page
- ✅ **Layout**: `[Icon] Product Name          [Input] Available: X/Y`

### **v2.78** - Consistent Modal Widths
- ✅ **Unified Modal Size**: Edit Job modal now matches New Job modal width (1200px)
- ✅ **Better Product Tree View**: Wider modal provides more space for horizontal product layout
- ✅ **Consistent UX**: Same visual experience across create and edit workflows
- ✅ **Improved Scrolling**: Added max-height and overflow-y properties for long forms

### **v2.77** - Single-Line Horizontal Product Layout
- ✅ **One-Line Design**: Product name, quantity input, and availability all on one horizontal line
- ✅ **Cleaner Interface**: Removed progress bars and quantity labels for minimalist look
- ✅ **Compact Input**: Reduced input field to 80px width
- ✅ **Smart Truncation**: Long product names truncate with ellipsis (...)
- ✅ **Better Scanning**: More items visible at once with horizontal layout
- ✅ **Responsive Mobile**: Wraps to multi-line on small screens
- ✅ **Consistent Styling**: Same layout in both modal and standalone page

### **v2.76** - Slim & Compact Product Tree Design
- ✅ **Reduced Entry Height**: Slimmer product entries with compact padding (40% less vertical space)
- ✅ **Tighter Spacing**: Minimal gaps between items for denser layout
- ✅ **Smaller Typography**: Optimized font sizes for better information density
- ✅ **Compact Controls**: Reduced input field and progress bar sizes
- ✅ **Horizontal Layout**: More width, less height for better scrolling experience
- ✅ **Consistent Styling**: Same slim design in both modal and standalone page

### **v2.75** - Product Tree Modal Styling Fix
- ✅ **Modal Styling Fixed**: Applied premium product tree styling to Edit Job modal
- ✅ **Complete Coverage**: Both standalone page and modal now share identical premium styling
- ✅ **Gradient Backgrounds**: Enhanced depth with sophisticated gradient effects
- ✅ **Smooth Animations**: Consistent transitions and hover effects across all views
- ✅ **Visual Consistency**: Unified design language between job form and modal

### **v2.74** - Premium Product Tree Styling
- ✅ **Premium Dark Theme**: Redesigned product tree with sophisticated gradient backgrounds and enhanced depth
- ✅ **Enhanced Visual Hierarchy**: Improved spacing, typography, and layout for better readability
- ✅ **Smooth Animations**: Added smooth transitions and hover effects throughout the product tree
- ✅ **Modern UI Elements**: Redesigned chips/badges with gradient backgrounds and better color coding
- ✅ **Interactive Product Cards**: Premium card design with left-side accent bars and lift animations
- ✅ **Enhanced Progress Bars**: Glowing effect on progress indicators with better visual feedback
- ✅ **Better Touch Targets**: Larger, more accessible interactive elements with improved hover states
- ✅ **Responsive Design**: Mobile-optimized layout with full responsive breakpoints

### **v2.69** - Enhanced Device Availability Error Logging
- ✅ **Detailed Error Messages**: Enhanced "not enough available devices" error with comprehensive diagnostics
- ✅ **Availability Breakdown**: Shows total needed, available (cases/loose), already kept, and remaining
- ✅ **Debugging Aid**: Helps identify quantity multiplication issues and availability calculation problems
- ✅ **Production Diagnostics**: Better error reporting for troubleshooting device assignment failures
- ✅ **Issue Resolution**: Provides actionable data for fixing availability check logic

### **v2.68** - Complete Modal System Rewrite
- ✅ **ARCHITECTURAL REWRITE**: Complete rebuild of modal system from scratch (~220 lines removed, replaced with clean implementation)
- ✅ **Flexbox-Based Centering**: New system uses simple, reliable Flexbox (display: flex, align-items: center, justify-content: center)
- ✅ **NO MORE Grid Issues**: Eliminated all grid/place-items complications that caused centering problems
- ✅ **NO MORE Width Conflicts**: Removed all fixed widths, only max-width for responsive sizing
- ✅ **Clean Architecture**: Reduced from 4044 to 3985 lines of CSS (59 lines less, more maintainable)
- ✅ **Modern Animations**: Clean fade-in with scale effect, smooth transitions
- ✅ **Size Variants**: .rc-modal-small, .rc-modal-medium, .rc-modal-large, .rc-modal-xl support
- ✅ **Responsive Design**: Built-in mobile/tablet/desktop breakpoints
- ✅ **Guaranteed Centering**: Perfect centering works 100% of the time, all scenarios
- ✅ **Zero Template Changes**: Works with existing HTML structure, backward compatible

### **v2.67** - Modal Centering Root Fix
- ✅ **Root Cause Fix**: Removed explicit `width` property from `.rc-modal-content` class that prevented proper CSS grid centering
- ✅ **All Modals**: Fixed centering issue across all modal types (Job Details, New Job, Equipment Packages, File Preview, etc.)
- ✅ **Flexible Sizing**: Modals now use only `max-width` for responsive sizing while maintaining proper centering
- ✅ **Clean CSS**: Simplified modal styling by removing conflicting width constraints
- ✅ **Grid Compatibility**: `place-items: center` now works correctly without width interference

### **v2.66** - Modal Centering Root Fix
- ✅ **Root Cause Fix**: Removed explicit `width` property from `.rc-modal-content` class that prevented proper CSS grid centering
- ✅ **All Modals**: Fixed centering issue across all modal types (Job Details, New Job, Equipment Packages, File Preview, etc.)
- ✅ **Flexible Sizing**: Modals now use only `max-width` for responsive sizing while maintaining proper centering
- ✅ **Clean CSS**: Simplified modal styling by removing conflicting width constraints
- ✅ **Grid Compatibility**: `place-items: center` now works correctly without width interference

### **v2.65** - Modal Centering Fix
- ✅ **Improved Modal Centering**: Removed explicit width values that interfered with CSS grid centering
- ✅ **Jobs Modal**: Fixed Job Details, New Job, and File Preview modals to center properly
- ✅ **Equipment Packages**: Fixed Package Detail and Edit Package modals centering
- ✅ **Responsive Design**: All modals now use only max-width for better responsiveness
- ✅ **Consistent Behavior**: Grid-based centering now works correctly across all modal types

### **v2.63** - Theme-Aligned Product Tree
- ✅ **Unified Look & Feel**: Reworked product tree containers, chips, and cards to reuse RentalCore's surface palette and spacing scale
- ✅ **Balanced Shadows**: Softened elevation and hover states so the modal matches other RC cards and tables
- ✅ **Readable Summaries**: Updated category headers and status chips for better contrast on light/dark modes
- ✅ **Consistent Controls**: Toolbar actions, progress indicators, and quantity inputs now inherit system focus/hover treatments
- ✅ **Pure Typography**: All labels, chips, and product details render in crisp white to match the dark UI across RentalCore/WarehouseCore

### **v2.62** - Product Tree Enhancements
- ✅ **Live Availability Summary**: Category toolbar aggregates totals (categories, products, free devices, utilisation %)
- ✅ **One-Click Tree Controls**: Added “Expand All” / “Collapse All” actions that toggle every node instantly
- ✅ **Contextual Product Cards**: Each product renders with brand/manufacturer subtitles plus selection/state badges
- ✅ **Availability Thermometer**: Gradient progress bar and warning states flag overbooked requests in real time
- ✅ **Accessibility Tweaks**: Inputs ship with descriptive aria labels and improved keyboard focus handling

### **v2.60** - Enhanced Product Tree Styling
- ✅ **Collapsed by Default**: All categories/subcategories start collapsed for cleaner view
- ✅ **Animated Icons**: Arrow icons (▶) rotate smoothly when expanding/collapsing
- ✅ **Modern Design**: Gradient backgrounds, smooth transitions, hover effects
- ✅ **Slide-Down Animation**: Content smoothly animates when expanding
- ✅ **Product Icons**: Each product entry has emoji icon (📦) for visual clarity
- ✅ **Enhanced Inputs**: Better styled input fields with focus states and hover effects
- ✅ **Visual Hierarchy**: Clear indentation and sizing for category levels
- ✅ **Professional UX**: Polished, modern appearance with attention to detail

### **v2.59** - Property Name Case Fix
- ✅ **Critical Fix**: Changed JavaScript property access from PascalCase to snake_case
- ✅ **Root Cause**: Backend JSON uses snake_case (id, name, available_count) but JS used PascalCase (ID, Name, AvailableCount)
- ✅ **Fixed Properties**: All category, subcategory, and product property references
- ✅ **Complete Fix**: Category names, products, and availability counts now display correctly
- ✅ **Issue #23 FULLY RESOLVED**: Product tree fully functional in modal

### **v2.58** - CSS Extraction Fix for Modal
- ✅ **Critical Fix**: Extract and inject `<style>` tags from fetched HTML
- ✅ **Root Cause**: CSS was in `<head>`, not included when extracting `<main>` content
- ✅ **Solution**: Append all style tags to document.head for modal context
- ✅ **Complete Solution**: Both scripts AND styles now properly injected
- ✅ **Product Tree Display**: Now fully styled and visible in modal

### **v2.57** - Debug Logging for Product Tree
- ✅ **Debug Enhancement**: Added comprehensive logging for product tree rendering
- ✅ **API Response Logging**: Shows API response data and parsed tree structure
- ✅ **Element Detection**: Logs productTreeRoot element availability
- ✅ **Render Status**: Logs categories count and rendering progress
- ✅ **Troubleshooting Aid**: Helps diagnose display issues with product tree

### **v2.56** - Script Extraction Fix
- ✅ **Critical Bug Fix**: Fixed script extraction from fetched HTML in modal
- ✅ **Root Cause**: Scripts were in `<head>` and end of `<body>`, not inside `<main>`
- ✅ **Solution**: Changed `content.querySelectorAll('script')` to `doc.querySelectorAll('script')`
- ✅ **Complete Fix**: All scripts from fetched page now properly execute in modal
- ✅ **Debug Logging**: Added script count logging for troubleshooting
- ✅ **Issue #23 Resolved**: Product tree now loads correctly in modal job creation

### **v2.55** - Window Exposure Timing Fix
- ✅ **Function Exposure**: Moved `window.initializeJobForm` exposure to immediately after function definition
- ✅ **Auto-Init Removed**: Removed problematic auto-init script that ran too early
- ✅ **Debug Enhancement**: Added console logs to track function availability
- ✅ **Execution Order**: Fixed timing issue where function wasn't available when modal called it

### **v2.54** - Script Execution Enhancement
- ✅ **Script Injection**: Scripts now appended to `document.body` for guaranteed execution
- ✅ **Promise Handling**: Added Promise.all for external script loading
- ✅ **Retry Mechanism**: Added interval checking for function availability (5 second timeout)
- ✅ **Error Reporting**: Enhanced error messages showing available window functions

### **v2.53** - Dual Init Mechanism
- ✅ **Auto-Init Script**: Added self-initializing script at end of job_form.html
- ✅ **Modal Detection**: Automatically detects modal context (no navbar)
- ✅ **Retry Logic**: Two retry mechanisms (auto-init + modal interval check)
- ✅ **Debug Logs**: Enhanced logging with emoji indicators
- ✅ **Guaranteed Init**: Product tree initialization now foolproof

### **v2.52** - Critical Product Tree Fix
- ✅ **Bug Fix**: Product tree now loads correctly in modal
- ✅ **Initialization**: Manual form initialization after AJAX content load
- ✅ **DOMContentLoaded**: Fixed event not firing for dynamically loaded content
- ✅ **Console Logging**: Added debug logs for initialization tracking
- ✅ **Both Contexts**: Works in both standalone page and modal popup

### **v2.51** - Modal-Based Job Creation
- ✅ **Popup Interface**: "New Job" button opens modal popup instead of navigating to new page
- ✅ **Streamlined UX**: Create jobs without leaving the jobs overview page
- ✅ **Dynamic Loading**: Modal fetches and displays job form via AJAX
- ✅ **Smart Content Extraction**: Automatically extracts <main> content for clean modal display
- ✅ **Fallback Support**: Direct link to /jobs/new available if modal fails to load

### **v2.50** - Product Tree Always Visible
- ✅ **Immediate Loading**: Product tree loads immediately on page open (no date requirement)
- ✅ **Always Visible**: All products and categories shown by default with total device counts
- ✅ **Dynamic Availability**: Availability counts update in real-time when dates are selected
- ✅ **Improved UX**: Users can browse products before selecting dates
- ✅ **API Enhancement**: Backend supports requests with or without date parameters

### **v2.49** - Issue #23: Product Assignment UI Fix
- ✅ **UI Layout Corrected**: Product availability now displays BELOW input field (not beside it)
- ✅ **Grid Layout Fixed**: Changed from 3-column to 2-column layout for product entries
- ✅ **Exact Requirement Match**: Layout now matches Issue #23 specification: "darunter steht die verfügbare anzahl"
- ✅ **All Features Verified**: Product tree, quantity input, validation, and smart case allocation all working
- ✅ **Documentation Added**: Complete resolution documentation in ISSUE_23_RESOLUTION.md

### **v2.48** - Product-First Job Builder & API Hardening
- ✅ **Unified Job Workflow**: Both create and edit forms use the new product availability tree with quantity selection and validation.
- ✅ **Legacy Modal Removed**: Old device-tree modals, quick-add flows, and `selected_devices` payloads retired for a cleaner UI.
- ✅ **Backend Alignment**: Job API now only accepts `selected_products` payloads; helper tests cover parsing and aggregation edge-cases.
- ✅ **Device Write Guard**: All POST/PUT/DELETE device endpoints return `410 Gone`, directing operators to WarehouseCore.
- ✅ **Documentation Refresh**: README and API docs outline the read-only scope for device operations in RentalCore.

### **v2.45** - Phase 5: Case Management Redirect to WarehouseCore
- ✅ **Case CRUD Redirected**: All case management operations now redirect to WarehouseCore
- ✅ **Web Routes Updated**: Case list, new form, edit, and device mapping routes redirect to `/admin/cases` in WarehouseCore
- ✅ **API Routes Removed**: All case API operations (POST, PUT, DELETE) removed from RentalCore
- ✅ **Read-Only Endpoint Preserved**: Case GET endpoint maintained for Jobs integration (if needed)
- ✅ **Navigation Updated**: Added "Cases (WH)" indicator in navbar and base template showing redirect to WarehouseCore
- ✅ **Templates Removed**: Deleted all 4 case template files (cases_list.html, case_form.html, case_detail.html, case_device_mapping.html)
- ✅ **Redirect Function**: Added buildWarehouseCasesURL for automatic WarehouseCore routing
- ✅ **Tests Added**: Comprehensive redirect tests for case URL building with env/fallback scenarios
- ✅ **Documentation Updated**: README reflects case management now exclusively in WarehouseCore
- ✅ **Complete Migration**: Phase 5 migration to WarehouseCore completed successfully

### **v2.44** - Phase 4: Cable Management Redirect to WarehouseCore
- ✅ **Cable CRUD Redirected**: All cable management operations now redirect to WarehouseCore
- ✅ **Web Routes Updated**: Cable list and new form routes redirect to `/admin/cables` in WarehouseCore
- ✅ **API Routes Removed**: All cable API write operations (POST, PUT, DELETE) removed from RentalCore
- ✅ **Navigation Updated**: Added "Cables (WH)" indicator in navbar showing redirect to WarehouseCore
- ✅ **Templates Removed**: Deleted cables_standalone.html and cable_form.html
- ✅ **Redirect Function**: Added buildWarehouseCablesURL for automatic WarehouseCore routing
- ✅ **Tests Added**: Comprehensive redirect tests for cable URL building with env/fallback scenarios
- ✅ **Documentation Updated**: README reflects cable management now exclusively in WarehouseCore
- ✅ **Read-Only APIs Removed**: Cable APIs removed as they are not required by Jobs/Invoices

### **v2.39** - Phase 2: Device Management Redirect to WarehouseCore
- ✅ **Device CRUD Redirected**: All device management operations now redirect to WarehouseCore
- ✅ **Read-Only APIs Preserved**: Device GET endpoints remain for Jobs/Invoices integration
- ✅ **Navigation Updated**: Added "Devices (WH)" cross-link to WarehouseCore in sidebar
- ✅ **Templates Removed**: Deleted device_form.html, device_detail.html, devices_standalone.html
- ✅ **Redirect Function**: Added buildWarehouseDevicesURL for automatic WarehouseCore routing
- ✅ **Tests Added**: Comprehensive redirect tests for device URL building
- ✅ **Documentation Updated**: README reflects device management now in WarehouseCore
- ✅ **Write Operations Removed**: POST/PUT/DELETE device routes removed from RentalCore

### **v2.38** - Clickable Job Rows
- ✅ **Entire Row Clickable**: Click anywhere on a job row to open job details
- ✅ **Better UX**: No need to find and click the specific eye icon
- ✅ **Visual Feedback**: Cursor changes to pointer when hovering over job rows
- ✅ **Action Buttons Protected**: Edit and Delete buttons still work independently
- ✅ **Event Handling**: Actions column uses stopPropagation to prevent double-triggers

### **v2.39** - Complete Scanner Removal
- ✅ **Scanner Code Deleted**: Removed all scanner template files (7 files)
- ✅ **Handler Removed**: Deleted scanner_handler.go completely
- ✅ **WASM Directory Deleted**: Removed entire web/scanner/ directory with Go WASM implementation
- ✅ **Routes Cleaned**: Removed all scanner routes from main.go (web, API, mobile)
- ✅ **Navigation Updated**: Removed scanner link from navbar
- ✅ **Config Deprecated**: Scanner config field marked deprecated, always returns false
- ✅ **Documentation Updated**: Removed scanner architecture and feature documentation
- ✅ **Templates Cleaned**: Updated job_detail.html to remove scanner references
- ✅ **Complete Removal**: Scanner functionality permanently disabled and removed from codebase

### **v2.37** - Scanner UI Disabled
- ✅ **Scanner UI Removed**: All scan buttons and links removed from UI
- ✅ **Home Page Updated**: Replaced "Start Scanning" with "Manage Jobs" button
- ✅ **Quick Actions Cleaned**: Removed "Scan QR Code" from quick actions menu
- ✅ **Job Detail Page**: Removed scan device buttons from job detail view
- ✅ **Jobs Table Updated**: Device count now displayed as plain text (no scan redirect)
- ✅ **API Endpoint Fixed**: Changed remove device endpoint from `/scan/` to `/jobs/`
- ✅ **Routes Protected**: Scan routes only registered when scanner_enabled=true
- ✅ **Clean UI**: No scan-related functionality visible to users

### **v2.36** - New Job Modal Device Tree Fix
- ✅ **New Job Modal Fixed**: Device tree now loads automatically when creating new job
- ✅ **Dynamic Form Loading**: Fixed DOMContentLoaded issue with dynamically injected forms
- ✅ **Manual Initialization**: Added manual trigger for tree loading after form injection
- ✅ **Fallback Strategy**: Programmatic refresh button click as additional safeguard
- ✅ **Complete Auto-Load**: All scenarios now covered (standalone, edit, new job)
- ✅ **Debug Logging**: Added console logging for troubleshooting modal injection

### **v2.35** - Edit Modal Device Tree Fix
- ✅ **Edit Modal Fixed**: Device tree now loads automatically in edit job modal
- ✅ **Removed Date Check**: No longer checks for dates before loading tree
- ✅ **Immediate Loading**: Tree appears instantly when opening edit modal
- ✅ **Consistent Behavior**: Same auto-load behavior as job creation form
- ✅ **Complete Coverage**: All modals now auto-load device tree

### **v2.34** - Device Tree Backend Fix
- ✅ **Backend API Updated**: Dates now optional in `/api/v1/devices/tree/availability`
- ✅ **Fixed Date Requirement**: Backend no longer returns error when dates missing
- ✅ **All Devices Available**: Without dates, all devices shown as available
- ✅ **Date Filtering Works**: When dates provided, proper availability checking
- ✅ **Complete Fix**: Both frontend and backend now support date-optional loading
- ✅ **Job Creation Works**: Device tree now truly loads on new job creation

### **v2.33** - Device Tree Auto-Load
- ✅ **Always Load Device Tree**: Removes date dependency for loading device tree
- ✅ **Automatic Loading**: Tree loads immediately on page load/modal open
- ✅ **Optional Date Filtering**: If dates provided, they filter availability
- ✅ **All Devices Available**: Without dates, all devices shown as available
- ✅ **Better UX**: No more waiting or clicking to load device tree
- ✅ **New Job Creation**: Tree now loads automatically when creating jobs

### **v2.32** - UI Theme Consistency
- ✅ **Global Scrollbar Styling**: Consistent dark theme scrollbars across all elements
- ✅ **Dropdown Options**: Select options now have dark background matching theme
- ✅ **WebKit Scrollbars**: Styled for Chrome, Safari, Edge with gray thumb
- ✅ **Firefox Scrollbars**: Thin scrollbar with matching colors
- ✅ **Hover States**: Accent color on hover/checked states for dropdowns
- ✅ **Better UX**: All UI elements now follow RentalCore dark theme

### **v2.31** - Quick Add by Product/Quantity
- ✅ **Quick Add Interface** (Issue #7): New Quick Add section in job edit modal
- ✅ **Product Dropdown**: Auto-populated with all available products and device counts
- ✅ **Quantity Selection**: Specify how many devices to add in one click
- ✅ **Smart Auto-Selection**: System automatically selects requested number of available devices
- ✅ **Availability Validation**: Shows warnings if not enough devices available
- ✅ **Duplicate Prevention**: Skips already-selected devices
- ✅ **Real-time Feedback**: Success/warning messages with auto-dismiss
- ✅ **Auto-Reset**: Form resets after successful add for quick repeated use
- ✅ **UX Enhancement**: Dramatically speeds up device assignment workflow

### **v2.30** - Multiple GitLab Issues Resolved
- ✅ **Scanner Disabled** (Issue #12): Scanner functionality disabled via config (`SCANNER_ENABLED=false`)
- ✅ **Uncategorized Devices** (Issue #11): Devices without product/category now appear in "Uncategorized Devices" section
- ✅ **Job Title Fix** (Issue #10): Job detail page now shows "Job #ID" instead of description as main heading
- ✅ **Category Filter** (Issue #9): Added job category filter dropdown in jobs table
- ✅ **Tree Collapse Fix** (Issue #8): Device tree no longer auto-collapses when selecting items
- ✅ **SQL Improvements**: Added category_name to job list query with proper GROUP BY
- ✅ **DOM Optimization**: Efficient visual state updates without full tree re-rendering
- ✅ **UI/UX Enhancements**: Better navigation and filtering experience

### **v2.29** - Job Title Display Update
- ✅ **JobCode Display**: Show JobCode (e.g., JOB000123) instead of description in jobs table
- ✅ **Cleaner UI**: Title column now shows structured job codes
- ✅ **Consistent Formatting**: Job codes displayed with monospace font
- ✅ **Better Identification**: Easier to reference specific jobs by code
- ✅ **Fixes GitLab Issue #10**: Description removed from job title display

### **v2.28** - Device Availability Check Fix
- ✅ **Nil Pointer Fix**: Fixed panic in IsDeviceAvailableForJob when checking device availability
- ✅ **Proper Error Handling**: Added nil check before calling err.Error()
- ✅ **Stability Improvement**: Prevents crashes during device assignment checks
- ✅ **Database Query Safety**: Ensures graceful handling of "not found" cases
- ✅ **Fixes GitLab Issue #11**: Device availability check no longer crashes

### **v2.27** - Scanner Feature Toggle
- ✅ **Feature Flag System**: New `SCANNER_ENABLED` environment variable
- ✅ **Configurable Scanner**: Scanner functionality can now be disabled without removing code
- ✅ **UI Integration**: Scanner nav link automatically hidden when disabled
- ✅ **Route Protection**: Scanner routes conditionally registered based on config
- ✅ **API Protection**: Scanner API endpoints conditionally enabled
- ✅ **Backwards Compatible**: Scanner enabled by default (set `SCANNER_ENABLED=false` to disable)
- ✅ **Clean Architecture**: Code remains in place, only activation controlled
- ✅ **Fixes GitLab Issue #12**: Scanner function can be disabled via configuration

### **v2.26** - Persistent Tree Expand State
- ✅ **Expand State Tracking**: Tree nodes remember their expanded/collapsed state
- ✅ **No Auto-Collapse**: Selecting a device no longer collapses the tree
- ✅ **Smart State Restoration**: Expanded nodes stay open when tree re-renders
- ✅ **All Levels**: Works for categories, subcategories, and product groups
- ✅ **Visual Feedback**: Chevron icons reflect current state
- ✅ **Improved UX**: Users can select multiple devices without re-expanding
- ✅ **Fixes GitLab Issue #8**: Tree view auto-collapse prevention

### **v2.25** - Smart Product/Amount Selection
- ✅ **Quick Add Button**: New "Quick Add by Product" button in device selection
- ✅ **Product Dropdown**: Auto-populated with all available products from device tree
- ✅ **Quantity Input**: Specify how many devices to add
- ✅ **Smart Auto-Selection**: System automatically selects available devices
- ✅ **Availability Check**: Shows warning if not enough devices available
- ✅ **Duplicate Prevention**: Skips already selected devices
- ✅ **Status Feedback**: Real-time feedback on success/warnings
- ✅ **Auto-Close Modal**: Modal closes automatically after successful add
- ✅ **Date Validation**: Requires start/end dates before opening
- ✅ **Fixes GitLab Issue #7**: Smart product/amount selection with auto-assignment

### **v2.24** - Jobs Table Filtering and Sorting
- ✅ **Column Sorting**: Click column headers to sort ascending/descending
- ✅ **Sort Indicators**: Dynamic arrow icons show current sort state
- ✅ **Customer Filter**: Real-time text search for customer names
- ✅ **Status Filter**: Dropdown to filter by job status
- ✅ **Device Count Filter**: Minimum device count filter
- ✅ **Clear Filters**: One-click button to reset all filters
- ✅ **Smart Sorting**: Handles numbers, dates, and text correctly
- ✅ **Visual Feedback**: Active sort column highlighted
- ✅ **Fixes GitLab Issue #6**: Complete table filtering and sorting

### **v2.23** - Device Tree View Button Fix
- ✅ **Tree View Button Works**: Fixed unresponsive "Tree View" button in job creation form
- ✅ **Event Handler Added**: Added click event handler for toggleTreeView button
- ✅ **Smart Date Validation**: Prompts user to select dates before loading device tree
- ✅ **Toggle Functionality**: Button now properly shows/hides device tree
- ✅ **User-Friendly**: Clear alert message when dates are missing
- ✅ **Fixes GitLab Issue #5**: Device selection tree button now fully functional

### **v2.22** - Job Creation Fix
- ✅ **Fixed HTTP 500 Error**: Fixed job creation failure caused by missing job_code
- ✅ **BeforeCreate Hook**: Added BeforeCreate hook to set temporary job_code before INSERT
- ✅ **Database Constraint Fix**: Satisfies NOT NULL constraint on job_code column
- ✅ **Automatic Job Codes**: Job codes still generated as JOB000123 format after insert
- ✅ **Production Ready**: Job creation now works reliably without errors

### **v2.14** - Improved Dropdown Auto-Scroll
- ✅ **Complete Visibility**: Last dropdown item now fully visible without manual scrolling
- ✅ **Precise Targeting**: Scrolls directly to last dropdown item instead of container
- ✅ **Enhanced Timing**: Increased delay to 150ms for complete CSS transition
- ✅ **Better Detection**: More accurate visibility detection for last item
- ✅ **Perfect Scrolling**: No more partial visibility of dropdown items
- ✅ **Works Flawlessly**: Tools, Products, and all future dropdowns fully supported

### **v2.13** - Auto-Scroll for Sidebar Dropdowns
- ✅ **Automatic Scrolling**: Sidebar auto-scrolls when dropdowns are opened
- ✅ **Smart Detection**: Only scrolls if dropdown extends below visible area
- ✅ **Smooth Animation**: Uses smooth scrolling behavior for better UX
- ✅ **Intelligent Positioning**: Positions dropdown at bottom of viewport
- ✅ **No Manual Scrolling**: Users don't need to scroll manually to see dropdown items
- ✅ **Works for All Dropdowns**: Products, Tools, and future dropdowns supported
- ✅ **Preserves State**: Scroll position maintained when closing dropdown

### **v2.12** - Unified Sidebar Font Sizes
- ✅ **Consistent Text Sizing**: All sidebar navigation items now have uniform font size
- ✅ **Fixed Dropdown Toggle Size**: Dropdown buttons (Products, Tools) match regular nav items
- ✅ **Explicit Font Declaration**: Added font-size: 1rem to all navigation elements
- ✅ **Button Inheritance Fix**: Added font-family: inherit to ensure proper font rendering
- ✅ **Professional Appearance**: Eliminated visual inconsistency in navigation

### **v2.11** - Sidebar Icon Vertical Centering
- ✅ **Perfectly Centered Icons**: All sidebar navigation icons now vertically centered
- ✅ **Flexbox Centering**: Applied display: flex with align-items and justify-content
- ✅ **Fixed All Icon Types**: Main nav, dropdowns, user menu, and collapsed icons
- ✅ **Consistent Alignment**: Icons no longer appear bottom-aligned
- ✅ **Visual Polish**: Professional appearance across all navigation elements

### **v2.10** - Simplified Header Bar
- ✅ **Removed Company Name**: Cleaner header with only RentalCore branding
- ✅ **Focused Design**: Simplified header layout for better visual hierarchy
- ✅ **Maintained Functionality**: All toggle buttons and branding remain intact

### **v2.9** - User Profile Dropdown Menu
- ✅ **Click-Triggered User Menu**: Click on username to open settings and logout dropdown
- ✅ **Consolidated User Actions**: All settings and logout in one dropdown menu
- ✅ **Removed Email Display**: Cleaner footer with just username (no email)
- ✅ **Upward Opening Dropdown**: Menu opens upward from footer with smooth animations
- ✅ **5 Menu Items**: Profile, Company, Users, Security & Audit, Logout
- ✅ **Divider Before Logout**: Visual separation with logout at bottom
- ✅ **Red Logout Accent**: Logout item has red hover effect for visual distinction
- ✅ **Click Outside to Close**: Dropdown closes when clicking anywhere outside
- ✅ **Chevron Animation**: Icon rotates when opening/closing (up ↔ down)
- ✅ **Active State Highlighting**: Current page highlighted in dropdown
- ✅ **Smooth Transitions**: 0.3s fade and height transitions
- ✅ **Collapsed Sidebar Support**: Icon-only view maintained when sidebar collapsed

### **v2.8** - Content Positioning Fix
- ✅ **Fixed Content Behind Sidebar/Header**: Content now displays in available space, not behind fixed elements
- ✅ **Body-Level Padding**: Applied padding-top for header and padding-left for sidebar to body element
- ✅ **Works With All Templates**: Fixes positioning for all 60+ templates using navbar.html
- ✅ **No Wrapper Required**: Content positioned correctly without needing .rc-main-content wrapper
- ✅ **Smooth Transitions**: Added transitions for padding/margin changes when toggling sidebar
- ✅ **Responsive Spacing**: Header clearance (3.5rem mobile, 4rem desktop)
- ✅ **Sidebar Offset**: 16rem for open sidebar, 5rem for collapsed, 0 on mobile
- ✅ **Template Compatibility**: Works with direct <main> tags after {{template "navbar.html" .}}
- ✅ **Proper Boundaries**: Header and sidebar correctly define content area boundaries

### **v2.7** - Sidebar Layout Fixes
- ✅ **Scrollable Sidebar Navigation**: Sidebar nav now scrolls vertically with custom scrollbar
- ✅ **Fixed Footer Overlap**: User profile section no longer overlaps navigation items
- ✅ **Proper Main Content Layout**: Main content correctly positioned below header and beside sidebar
- ✅ **Flexbox Sidebar Structure**: Header (fixed) + Navigation (scrollable) + Footer (fixed)
- ✅ **Body Class Management**: JavaScript sets `sidebar-open`/`sidebar-collapsed` classes on body
- ✅ **Custom Scrollbar**: 6px width with subtle white thumb for sidebar navigation
- ✅ **Improved Footer Background**: Enhanced backdrop blur and opacity for footer
- ✅ **Responsive Breakpoints**: Fixed mobile/desktop transitions at 767px/768px
- ✅ **Min-height Content**: Main content fills viewport height properly
- ✅ **No More :has() Selectors**: Replaced with body class approach for better browser support

### **v2.6** - Sidebar Navigation with Dropdown Groups
- ✅ **Dropdown Navigation**: Organized navigation items into collapsible groups
- ✅ **Products Dropdown**: Own Products, Rental Equipment, Rental Analytics
- ✅ **Tools Dropdown**: Equipment Packages, Invoices, Analytics, Financial
- ✅ **Settings Dropdown**: Profile, Company, Users, Security & Audit
- ✅ **Smooth Animations**: 0.3s transitions for dropdown expand/collapse
- ✅ **Auto-expand Logic**: Dropdowns open if current page is inside
- ✅ **Fixes Overlap**: Resolved navigation items overlapping issue
- ✅ **Indented Items**: Visual hierarchy with 2.5rem left padding
- ✅ **Chevron Icons**: Dynamic chevron rotation (down/up) for visual feedback
- ✅ **Active State Handling**: Proper highlighting for active pages in dropdowns
- ✅ **Responsive Behavior**: Dropdowns hidden when sidebar collapsed on desktop

### **v2.5** - Sidebar Navigation Complete (All Pages)
- ✅ **Fixed Sidebar on All Pages**: Replaced navbar.html (used by 60+ templates) with sidebar navigation
- ✅ **Universal Navigation**: Sidebar now appears on Dashboard, Jobs, Devices, and all other pages
- ✅ **Template Architecture Fix**: Discovered that most templates use navbar.html, not base.html
- ✅ **Mobile Navigation Working**: Hamburger menu and overlay drawer functional on mobile
- ✅ **Desktop Collapse Working**: Chevron toggle for expanding/collapsing sidebar
- ✅ **JavaScript Integration**: Sidebar toggle script embedded in navbar.html
- ✅ **Consistent UX**: All 67 templates now have unified sidebar navigation
- ✅ **Storage Cross-Link**: Dynamic WarehouseCore URL detection for seamless switching

### **v2.4** - Left Sidebar Navigation Redesign (base.html only)
- ✅ **Professional Left Sidebar**: Implemented WarehouseCore-style left sidebar navigation
- ✅ **Collapsible Sidebar**: Desktop sidebar collapses to icon-only mode (w-64 expanded, w-20 collapsed)
- ✅ **Mobile Overlay Drawer**: Mobile devices use overlay drawer with backdrop and hamburger menu
- ✅ **Chevron Toggle Icons**: Desktop shows chevron-left (collapse) and chevron-right (expand) indicators
- ✅ **Responsive Header**: Header position adjusts dynamically based on sidebar state
- ✅ **Dynamic Content Margins**: Main content adapts margins automatically (16rem, 5rem, or 0)
- ✅ **Cross-Navigation**: Integrated WarehouseCore cross-link in sidebar for seamless app switching
- ✅ **User Profile Section**: User info with email displayed in sidebar footer
- ✅ **Settings Menu**: Profile, Company, Users, and Security links in sidebar footer
- ✅ **Glassmorphism Design**: Modern glass-effect design matching overall aesthetic
- ✅ **Smooth Transitions**: 300ms ease transitions for all sidebar state changes
- ✅ **Touch-Optimized**: 44px minimum tap targets for mobile usability
- ✅ **LocalStorage Persistence**: Sidebar state persists across page reloads
- ✅ **Zero Horizontal Scroll**: No horizontal scrolling on any viewport size
- ✅ **All Navigation Preserved**: All existing nav items and functionality maintained
- ✅ **Closes GitLab Issue #1**: Navigation bar redesign completed

### **v2.3** - Enhanced Profile Settings Style App-Wide
- ✅ **Enhanced Button Styling**: All buttons now feature thicker borders (2px) and enhanced hover effects
- ✅ **Improved Hover Animations**: Cards lift 8px on hover (was 2px) with white glow shadows
- ✅ **Enhanced Primary Buttons**: Gradient backgrounds with reverse animation on hover
- ✅ **Enhanced Secondary Buttons**: Improved shadows and 3px lift on hover with white glow
- ✅ **Enhanced Ghost & Outline Buttons**: Pure white text with enhanced hover transformations
- ✅ **Enhanced Input Fields**: Thicker 2px borders, red glow on focus, subtle lift effect
- ✅ **Enhanced Modal Animations**: Bounce entry animation with enhanced shadows and white glow
- ✅ **Enhanced Modal Close Button**: Red border on hover with subtle lift effect
- ✅ **Enhanced Dropdown Menus**: Gradient backgrounds, thicker borders, bounce animations
- ✅ **Enhanced Dropdown Items**: Pure white text, red icon accents on hover, slide effect (6px)
- ✅ **Smoother Transitions**: All interactive elements use 0.3s ease transitions
- ✅ **Professional Polish**: Modern, polished look matching profile settings throughout app

### **v1.62** - Support Experience Refresh
- ✅ **Help / About / Contact makeover**: New gradient heroes, richer layouts, and responsive grids aligned with the dashboard aesthetic
- ✅ **Clearer support playbooks**: Channel tiles, SLA chips, and onboarding checklists make it easier to find the right escalation path
- ✅ **Contact call-to-action**: Structured contact details with callback option and status references keep crews in the loop

### **v1.61** - Customizable Dashboard Widgets
- ✅ **Per-User Metrics**: Dashboard cards can be tailored per user and persist across sessions
- ✅ **Direct Module Shortcuts**: Every widget links straight to the relevant management section
- ✅ **New Preferences API**: Added `/api/v1/users/me/dashboard/widgets` for loading and saving selections
- ✅ **Database Support**: Introduced `user_dashboard_widgets` table for storing widget preferences

### **v2.2** - Pure Grayscale Design & Gradients
- ✅ **Removed All Bluish Colors**: Eliminated all remaining blue/cyan tints from UI
- ✅ **Fixed Bluish Glows**: Shadow-glow now uses white instead of blue-gray
- ✅ **Removed Bluish Backgrounds**: All rgba(15, 23, 42) changed to pure gray
- ✅ **Fixed Bluish Text-Shadow**: Brand hover glow changed from blue to white
- ✅ **Beautiful Grayscale Gradients**: Added 5 professional gradient definitions
- ✅ **Cards Use Gradients**: Cards now use gradient-card for visual depth
- ✅ **Buttons Use Gradients**: Secondary buttons feature grayscale gradients
- ✅ **Modal Gradients**: Modal headers and content styled with subtle gradients
- ✅ **All Text White or Light**: Changed all gray text (var(--gray-400)) to white/light gray
- ✅ **Enhanced Visual Depth**: Gradients add professional polish without color

### **v2.1** - Cookie Domain SSO Fix
- ✅ **Login Functionality Restored**: Fixed cookie domain parsing for SSO
- ✅ **Proper Domain Extraction**: Rewrote getCookieDomain() with clearer logic
- ✅ **Cross-Subdomain Cookies**: Cookie now correctly uses `.server-nt.de` (with leading dot)
- ✅ **SSO Working**: Sessions persist across rent.server-nt.de and storage.server-nt.de
- ✅ **Debug Logging Added**: Track domain parsing for troubleshooting
- ✅ **Fixed Parse Logic**: Simplified string parsing eliminates edge cases
- ✅ **Production Ready**: Login works properly with session cookie recognition

### **v2.0** - White Text & Login Fix
- ✅ **All Text White**: Changed all text colors to pure white (#FFFFFF)
- ✅ **Fixed Login Page**: Replaced old color variables with WarehouseCore red
- ✅ **Login Button Red**: Login button now uses #D0021B background
- ✅ **Red Input Focus**: Login inputs show red focus border
- ✅ **Fixed Broken Colors**: Removed all references to deleted accent-electric/purple variables
- ✅ **White Secondary Text**: text-secondary now white instead of gray
- ✅ **Light Muted Text**: text-muted now very light gray for better readability
- ✅ **Login Functional**: All login colors fixed and working

### **v1.9** - WarehouseCore Color Pattern Implementation
- ✅ **Red Primary Buttons**: All primary buttons now use #D0021B (WarehouseCore red)
- ✅ **Red Active Navigation**: Active nav items highlighted with red background + shadow
- ✅ **Red Input Focus**: Input fields show red border on focus (like WarehouseCore)
- ✅ **Semantic Colors Restored**: Green (success), Yellow (warning), Blue (info), Red (error)
- ✅ **Button Hover Effects**: Red shadow effects on hover for primary actions
- ✅ **Active Dropdown Items**: Red background for active selections
- ✅ **Removed Gray-Only Limitation**: Now uses proper color semantics
- ✅ **Visual Consistency**: Matches WarehouseCore's color usage pattern

### **v1.8** - WarehouseCore Color Scheme Match
- ✅ **Complete Color Overhaul**: Matched WarehouseCore's exact color palette
- ✅ **WarehouseCore Red**: Changed accent from #dc2626 to #D0021B (WarehouseCore red)
- ✅ **Pure Grayscale Palette**: #0B0B0B, #111111, #161616, #1F1F1F, #2A2A2A backgrounds
- ✅ **Removed ALL Color Tints**: Eliminated cyan, blue, green, orange rgba values
- ✅ **Solid Backgrounds**: Replaced gradients with solid colors
- ✅ **Pure White Text**: #FFFFFF on dark backgrounds, #111111 on light backgrounds
- ✅ **Gray Neutrals**: All semantic colors (success/warning/info) now use gray tones
- ✅ **Brand Consistency**: 100% alignment with WarehouseCore design language

### **v1.7** - Pure Grayscale Color Correction
- ✅ **Complete Bluish Tone Removal**: Eliminated all remaining blue/bluish tints
- ✅ **Pure Neutral Grays**: Primary colors converted from blue-grey to pure neutral (#111827, #1f2937, etc.)
- ✅ **Pure Black Surfaces**: Surface colors changed from bluish slate to pure black/gray (#000000)
- ✅ **Neutral Text**: Text colors changed from bluish to neutral white/gray (#ffffff, #d1d5db)
- ✅ **Grayscale Status**: Success/Warning changed from green/orange to gray tones
- ✅ **Final Color Palette**: Only gray, white, black, and red (#dc2626) accents
- ✅ **Brand Consistency**: Complete alignment with the unified RentalCore/WarehouseCore visual language

### **v1.6** - Professional Design Refinement
- ✅ **Color Scheme Overhaul**: Replaced colorful accents with professional grayscale + red
- ✅ **Grayscale Palette**: Extended gray-50 to gray-900 for consistent UI elements
- ✅ **Accent Red**: Primary accent color for important actions (#dc2626)
- ✅ **Removed Cyan/Purple/Gold**: Eliminated distracting colorful accents
- ✅ **Animation Reduction**: Removed aggressive slide-in and glow effects
- ✅ **Subtle Transitions**: Reduced animation durations (0.6s → 0.2s)
- ✅ **Accessibility**: Added `prefers-reduced-motion` media query support
- ✅ **Performance**: Lighter page loads without heavy animations
- ✅ **WCAG 2.2 AA**: Maintained contrast compliance with new color scheme

### **v1.3** - WebAuthn/Passkey SSO Fix
- ✅ **Fixed WebAuthn Login**: Passkey login now properly sets shared cookie domain
- ✅ **Complete SSO Support**: Both username/password and passkey login work with SSO
- ✅ **Seamless Cross-App Authentication**: Login persists across RentalCore and WarehouseCore
- ✅ **Production Ready**: Full SSO support for all authentication methods

### **v1.2** - SSO Support with WarehouseCore
- ✅ **Single Sign-On (SSO)**: Shared session cookies across RentalCore and WarehouseCore
- ✅ **Shared Cookie Domain**: Automatic cookie domain detection for subdomain SSO
- ✅ **Enhanced Session Management**: Cookie domain support for cross-app authentication
- ✅ **Seamless Navigation**: Users stay logged in when switching between apps
- ✅ **Production Ready**: Works with localhost (no domain) and production subdomains (.server-nt.de)

### **v2.85** - Comprehensive PDF Processing Workflow
- ✅ **OCR Text Extraction**: Enhanced PDF text extraction using ledongthuc/pdf library
- ✅ **Intelligent Document Parsing**: Automatic detection of customer info, dates, line items, prices
- ✅ **PDF Review Screen**: Visual review interface showing all extracted data with confidence scores
- ✅ **Auto-Mapping Engine**: Fuzzy matching algorithm with Levenshtein distance for product mapping
- ✅ **Mapping Interface**: Interactive mapping screen with suggestions and manual search
- ✅ **Smart Suggestions**: AI-powered product suggestions with confidence scoring
- ✅ **Learning System**: pdf_product_mappings table saves user mappings for improved accuracy
- ✅ **Complete Workflow**: Upload → Extract → Review → Map → Assign to Jobs
- ✅ **API Endpoints**: 9 new endpoints for PDF processing and product mapping
- ✅ **Mobile-Responsive UI**: Fully responsive templates for review and mapping screens

### **v2.9** - Complete Responsive Design System
- ✅ **Mobile-First Responsive Design**: Complete overhaul with mobile-first approach
- ✅ **Adaptive Navigation**: Mobile drawer, tablet rail, desktop sidebar navigation
- ✅ **Responsive Tables**: Card transformation and horizontal scroll options for mobile
- ✅ **Fluid Typography**: CSS clamp() implementation for scalable text (14px-48px)
- ✅ **Touch Optimization**: 44px minimum touch targets, enhanced focus states
- ✅ **Responsive Forms**: Single-column mobile, multi-column desktop layouts
- ✅ **Modal Enhancements**: Full-screen mobile modals, adaptive tablet/desktop sizing
- ✅ **Accessibility Improvements**: WCAG 2.2 AA compliance, reduced motion support
- ✅ **Layout Primitives**: Stack, Inline, Cluster, Sidebar responsive patterns
- ✅ **Responsive Utilities**: Breakpoint visibility controls, responsive images

### **v2.4** - Rental Equipment System
- ✅ Complete rental equipment management system for external equipment
- ✅ Rental equipment database tables with job integration
- ✅ Dedicated rental equipment management page with CRUD operations
- ✅ Rental equipment analytics dashboard with charts and statistics
- ✅ Job integration with manual entry and existing equipment selection
- ✅ Products navbar dropdown for Own Products vs Rental Equipment
- ✅ Supplier management and category-based organization
- ✅ Real-time cost calculation and usage tracking

### **v1.4**
- ✅ Enhanced job view modal with comprehensive device management
- ✅ Revenue and device count display in job modals
- ✅ Clickable device count for detailed device overview
- ✅ Device overview grouped by 5 categories (Sound, Light, Effect, Stage, Other)
- ✅ Real-time device price adjustment per job with API integration
- ✅ Toast notifications for user feedback on price changes
- ✅ Improved customer display and status handling in job modals

### **v1.3.0**
- ✅ Complete device analytics modal with detailed insights
- ✅ Enhanced Docker deployment with configuration examples
- ✅ Comprehensive .gitignore with credential protection
- ✅ UTF-8 PDF export fixes for proper currency formatting

### **v1.1.0**
- ✅ Analytics dashboard complete rewrite
- ✅ Fixed dropdown functionality and data display issues
- ✅ Professional RentalCore theming implementation

## 📧 Support & Contact

- **Issues**: [GitHub Issues](https://github.com/nbt4/RentalCore/issues)
- **Docker Hub**: [nbt4/rentalcore](https://hub.docker.com/r/nbt4/rentalcore)
- **Documentation**: [GitHub Repository](https://github.com/nbt4/RentalCore)

## 📱 Responsive Design

RentalCore features a comprehensive responsive design system built from the ground up for optimal user experience across all devices.

### 🎯 **Design Philosophy**
- **Mobile-First Approach**: Designed primarily for mobile devices, progressively enhanced for larger screens
- **Touch-Optimized**: All interactive elements meet WCAG 2.2 AA guidelines with 44×44px minimum touch targets
- **Accessibility-Focused**: Enhanced focus states, reduced motion support, and screen reader optimization
- **Performance-Oriented**: Fluid typography and spacing using CSS clamp() functions

## 🎨 Design System & Responsive Guidelines

### **Design Tokens**

#### Spacing Scale (Consistent 4,8,12,16,20,24,32,40,48)
```css
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
```

#### Typography Scale (Fluid with clamp)
```css
--text-xs: clamp(0.75rem, 0.9vw, 0.875rem);    /* 12-14px - Label/Meta */
--text-sm: clamp(0.875rem, 1.0vw, 1rem);       /* 14-16px - Body */
--text-base: clamp(1rem, 1.0vw, 1rem);         /* 16px - Body baseline */
--text-xl: clamp(1.125rem, 1.4vw, 1.375rem);   /* 18-22px - H3 */
--text-2xl: clamp(1.375rem, 1.8vw, 1.75rem);   /* 22-28px - H2 */
--text-3xl: clamp(1.75rem, 2.4vw, 2.25rem);    /* 28-36px - H1 */
```

#### Border Radius
```css
--radius-sm: 0.375rem;   /* 6px */
--radius-md: 0.625rem;   /* 10px */
--radius-lg: 1rem;       /* 16px */
```

#### Container Max-Widths
```css
--container-sm: 40rem;     /* 640px */
--container-md: 48rem;     /* 768px */
--container-lg: 64rem;     /* 1024px */
--container-xl: 80rem;     /* 1280px */
--container-2xl: 90rem;    /* 1440px max content width */
```

### **Grid System**

#### Auto-Fill Grids (Prevents Tiny Cards)
```css
.rc-grid-auto {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--space-6); /* 24px on mobile */
    gap: var(--space-8); /* 32px on desktop */
}
```

#### Card Grids with Minmax Rules
- **Mobile**: Single column, full width
- **Tablet/Desktop**: `repeat(auto-fill, minmax(280px, 1fr))`
- **Large Desktop**: `repeat(auto-fill, minmax(320px, 1fr))`

### **Layout Patterns**

#### App Shell Centering
```css
.rc-container {
    max-width: var(--container-xl); /* 1280px */
    margin: 0 auto;
    padding: 0 var(--space-4); /* 16px base */
    padding: 0 var(--space-6); /* 24px on tablet+ */
    padding: 0 var(--space-8); /* 32px on large desktop */
}
```

#### Form Grids
- **Mobile**: Single column layout
- **Tablet (768px+)**: Two-column grid for appropriate fields
- **Desktop (1024px+)**: Three-column grid for complex forms
- **Gap**: Consistent `var(--space-5)` (20px)

#### Data Tables
- **Desktop**: Generous padding (16px vertical, 24px horizontal)
- **Tablet**: Reduced padding (12px vertical, 16px horizontal)
- **Mobile**: Card transformation OR horizontal scroll with sticky first column
- **Touch Targets**: 44-56px row heights

### **Accessibility Features**

#### Touch Targets
- **Minimum**: 44×44px for all interactive elements
- **Buttons**: Consistent `min-height: 44px`
- **Form Inputs**: `min-height: 44px` with proper padding

#### Focus States
- **Focus-visible**: 2px solid accent outline with 2px offset
- **Focus Rings**: Only visible for keyboard navigation
- **High Contrast**: Maintains WCAG 2.2 AA compliance

#### Responsive Images
```css
.responsive-image {
    width: 100%;
    height: auto;
    aspect-ratio: 16/9; /* Intrinsic sizing */
    object-fit: cover;
}
```

### **Performance Optimizations**

#### CSS Features
- **Fluid Typography**: Uses `clamp()` for optimal scaling
- **Container Queries**: Future-ready responsive patterns
- **CSS Grid**: Prefer over flexbox for 2D layouts
- **Aspect Ratio**: Prevents cumulative layout shift

#### Best Practices
- Content-driven heights (avoid forced equal heights)
- Lazy loading for non-critical media
- Efficient selector specificity
- Minimal layout thrash with CSS-only solutions

### 📱 **Breakpoint Strategy**
- **xs (360-479px)**: Compact phones with stacked layouts
- **sm (480-639px)**: Large phones with selective horizontal arrangements
- **md (640-767px)**: Small tablets and landscape phones
- **lg (768-1023px)**: Tablets with icon rail navigation
- **xl (1024-1279px)**: Small laptops with full features
- **2xl (1280px+)**: Desktop monitors with expanded layouts

### 🧩 **Component Responsiveness**

#### Navigation System
- **Mobile**: Full-screen drawer navigation with backdrop blur
- **Tablet**: Compact icon rail with tooltips for space efficiency
- **Desktop**: Full sidebar navigation with labels and dropdowns

#### Data Tables
- **Mobile**: Transform to card-based layout for better readability
- **Alternative**: Horizontal scroll with sticky first column and header
- **Tablet**: Compact spacing with column prioritization
- **Desktop**: Full table layout with enhanced hover states

#### Forms & Modals
- **Mobile**: Single-column layouts, full-screen modals for complex dialogs
- **Tablet**: Two-column grids where appropriate, adaptive modal sizing
- **Desktop**: Multi-column layouts with optimized field grouping

#### Layout Primitives
- **Stack**: Vertical layouts with responsive spacing
- **Inline**: Horizontal wrapping with intelligent overflow
- **Cluster**: Flexible button groups that stack on mobile
- **Sidebar**: Responsive content/sidebar combinations

### 🎨 **Fluid Design System**
- **Typography**: Scales from 14px to 48px using clamp() functions
- **Spacing**: Responsive spacing scale from 4px to 96px
- **Components**: Auto-adapting cards, forms, and data displays
- **Images**: Aspect ratio preservation with responsive sizing

## 📷 Demo Images

Get a visual overview of RentalCore's professional interface and features:

### Login & Authentication
<img src="img/login-page.png" alt="RentalCore Login Page" width="600">

Professional login interface with secure authentication and modern design.

### Main Dashboard
<img src="img/dashboard-overview.png" alt="RentalCore Dashboard Overview" width="600">

Comprehensive dashboard showing active customers, equipment items, jobs, and quick action buttons for daily operations.

### Equipment Management
<img src="img/devices-management.png" alt="Devices Management" width="600">

Complete device inventory management with search, status tracking, and bulk operations.

<img src="img/products-management.png" alt="Products Management" width="600">

Product catalog management with categories, pricing, and detailed descriptions.

<img src="img/device-tree-view.png" alt="Device Tree View" width="600">

Hierarchical device organization by categories and types for easy navigation and management.

### Cable & Case Management (WarehouseCore)
<img src="img/cables-management.png" alt="Cables Management" width="600">

Specialized cable management with connector types, lengths, and cross-section specifications. Now managed exclusively in WarehouseCore.

<img src="img/cases-management.png" alt="Cases Management" width="600">

Equipment case tracking with device capacity and availability status. Now managed exclusively in WarehouseCore.

<img src="img/case-devices-modal.png" alt="Case Devices Modal" width="600">

Detailed view of devices within equipment cases for precise inventory control. Case management now handled in WarehouseCore.

### Job Management
Job selection interface with overview of active jobs, available devices, and assignments.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

The MIT License allows you to:
- ✅ Use the software for any purpose (commercial or non-commercial)
- ✅ Modify and distribute the software
- ✅ Create derivative works
- ✅ Use in private projects
- ✅ Sell copies or services based on the software

**No warranty is provided - use at your own risk.**

---

**🎯 Ready for Production**: RentalCore is designed for professional equipment rental businesses requiring comprehensive analytics, robust security, and scalable Docker deployment.
