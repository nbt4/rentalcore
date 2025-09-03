# RentalCore Security Guide

## Overview
RentalCore implements enterprise-grade security features to protect your rental management data.

## Authentication & Authorization

### Multi-Factor Authentication (2FA)
- WebAuthn support for passwordless authentication
- Time-based One-Time Password (TOTP) support
- SMS verification for additional security

### Role-Based Access Control (RBAC)
- **Admin**: Full system access
- **Manager**: Job and device management
- **User**: Limited read access

## Data Protection

### Encryption
- AES-256 encryption for sensitive data
- TLS 1.3 for data in transit
- Encrypted database credentials

### Password Security
- Minimum 8 characters
- Must include uppercase, lowercase, number, special character
- Password hashing with bcrypt
- Session timeout after 1 hour of inactivity

## Security Features

### Input Validation
- SQL injection prevention
- XSS protection
- CSRF tokens
- Input sanitization

### Network Security
- HTTPS/TLS termination
- CORS protection
- Rate limiting
- IP whitelisting support

## Compliance

### GDPR Features
- Data retention policies
- Right to be forgotten
- Data export functionality
- Privacy controls

### Audit Logging
- All user actions logged
- Failed login attempts tracked
- Data changes audited
- Security event monitoring

## Best Practices

1. **Change Default Credentials**: Always change the default admin password
2. **Use Strong Passwords**: Enforce password complexity
3. **Regular Updates**: Keep Docker images updated
4. **Secure Headers**: Implement security headers in reverse proxy
5. **Database Security**: Use dedicated database user with minimal privileges
6. **Backup Encryption**: Encrypt all backup files

## Security Checklist

- [ ] Changed default admin password
- [ ] Configured HTTPS/TLS
- [ ] Set up firewall rules
- [ ] Configured secure headers
- [ ] Enabled audit logging
- [ ] Set up monitoring alerts
- [ ] Configured backup encryption