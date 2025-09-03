# RentalCore Troubleshooting Guide

## Common Issues and Solutions

### Installation and Setup Issues

#### Database Connection Failed
**Problem**: Application fails to connect to database
**Solutions**:
1. Verify database credentials in `.env` file
2. Ensure database server is running and accessible
3. Check firewall settings on database host
4. Verify database name exists and user has proper permissions
5. Test connection manually:
   ```bash
   mysql -h your-host -u username -p database_name
   ```

#### Docker Container Won't Start
**Problem**: RentalCore container fails to start
**Solutions**:
1. Check Docker logs:
   ```bash
   docker-compose logs rentalcore
   ```
2. Verify environment variables are set correctly
3. Ensure ports are not already in use:
   ```bash
   netstat -tulpn | grep :8080
   ```
4. Check file permissions on mounted volumes
5. Verify Docker image is latest version:
   ```bash
   docker pull nbt4/rentalcore:latest
   ```

### Application Issues

#### Analytics Page Not Loading
**Problem**: Analytics dashboard shows no data or fails to load
**Solutions**:
1. Check if you have sufficient data for the selected time period
2. Clear browser cache and reload the page
3. Check browser console for JavaScript errors
4. Verify database has completed jobs with revenue data
5. Check application logs for database query errors

#### Dropdowns Not Working
**Problem**: Dropdown menus not responding or showing options
**Solutions**:
1. Clear browser cache and cookies
2. Disable browser extensions that might interfere
3. Check browser console for JavaScript errors
4. Verify page is fully loaded before interacting
5. Try different browser or incognito mode

#### PDF Export Issues
**Problem**: PDF exports showing corrupted characters or failing
**Solutions**:
1. Ensure UTF-8 encoding is properly configured
2. Check for special characters in data (€ symbols, umlauts)
3. Verify sufficient system memory for PDF generation
4. Check application logs for PDF generation errors
5. Try exporting smaller date ranges

### Performance Issues

#### Slow Loading Times
**Problem**: Application pages load slowly
**Solutions**:
1. Check database performance and optimize queries
2. Verify adequate system resources (CPU, RAM)
3. Check network connectivity between application and database
4. Clear analytics cache to refresh stale data
5. Consider database indexing optimization

#### High Memory Usage
**Problem**: Docker container using excessive memory
**Solutions**:
1. Monitor database connection pool size
2. Check for memory leaks in application logs
3. Restart the container to clear memory:
   ```bash
   docker-compose restart rentalcore
   ```
4. Reduce concurrent database connections
5. Consider upgrading server resources

### User Access Issues

#### Login Failed
**Problem**: Unable to login with correct credentials
**Solutions**:
1. Verify username and password are correct
2. Check if account is active and not locked
3. Clear browser cookies and cache
4. Verify database connectivity
5. Check for typos in username (case sensitive)
6. Reset password if necessary

#### Permission Denied
**Problem**: User cannot access certain features
**Solutions**:
1. Verify user role and permissions
2. Check if feature is enabled in configuration
3. Ensure user is logged in with correct account
4. Contact administrator to verify role assignments
5. Check application logs for authorization errors

### Data Issues

#### Missing Equipment
**Problem**: Devices not showing in equipment lists
**Solutions**:
1. Check device status filters (available, maintenance, etc.)
2. Verify device is not assigned to active job
3. Check if device was accidentally deleted
4. Search by device ID or serial number
5. Check database for data consistency

#### Revenue Data Incorrect
**Problem**: Analytics showing wrong revenue numbers
**Solutions**:
1. Verify job completion dates are set correctly
2. Check if both `revenue` and `final_revenue` fields are populated
3. Ensure jobs have proper status (completed vs. active)
4. Verify currency settings in configuration
5. Check for duplicate job entries

#### Customer Information Missing
**Problem**: Customer data not displaying correctly
**Solutions**:
1. Check if customer record exists in database
2. Verify customer is not accidentally archived
3. Check for special characters in customer names
4. Ensure proper data encoding (UTF-8)
5. Check import process if data was bulk imported

### System Administration Issues

#### Backup Failures
**Problem**: Automated backups not working
**Solutions**:
1. Check disk space on backup destination
2. Verify backup script permissions
3. Check database credentials for backup user
4. Ensure backup directory is writable
5. Test backup process manually

#### SSL/HTTPS Issues
**Problem**: SSL certificate errors or HTTPS not working
**Solutions**:
1. Verify SSL certificate is valid and not expired
2. Check reverse proxy configuration (Traefik, Nginx)
3. Ensure proper DNS configuration
4. Verify certificate chain is complete
5. Check firewall settings for HTTPS traffic

### Diagnostic Commands

#### Health Check
```bash
# Check application health
curl http://localhost:8080/health

# Check Docker container status
docker-compose ps

# View recent logs
docker-compose logs --tail=50 rentalcore
```

#### Database Diagnostics
```bash
# Test database connection
docker exec -it mysql-container mysql -u root -p

# Check database size
SELECT 
    table_schema AS "Database",
    ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS "Size (MB)"
FROM information_schema.tables 
GROUP BY table_schema;

# Verify sample data
USE rentalcore;
SELECT COUNT(*) FROM customers;
SELECT COUNT(*) FROM devices;
SELECT COUNT(*) FROM jobs;
```

#### Performance Monitoring
```bash
# Monitor resource usage
docker stats rentalcore

# Check system load
htop

# Monitor disk usage
df -h
```

## Getting Additional Help

### Log Analysis
Always check the application logs when troubleshooting:
```bash
# Application logs
docker-compose logs -f rentalcore

# System logs
journalctl -u docker
```

### Support Resources
1. **GitHub Issues**: https://github.com/nbt4/rentalcore/issues
2. **Documentation**: Check all files in the `docs/` folder
3. **Community Support**: GitHub Discussions
4. **Database Setup**: Refer to `docs/DATABASE_SETUP.md`

### Reporting Issues
When reporting issues, please include:
1. RentalCore version (Docker image tag)
2. Operating system and Docker version
3. Relevant log excerpts
4. Steps to reproduce the problem
5. Expected vs actual behavior

### Emergency Recovery
If the system is completely unresponsive:
1. Stop all containers: `docker-compose down`
2. Check logs: `docker-compose logs rentalcore`
3. Restore from backup if necessary
4. Start with minimal configuration to isolate issues
5. Contact support with detailed error information