# RentalCore Administrator Guide

## System Administration

### Initial Setup
1. **Change Default Password**: First priority after installation
2. **Configure Database**: Ensure proper database credentials and connectivity
3. **Set Up SSL/TLS**: Configure HTTPS for production environments
4. **Configure Email**: Set up SMTP for notifications (optional)

### User Management

#### Creating Admin Users
1. Navigate to "Users" in the admin panel
2. Click "Add New User"
3. Set role to "Admin" for full system access
4. Require strong password and enable 2FA

#### User Roles
- **Admin**: Full system access, user management, system configuration
- **Manager**: Job and equipment management, customer access
- **User**: Read-only access to assigned jobs and equipment

#### Security Policies
- Enforce strong password requirements
- Enable session timeout (default: 1 hour)
- Implement account lockout after failed attempts
- Regular password rotation policy

### System Configuration

#### Environment Variables
Critical settings in `.env` file:
```bash
# Security
ENCRYPTION_KEY=your-256-bit-key
SESSION_SECRET=your-session-secret
SESSION_TIMEOUT=3600

# Database
DB_HOST=database-host
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=secure-password

# Features
ENABLE_2FA=true
ENABLE_AUDIT_LOG=true
```

#### Application Configuration
Key settings in `config.json`:
- Company branding and theme
- Currency and localization
- Feature toggles
- Performance settings

### Database Administration

#### Backup Strategy
```bash
# Daily automated backup
mysqldump -u username -p rentalcore > backup_$(date +%Y%m%d).sql

# Docker backup
docker exec mysql-container mysqldump -u root -p rentalcore > backup.sql
```

#### Database Maintenance
- Regular optimization of analytics cache
- Cleanup of old log entries
- Index maintenance for performance
- Monitor database size and growth

### Monitoring and Logging

#### Application Logs
Monitor these log files:
- `/app/logs/application.log` - General application events
- `/app/logs/security.log` - Security events and failed logins
- `/app/logs/error.log` - Application errors and exceptions

#### Health Monitoring
- Use `/health` endpoint for application health checks
- Monitor response times and error rates
- Set up alerts for critical issues
- Track resource usage (CPU, memory, disk)

### Security Administration

#### SSL/TLS Configuration
```yaml
# Docker Compose with Traefik
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.rentalcore.rule=Host(`rental.yourdomain.com`)"
  - "traefik.http.routers.rentalcore.tls.certresolver=letsencrypt"
```

#### Security Headers
Implement these headers in your reverse proxy:
```
Strict-Transport-Security: max-age=31536000
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

#### Audit Logging
All administrative actions are logged including:
- User creation and deletion
- Password changes
- System configuration changes
- Data exports and sensitive operations

### Performance Optimization

#### Database Performance
- Monitor slow query log
- Optimize indexes based on query patterns
- Configure appropriate connection pool size
- Regular ANALYZE TABLE maintenance

#### Application Performance
- Monitor memory usage and optimize if needed
- Configure appropriate cache settings
- Optimize image and file uploads
- Use CDN for static assets in production

### Backup and Recovery

#### Backup Checklist
- [ ] Daily database backups
- [ ] Weekly full system backups
- [ ] Monthly backup verification
- [ ] Offsite backup storage
- [ ] Document recovery procedures

#### Recovery Procedures
1. **Database Recovery**:
   ```bash
   mysql -u username -p rentalcore < backup.sql
   ```

2. **File Recovery**:
   ```bash
   tar -xzf uploads_backup.tar.gz -C /app/uploads/
   ```

3. **Full System Recovery**:
   - Restore database from backup
   - Restore uploaded files
   - Verify application configuration
   - Test all critical functions

### Maintenance Tasks

#### Daily Tasks
- Monitor application logs
- Check system health status
- Verify backup completion
- Review security events

#### Weekly Tasks
- Analyze performance metrics
- Review user activity reports
- Update system documentation
- Plan capacity requirements

#### Monthly Tasks
- Security vulnerability assessment
- System performance review
- Backup recovery testing
- User access review

### Troubleshooting

#### Common Admin Issues

**Application Won't Start**
- Check environment variables
- Verify database connectivity
- Review application logs
- Validate configuration files

**Performance Issues**
- Monitor database query performance
- Check system resources (CPU, memory, disk)
- Review analytics cache settings
- Optimize database indexes

**Security Concerns**
- Review audit logs for suspicious activity
- Check failed login attempts
- Verify SSL/TLS configuration
- Validate user permissions

#### Log Analysis
```bash
# Check recent errors
tail -f /app/logs/error.log

# Monitor security events
grep "SECURITY" /app/logs/application.log

# Analyze performance
grep "SLOW_QUERY" /app/logs/application.log
```

### System Updates

#### Update Procedure
1. **Backup Current System**
   - Database backup
   - Configuration backup
   - File backup

2. **Update Application**
   ```bash
   docker pull nbt4/rentalcore:latest
   docker-compose up -d
   ```

3. **Verify Update**
   - Check application health
   - Test critical functions
   - Verify data integrity

4. **Rollback if Needed**
   ```bash
   docker-compose down
   docker-compose up -d nbt4/rentalcore:previous-version
   ```

### Best Practices

#### Security Best Practices
- Regular security updates
- Strong authentication policies
- Principle of least privilege
- Regular security audits
- Incident response procedures

#### Operational Best Practices
- Automated monitoring and alerts
- Regular backup testing
- Capacity planning
- Documentation maintenance
- Change management procedures