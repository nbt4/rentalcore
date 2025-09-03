# RentalCore User Guide

## Getting Started

### First Login
1. Navigate to your RentalCore installation (e.g., http://localhost:8080)
2. Login with default credentials: `admin` / `admin123`
3. **Important**: Change the default password immediately after first login

## Main Features

### Dashboard Overview
The dashboard provides a quick overview of:
- Active rentals and upcoming returns
- Equipment availability status
- Recent customer activity
- Revenue summary

### Customer Management

#### Adding Customers
1. Navigate to "Customers" in the main menu
2. Click "Add New Customer"
3. Fill in customer details:
   - Personal information (name, email, phone)
   - Company details (if applicable)
   - Address information
4. Click "Save Customer"

#### Managing Customers
- **Search**: Use the search bar to find customers by name, company, or email
- **Edit**: Click the edit icon to modify customer information
- **View History**: Click on a customer to see their rental history

### Equipment Management

#### Device Categories and Products
- **Categories**: Organize equipment into logical groups (Audio, Lighting, etc.)
- **Products**: Define equipment types with daily rental rates
- **Devices**: Individual equipment items with serial numbers and status

#### Adding Equipment
1. Navigate to "Devices" in the main menu
2. Click "Add New Device"
3. Select product type and enter device details:
   - Device ID (unique identifier)
   - Serial number
   - Purchase information
   - Condition notes
4. Click "Save Device"

#### Device Status Management
- **Available**: Ready for rental
- **Checked Out**: Currently on a job
- **Maintenance**: Under repair or servicing
- **Retired**: No longer available for rental

### Job Management

#### Creating Rental Jobs
1. Navigate to "Jobs" in the main menu
2. Click "Create New Job"
3. Fill in job details:
   - Select customer
   - Set start and end dates
   - Add job description and location
   - Assign equipment to the job
4. Click "Create Job"

#### Job Lifecycle
1. **Planning**: Initial job creation and equipment assignment
2. **Active**: Equipment deployed for the rental period
3. **Completed**: Job finished and equipment returned
4. **Cancelled**: Job cancelled before completion

#### Equipment Assignment
- Use bulk scanning to quickly assign multiple devices
- Generate QR codes for easy device identification
- Track equipment condition before and after rental

### Analytics and Reporting

#### Analytics Dashboard
Access comprehensive analytics including:
- Revenue trends over different time periods
- Equipment utilization rates
- Customer activity analysis
- Top performing equipment and customers

#### Individual Device Analytics
Click on any device to view:
- Revenue generated over time
- Utilization statistics
- Booking history
- Performance metrics

#### Exporting Data
- **PDF Export**: Professional reports for presentations
- **CSV Export**: Data for external analysis
- **Invoice Generation**: Professional invoices for customers

### Search and Navigation

#### Global Search
Use the search bar in the top navigation to quickly find:
- Customers by name, company, or email
- Devices by ID or serial number
- Jobs by description or customer
- Any combination of the above

#### Quick Actions
- **Scan QR Code**: Use camera to quickly identify equipment
- **Bulk Operations**: Perform actions on multiple items
- **Quick Add**: Rapid creation of common items

## Best Practices

### Equipment Management
1. Always update device status when equipment is deployed or returned
2. Use descriptive device IDs and serial numbers
3. Regular maintenance scheduling to keep equipment in good condition
4. Keep purchase and condition information up to date

### Customer Relations
1. Maintain accurate customer contact information
2. Keep notes about customer preferences and requirements
3. Follow up on overdue returns promptly
4. Generate professional invoices for all completed jobs

### Data Management
1. Regular backups of important data
2. Keep job descriptions detailed for future reference
3. Use consistent naming conventions
4. Regular cleanup of old or irrelevant data

## Troubleshooting

### Common Issues

#### Login Problems
- Verify username and password
- Clear browser cache and cookies
- Check if account is active

#### Equipment Not Showing
- Check device status filter settings
- Verify device is not already assigned to another job
- Ensure device exists in the system

#### Analytics Not Loading
- Check if you have sufficient data for the selected time period
- Try refreshing the page
- Verify database connectivity

### Getting Help
If you encounter issues not covered in this guide:
1. Check the troubleshooting documentation
2. Review application logs for error messages
3. Contact your system administrator
4. Submit an issue on GitHub for bugs or feature requests