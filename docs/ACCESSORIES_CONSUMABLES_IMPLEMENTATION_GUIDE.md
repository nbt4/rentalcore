# Accessories & Consumables - Implementation Guide

## Issue #37 Implementation Status

### ✅ Completed (Backend Foundation)

#### Database Layer
- ✅ Migration scripts created (`migrations/037_accessories_consumables*.sql`)
- ✅ Tables: `count_types`, `product_accessories`, `product_consumables`, `job_accessories`, `job_consumables`, `inventory_transactions`
- ✅ Database views for efficient querying
- ✅ Extended `products` table with accessory/consumable fields
- ✅ Applied to production database

#### Backend Code
- ✅ Go models for all entities (`internal/models/accessories_consumables.go`)
- ✅ Updated Product model with new fields
- ✅ Full CRUD repository with transaction support (`internal/repository/accessories_consumables_repository.go`)
- ✅ Comprehensive API handlers (`internal/handlers/accessories_consumables_handler.go`)
- ✅ All routes registered in main.go
- ✅ Compilation successful
- ✅ API documentation created

#### Git & Deployment
- ✅ Committed to GitHub (commit: 44e9670)
- ✅ Pushed to origin/main
- ⏳ Docker image building (version 4.1.36)

---

## 🚧 Remaining Work (Frontend & Integration)

### 1. Product Management UI

#### Create Product as Accessory/Consumable
**Location**: Product creation/edit form

**Required Changes**:
1. Add checkbox: "Is Accessory"
2. Add checkbox: "Is Consumable"
3. Show additional fields when checked:
   - Count Type dropdown (from `/api/count-types`)
   - Stock Quantity input
   - Min Stock Level input
   - Generic Barcode input
   - Price Per Unit input

**Example Code** (pseudocode):
```html
<div class="form-group">
  <label>
    <input type="checkbox" id="is_accessory" name="is_accessory">
    This is an Accessory
  </label>
</div>

<div id="accessory-fields" style="display: none;">
  <select name="count_type_id">
    <!-- Populated from /api/count-types -->
  </select>
  <input type="number" name="stock_quantity" placeholder="Current Stock">
  <input type="number" name="min_stock_level" placeholder="Min Stock Alert">
  <input type="text" name="generic_barcode" placeholder="Barcode">
  <input type="number" name="price_per_unit" placeholder="Price per Unit">
</div>
```

#### Link Accessories to Products
**Location**: Product edit page, new tab "Accessories"

**Required**:
1. List current accessories (GET `/api/products/:id/accessories`)
2. Button "Add Accessory" opens modal
3. Modal shows:
   - Dropdown of all accessory products
   - Optional checkbox
   - Default quantity
   - Sort order
4. Save: POST `/api/products/:id/accessories`
5. Remove: DELETE `/api/products/:id/accessories/:accessoryID`

#### Link Consumables to Products
Same as accessories but for consumables tab.

---

### 2. Job Assignment UI

#### When Adding Product to Job
**Location**: Job edit page, device assignment

**Current Behavior**: User adds a product/device to job

**New Behavior**:
1. After adding product, check if it has accessories/consumables:
   - GET `/api/products/:id/accessories`
   - GET `/api/products/:id/consumables`

2. If has accessories, show popup:
   ```
   Select Accessories for [Product Name]
   [ ] Safety Cable 40kg (2 available) - Qty: [1]
   [ ] Clamp (10 available) - Qty: [1]
   [Cancel] [Add Selected]
   ```

3. If has consumables, show input:
   ```
   Consumables for [Product Name]
   Fog Fluid (25.5 kg available)
   Quantity: [___] kg
   [Skip] [Add]
   ```

4. On confirm, make API calls:
   - POST `/api/jobs/:jobID/accessories` (for each selected)
   - POST `/api/jobs/:jobID/consumables` (for each)

---

### 3. Job View Display

#### Show Accessories/Consumables Under Devices
**Location**: Job detail view, device list

**Current Display**:
```
- LED Par (DEV-001)
```

**New Display**:
```
- LED Par (DEV-001)
    └─ Safety Cable 40kg (2x) [2/0 scanned]
    └─ Fog Fluid (5.5kg) [0.0/0.0 scanned]
```

**Implementation**:
1. GET `/api/jobs/:jobID/accessories`
2. GET `/api/jobs/:jobID/consumables`
3. Group by `parent_device_id`
4. Display indented under each device
5. Show scan status (scanned_out/scanned_in)

---

### 4. Inventory Management UI

#### Create Inventory Dashboard
**Location**: New page `/inventory`

**Sections**:

1. **Low Stock Alerts** (GET `/api/inventory/low-stock`)
   ```
   ⚠️ Low Stock Items
   - Safety Cable 40kg: 5 pcs (min: 10)
   - Fog Fluid: 2 kg (min: 5)
   ```

2. **Stock Adjustment** (POST `/api/inventory/adjust`)
   ```
   Manual Stock Adjustment
   Product: [dropdown]
   Quantity: [+/-___]
   Reason: [___________]
   [Adjust Stock]
   ```

3. **Transaction History** (GET `/api/inventory/transactions`)
   ```
   Recent Transactions
   Date       | Product    | Type | Qty  | Reference
   2025-11-24 | Safety 40kg| Out  | -4   | Job #10001
   2025-11-24 | Fog Fluid  | In   | +10  | Adjustment
   ```

---

### 5. WarehouseCore Integration

#### Scanning Interface
**Location**: WarehouseCore job scanning page

**Current Flow**:
1. Select job
2. Scan device barcodes
3. Mark as scanned out/in

**New Flow for Accessories**:
1. Scan generic barcode (e.g., "ACC-SAFE40")
2. System calls: POST `/api/scan/accessory`
   ```json
   {
     "barcode": "ACC-SAFE40",
     "job_id": 10001,
     "direction": "out",
     "quantity": 1
   }
   ```
3. Show confirmation: "Scanned 1x Safety Cable 40kg"
4. Repeat for each piece (scan 4 times for 4 pieces)

**New Flow for Consumables**:
1. Scan generic barcode (e.g., "CONS-FOG")
2. Show popup: "Fog Fluid - Enter quantity (kg): [___]"
3. User enters quantity (e.g., 5.5)
4. System calls: POST `/api/scan/consumable`
   ```json
   {
     "barcode": "CONS-FOG",
     "job_id": 10001,
     "direction": "out",
     "quantity": 5.5
   }
   ```
5. Show confirmation: "Scanned 5.5kg Fog Fluid"

**When Scanning Back In**:
- Accessories: Same as out (scan each piece)
- Consumables: Prompt for quantity being returned

---

### 6. Admin Configuration (WarehouseCore)

#### Category Management
**Location**: WarehouseCore admin panel

**Required**:
1. Page to manage categories
2. Add "Accessories" and "Consumables" as special categories
3. These categories are referenced when creating products in RentalCore

**Note**: The category names should be configurable per the issue requirements.

---

## Testing Checklist

### Database
- [ ] Run migration on test database
- [ ] Verify all tables created
- [ ] Verify views work correctly
- [ ] Test inventory transactions trigger correctly

### API Endpoints
- [ ] Test all CRUD operations for accessories
- [ ] Test all CRUD operations for consumables
- [ ] Test scanning endpoints
- [ ] Test inventory adjustments
- [ ] Test low stock alerts

### Frontend (when implemented)
- [ ] Create accessory product
- [ ] Create consumable product
- [ ] Link accessories to product
- [ ] Link consumables to product
- [ ] Add product with accessories to job
- [ ] Add product with consumables to job
- [ ] View job with accessories/consumables
- [ ] Scan accessory out
- [ ] Scan consumable out with quantity
- [ ] Scan back in
- [ ] View inventory dashboard
- [ ] Adjust stock manually
- [ ] View transaction history

### Integration
- [ ] Test full workflow: create → assign → scan → return
- [ ] Verify stock quantities update correctly
- [ ] Verify transactions are logged
- [ ] Test low stock alerts appear when threshold reached

---

## Quick Start for Frontend Developers

### 1. Test the API

```bash
# Get count types
curl http://localhost:8081/api/count-types

# Create an accessory product (via products API)
curl -X POST http://localhost:8081/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Safety Cable 40kg",
    "is_accessory": true,
    "count_type_id": 1,
    "stock_quantity": 50,
    "min_stock_level": 10,
    "generic_barcode": "ACC-SAFE40",
    "price_per_unit": 2.50
  }'

# Link accessory to product
curl -X POST http://localhost:8081/api/products/100/accessories \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 100,
    "accessory_product_id": 500,
    "is_optional": true,
    "default_quantity": 1
  }'

# Assign to job
curl -X POST http://localhost:8081/api/jobs/10001/accessories \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": 10001,
    "parent_device_id": "DEV-001",
    "accessory_product_id": 500,
    "quantity_assigned": 4
  }'

# Scan out
curl -X POST http://localhost:8081/api/scan/accessory \
  -H "Content-Type: application/json" \
  -d '{
    "barcode": "ACC-SAFE40",
    "job_id": 10001,
    "direction": "out",
    "quantity": 1
  }'
```

### 2. Frontend Integration Points

The frontend needs to integrate at these points:
1. **Product Form**: Add accessory/consumable fields
2. **Product Edit**: Add accessories/consumables tabs
3. **Job Assignment**: Show accessory/consumable popups
4. **Job View**: Display accessories/consumables indented
5. **Inventory Page**: New page for stock management
6. **Scanning (WH)**: Handle generic barcodes with quantity prompts

### 3. UI/UX Considerations

- Accessories are optional during job assignment
- Consumables might be required (depending on product)
- Clear visual distinction between accessories and consumables
- Show stock availability when selecting
- Warn if adding more than available stock
- Use indentation or tree view for job display
- Color-code low stock items

---

## Database Schema Quick Reference

### Main Tables
- `count_types`: Measurement units (kg, pcs, L, etc.)
- `product_accessories`: Links products to accessories
- `product_consumables`: Links products to consumables
- `job_accessories`: Accessories assigned to jobs
- `job_consumables`: Consumables assigned to jobs
- `inventory_transactions`: Audit trail for stock changes

### Extended Products Table
New fields: `is_accessory`, `is_consumable`, `count_type_id`, `stock_quantity`, `min_stock_level`, `generic_barcode`, `price_per_unit`

### Views
- `vw_product_accessories`: Product accessories with details
- `vw_product_consumables`: Product consumables with details
- `vw_job_accessories_detail`: Job accessories with stock status
- `vw_job_consumables_detail`: Job consumables with stock status
- `vw_low_stock_alert`: Items below minimum stock

---

## Support

For questions or issues:
- Check API documentation: `docs/API_ACCESSORIES_CONSUMABLES.md`
- Review database migration: `migrations/037_accessories_consumables*.sql`
- Examine models: `internal/models/accessories_consumables.go`
- Review handlers: `internal/handlers/accessories_consumables_handler.go`
