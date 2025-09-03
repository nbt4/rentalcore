# RentalCore API Documentation

## Overview
RentalCore provides a comprehensive REST API for managing equipment rentals, customers, and analytics.

## Base URL
```
http://localhost:8080/api/v1
```

## Authentication
All API endpoints require authentication via session cookies or API tokens.

## Core Endpoints

### Jobs Management
- `GET /api/v1/jobs` - List all jobs
- `POST /api/v1/jobs` - Create new job
- `PUT /api/v1/jobs/:id` - Update job
- `DELETE /api/v1/jobs/:id` - Delete job

### Device Management
- `GET /api/v1/devices` - List all devices
- `POST /api/v1/devices` - Create new device
- `PUT /api/v1/devices/:id` - Update device
- `DELETE /api/v1/devices/:id` - Delete device

### Customer Management
- `GET /api/v1/customers` - List all customers
- `POST /api/v1/customers` - Create new customer
- `PUT /api/v1/customers/:id` - Update customer
- `DELETE /api/v1/customers/:id` - Delete customer

### Analytics Endpoints
- `GET /analytics` - Main analytics dashboard
- `GET /analytics/devices/:deviceId` - Individual device analytics
- `GET /analytics/export` - Export analytics data

## Response Format
All API responses follow this structure:
```json
{
  "status": "success|error",
  "data": {},
  "message": "Optional message"
}
```

## Error Codes
- `200` - Success
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error