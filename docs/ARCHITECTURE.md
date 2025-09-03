# RentalCore System Architecture

## Overview
RentalCore follows a clean architecture pattern with separation of concerns and modular design.

## Architecture Layers

### 1. Presentation Layer
- **Web Templates**: HTML templates with Go templating
- **Static Assets**: CSS, JavaScript, images
- **API Endpoints**: RESTful API for external integration

### 2. Handler Layer
- **HTTP Handlers**: Request processing and response generation
- **Middleware**: Authentication, logging, CORS, rate limiting
- **Input Validation**: Request validation and sanitization

### 3. Service Layer
- **Business Logic**: Core rental management logic
- **Data Processing**: Analytics calculations and reporting
- **File Management**: Document upload and processing

### 4. Data Layer
- **GORM ORM**: Database abstraction and migrations
- **MySQL Database**: Primary data storage
- **File Storage**: Local filesystem for uploads

## Directory Structure

```
rentalcore/
├── cmd/server/              # Application entry point
├── internal/
│   ├── handlers/           # HTTP request handlers
│   │   ├── analytics_handler.go
│   │   ├── device_handler.go
│   │   └── customer_handler.go
│   ├── models/             # Database models
│   │   ├── job.go
│   │   ├── device.go
│   │   └── customer.go
│   ├── services/           # Business logic
│   └── middleware/         # HTTP middleware
├── web/
│   ├── templates/          # HTML templates
│   └── static/            # Static assets
├── migrations/             # Database migrations
└── docs/                  # Documentation
```

## Database Schema

### Core Tables
- **customers**: Customer information and contacts
- **jobs**: Rental jobs and bookings
- **devices**: Equipment inventory
- **jobdevices**: Many-to-many relationship for job assignments
- **products**: Equipment types and categories
- **users**: System users and authentication

### Supporting Tables
- **categories**: Equipment categories
- **statuses**: Job status definitions
- **analytics_cache**: Performance optimization for reports

## Design Patterns

### Repository Pattern
- Abstraction layer for data access
- Testable data operations
- Database-agnostic queries

### MVC Pattern
- Model: Database entities and business logic
- View: HTML templates and user interface
- Controller: HTTP handlers and request processing

### Dependency Injection
- Service dependencies injected at startup
- Testable and maintainable code
- Configuration-driven initialization

## Performance Considerations

### Database Optimization
- Proper indexing on frequently queried columns
- Connection pooling (50 connections default)
- Query optimization for analytics

### Caching Strategy
- Analytics data caching for improved performance
- Static asset caching
- Template compilation caching

### Scalability
- Horizontal scaling ready
- Stateless application design
- External session storage support

## Security Architecture

### Authentication Flow
1. User login with credentials
2. Session creation and cookie setting
3. Request authentication via middleware
4. Role-based authorization

### Data Flow Security
- Input validation at handler level
- SQL injection prevention via ORM
- Output encoding for XSS prevention
- CSRF protection for state-changing operations