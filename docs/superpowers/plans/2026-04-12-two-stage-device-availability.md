# Two-Stage Device Availability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fragile auto-assignment system with a two-stage model: (1) job creation stores product *requirements* (quantities only, no device assignment), (2) warehouse staff explicitly assigns specific devices in the job detail view with a simple date-overlap availability check.

**Architecture:** A new `job_product_requirements` table decouples planning from inventory. The job form stops calling `applyProductSelections` — it just saves `(job_id, product_id, quantity)` rows. The job detail page gains a `RequirementsPanel` showing fulfilment status per product, with a modal to pick an available device for each slot.

**Tech Stack:** Go/Gin, GORM, PostgreSQL 16, React 18/TypeScript, axios (baseURL `/api/v1`)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `migrations/postgresql/000_combined_init.sql` | Modify | Add `job_product_requirements` table + indexes |
| `internal/models/job_product_requirement.go` | Create | `JobProductRequirement` GORM model |
| `internal/repository/requirement_repository.go` | Create | `RequirementRepository` — save/fetch/delete requirements |
| `internal/repository/device_repository.go` | Modify | Add `IsDeviceAvailableForJob` method |
| `internal/handlers/job_handler.go` | Modify | Wire `requirementRepo`, add `saveRequirements`, replace `applyProductSelections` in Create/Update, add `GetJobRequirementsAPI` + `GetAvailableDevicesForRequirementAPI`, add availability check to `AssignDeviceAPI`, remove `resolveProductSelections`/`applyProductSelections` |
| `cmd/server/main.go` | Modify | Init `RequirementRepository`, pass to `NewJobHandler`, register 2 new API routes |
| `web/src/pages/JobsPage.tsx` | Modify | Add `RequirementsPanel` + `DeviceAssignModal` components; wire into `JobDetail`; remove quantity cap from `ProductPicker` |

---

### Task 1: DB — add job_product_requirements table

**Files:**
- Modify: `migrations/postgresql/000_combined_init.sql`

- [ ] **Step 1: Add table + indexes to 000_combined_init.sql**

Open `migrations/postgresql/000_combined_init.sql`. Find the "PART 4b" section (after `job_history` table). Add right after the last CREATE TABLE in that section, before the PART 5 indexes comment:

```sql
-- job_product_requirements: stores what products a job needs (stage 1 of two-stage availability)
CREATE TABLE IF NOT EXISTS job_product_requirements (
    id BIGSERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(productid) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT job_product_requirements_job_product_unique UNIQUE (job_id, product_id)
);
```

In the PART 5 indexes section, add:

```sql
CREATE INDEX IF NOT EXISTS idx_job_product_req_job     ON job_product_requirements(job_id);
CREATE INDEX IF NOT EXISTS idx_job_product_req_product ON job_product_requirements(product_id);
```

- [ ] **Step 2: Create the table on the live DB via SSH**

```bash
ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore -c \"
CREATE TABLE IF NOT EXISTS job_product_requirements (
    id BIGSERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(productid) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT job_product_requirements_job_product_unique UNIQUE (job_id, product_id)
);\""
ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore -c \"CREATE INDEX IF NOT EXISTS idx_job_product_req_job ON job_product_requirements(job_id);\""
ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore -c \"CREATE INDEX IF NOT EXISTS idx_job_product_req_product ON job_product_requirements(product_id);\""
```

Expected: `CREATE TABLE`, `CREATE INDEX`, `CREATE INDEX`

- [ ] **Step 3: Verify table exists**

```bash
ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore -c \"\\d job_product_requirements\""
```

Expected: table description with columns `id`, `job_id`, `product_id`, `quantity`, `created_at`.

- [ ] **Step 4: Commit**

```bash
cd /opt/dev/cores/rentalcore
git add migrations/postgresql/000_combined_init.sql
git commit -m "feat: add job_product_requirements table to schema"
```

---

### Task 2: Go model — JobProductRequirement

**Files:**
- Create: `internal/models/job_product_requirement.go`

- [ ] **Step 1: Create the model file**

```go
package models

import "time"

// JobProductRequirement records the required quantity of a product for a job.
// This is Stage 1 of the two-stage device availability model:
// the planner says "I need 3× GLXD4 for this job" without touching devices.
type JobProductRequirement struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`
	JobID     uint      `gorm:"column:job_id;not null;index" json:"job_id"`
	ProductID uint      `gorm:"column:product_id;not null" json:"product_id"`
	Quantity  int       `gorm:"column:quantity;not null;default:1" json:"quantity"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	Product   *Product  `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
}

func (JobProductRequirement) TableName() string {
	return "job_product_requirements"
}
```

- [ ] **Step 2: Verify the build compiles**

```bash
cd /opt/dev/cores/rentalcore
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/models/job_product_requirement.go
git commit -m "feat: add JobProductRequirement model"
```

---

### Task 3: RequirementRepository

**Files:**
- Create: `internal/repository/requirement_repository.go`

- [ ] **Step 1: Create the repository**

```go
package repository

import "go-barcode-webapp/internal/models"

type RequirementRepository struct {
	db *Database
}

func NewRequirementRepository(db *Database) *RequirementRepository {
	return &RequirementRepository{db: db}
}

// SaveRequirements replaces all requirements for the given job atomically.
// Pass an empty slice to clear all requirements.
func (r *RequirementRepository) SaveRequirements(jobID uint, reqs []models.JobProductRequirement) error {
	return r.db.Transaction(func(tx *Database) error {
		if err := tx.Where("job_id = ?", jobID).Delete(&models.JobProductRequirement{}).Error; err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		return tx.Create(&reqs).Error
	})
}

// GetByJobID returns all requirements for a job, with product preloaded.
func (r *RequirementRepository) GetByJobID(jobID uint) ([]models.JobProductRequirement, error) {
	var reqs []models.JobProductRequirement
	err := r.db.Where("job_id = ?", jobID).
		Preload("Product").
		Order("id ASC").
		Find(&reqs).Error
	return reqs, err
}
```

Note: `r.db.Transaction` receives `*gorm.DB`, not `*Database`. The `Database` type wraps `*gorm.DB`. Use the underlying gorm transaction instead:

```go
// SaveRequirements replaces all requirements for the given job atomically.
func (r *RequirementRepository) SaveRequirements(jobID uint, reqs []models.JobProductRequirement) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id = ?", jobID).Delete(&models.JobProductRequirement{}).Error; err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		return tx.Create(&reqs).Error
	})
}
```

Full file with correct imports:

```go
package repository

import (
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type RequirementRepository struct {
	db *Database
}

func NewRequirementRepository(db *Database) *RequirementRepository {
	return &RequirementRepository{db: db}
}

// SaveRequirements replaces all requirements for the given job atomically.
func (r *RequirementRepository) SaveRequirements(jobID uint, reqs []models.JobProductRequirement) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id = ?", jobID).Delete(&models.JobProductRequirement{}).Error; err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		return tx.Create(&reqs).Error
	})
}

// GetByJobID returns all requirements for a job, with product preloaded.
func (r *RequirementRepository) GetByJobID(jobID uint) ([]models.JobProductRequirement, error) {
	var reqs []models.JobProductRequirement
	err := r.db.Where("job_id = ?", jobID).
		Preload("Product").
		Order("id ASC").
		Find(&reqs).Error
	return reqs, err
}
```

- [ ] **Step 2: Build**

```bash
cd /opt/dev/cores/rentalcore
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/requirement_repository.go
git commit -m "feat: add RequirementRepository for job product requirements"
```

---

### Task 4: DeviceRepository — IsDeviceAvailableForJob

**Files:**
- Modify: `internal/repository/device_repository.go`

- [ ] **Step 1: Add the method at the end of the file (before the last closing brace)**

Find the bottom of `device_repository.go` (after the last function). Add:

```go
// IsDeviceAvailableForJob returns true if the device has no confirmed job overlap
// in the given date range, excluding the specified job (use 0 to exclude none).
func (r *DeviceRepository) IsDeviceAvailableForJob(deviceID string, excludeJobID uint, start, end time.Time) (bool, error) {
	var count int64
	q := r.db.Table("job_devices jd").
		Joins("JOIN jobs j ON jd.jobid = j.jobid").
		Where("jd.deviceid = ?", deviceID).
		Where("NOT (COALESCE(j.enddate, j.startdate) < ? OR j.startdate > ?)", end, start)
	if excludeJobID != 0 {
		q = q.Where("j.jobid != ?", excludeJobID)
	}
	err := q.Count(&count).Error
	return count == 0, err
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/device_repository.go
git commit -m "feat: add IsDeviceAvailableForJob to DeviceRepository"
```

---

### Task 5: Handler — wire requirementRepo, saveRequirements helper

**Files:**
- Modify: `internal/handlers/job_handler.go`

This task wires the repository into the handler struct and adds the helper. The next task removes the old auto-assign logic.

- [ ] **Step 1: Add `requirementRepo` field to `JobHandler` struct (around line 114)**

Find:
```go
type JobHandler struct {
	jobRepo            *repository.JobRepository
	jobPackageRepo     *repository.JobPackageRepository
	deviceRepo         *repository.DeviceRepository
	customerRepo       *repository.CustomerRepository
	statusRepo         *repository.StatusRepository
	jobCategoryRepo    *repository.JobCategoryRepository
	jobEditSessionRepo *repository.JobEditSessionRepository
	jobHistoryService  *services.JobHistoryService
	rentalEquipRepo    *repository.RentalEquipmentRepository
	warehouseClient    *warehousecore.Client
}
```

Replace with:
```go
type JobHandler struct {
	jobRepo            *repository.JobRepository
	jobPackageRepo     *repository.JobPackageRepository
	deviceRepo         *repository.DeviceRepository
	requirementRepo    *repository.RequirementRepository
	customerRepo       *repository.CustomerRepository
	statusRepo         *repository.StatusRepository
	jobCategoryRepo    *repository.JobCategoryRepository
	jobEditSessionRepo *repository.JobEditSessionRepository
	jobHistoryService  *services.JobHistoryService
	rentalEquipRepo    *repository.RentalEquipmentRepository
	warehouseClient    *warehousecore.Client
}
```

- [ ] **Step 2: Update `NewJobHandler` signature and body (around line 173)**

Find:
```go
func NewJobHandler(jobRepo *repository.JobRepository, jobPackageRepo *repository.JobPackageRepository, deviceRepo *repository.DeviceRepository, customerRepo *repository.CustomerRepository, statusRepo *repository.StatusRepository, jobCategoryRepo *repository.JobCategoryRepository, jobEditSessionRepo *repository.JobEditSessionRepository, jobHistoryService *services.JobHistoryService, rentalEquipRepo *repository.RentalEquipmentRepository) *JobHandler {
	return &JobHandler{
		jobRepo:            jobRepo,
		jobPackageRepo:     jobPackageRepo,
		deviceRepo:         deviceRepo,
		customerRepo:       customerRepo,
		statusRepo:         statusRepo,
		jobCategoryRepo:    jobCategoryRepo,
		jobEditSessionRepo: jobEditSessionRepo,
		jobHistoryService:  jobHistoryService,
		rentalEquipRepo:    rentalEquipRepo,
		warehouseClient:    warehousecore.NewClient(),
	}
}
```

Replace with:
```go
func NewJobHandler(jobRepo *repository.JobRepository, jobPackageRepo *repository.JobPackageRepository, deviceRepo *repository.DeviceRepository, requirementRepo *repository.RequirementRepository, customerRepo *repository.CustomerRepository, statusRepo *repository.StatusRepository, jobCategoryRepo *repository.JobCategoryRepository, jobEditSessionRepo *repository.JobEditSessionRepository, jobHistoryService *services.JobHistoryService, rentalEquipRepo *repository.RentalEquipmentRepository) *JobHandler {
	return &JobHandler{
		jobRepo:            jobRepo,
		jobPackageRepo:     jobPackageRepo,
		deviceRepo:         deviceRepo,
		requirementRepo:    requirementRepo,
		customerRepo:       customerRepo,
		statusRepo:         statusRepo,
		jobCategoryRepo:    jobCategoryRepo,
		jobEditSessionRepo: jobEditSessionRepo,
		jobHistoryService:  jobHistoryService,
		rentalEquipRepo:    rentalEquipRepo,
		warehouseClient:    warehousecore.NewClient(),
	}
}
```

- [ ] **Step 3: Add `saveRequirements` helper — add this after the `lookupProductLabel` function (around line 1071)**

```go
// saveRequirements converts product selections to JobProductRequirement rows
// and persists them via the requirement repository (replaces all existing).
func (h *JobHandler) saveRequirements(jobID uint, selections []JobProductSelection) error {
	reqs := make([]models.JobProductRequirement, 0, len(selections))
	for _, s := range selections {
		if s.Quantity <= 0 {
			continue
		}
		reqs = append(reqs, models.JobProductRequirement{
			JobID:     jobID,
			ProductID: s.ProductID,
			Quantity:  s.Quantity,
		})
	}
	return h.requirementRepo.SaveRequirements(jobID, reqs)
}
```

- [ ] **Step 4: Build (will fail until main.go is updated in Task 8)**

Skip build until after Task 8 (main.go will fail to compile because `NewJobHandler` signature changed).

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/job_handler.go
git commit -m "feat: wire RequirementRepository into JobHandler, add saveRequirements helper"
```

---

### Task 6: Handler — replace applyProductSelections in Create/Update, add new endpoints

**Files:**
- Modify: `internal/handlers/job_handler.go`

- [ ] **Step 1: Replace applyProductSelections in CreateJobAPI (around line 1195)**

Find:
```go
	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.applyProductSelections(&job, selections); err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, job)
```

Replace with:
```go
	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, job)
```

- [ ] **Step 2: Replace applyProductSelections in UpdateJobAPI (around line 1362)**

Find:
```go
	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.applyProductSelections(&job, selections); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, job)
```

Replace with:
```go
	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, job)
```

- [ ] **Step 3: Delete `resolveProductSelections` and `applyProductSelections` functions**

Delete the entire `resolveProductSelections` function (lines ~869–993) and the `applyProductSelections` function (lines ~995–1037) and the `ApplyProductSelections` exported wrapper (~1039–1041).

Also delete `lookupProductLabel` (~1044–1071) since it was only used by `resolveProductSelections`. And remove the `"sort"` import if it's now unused.

- [ ] **Step 4: Add availability check to AssignDeviceAPI**

Find the existing `AssignDeviceAPI`:
```go
func (h *JobHandler) AssignDeviceAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	var request struct {
		Price float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.jobRepo.AssignDevice(uint(jobID), deviceID, request.Price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device assigned successfully"})
}
```

Replace with:
```go
func (h *JobHandler) AssignDeviceAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	var request struct {
		Price float64 `json:"price"`
	}
	_ = c.ShouldBindJSON(&request)

	// Availability check — only when the job has a date range
	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.StartDate != nil && job.EndDate != nil {
		ok, err := h.deviceRepo.IsDeviceAvailableForJob(deviceID, uint(jobID), *job.StartDate, *job.EndDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusConflict, gin.H{"error": "Gerät ist im gewählten Zeitraum bereits vergeben"})
			return
		}
	}

	if err := h.jobRepo.AssignDevice(uint(jobID), deviceID, request.Price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device assigned successfully"})
}
```

- [ ] **Step 5: Add GetJobRequirementsAPI handler — add after AssignDeviceAPI**

```go
// GetJobRequirementsAPI returns the product requirements for a job,
// enriched with how many devices are currently assigned per product.
func (h *JobHandler) GetJobRequirementsAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	reqs, err := h.requirementRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jobDevices, _ := h.jobRepo.GetJobDevices(uint(jobID))
	assignedPerProduct := make(map[uint]int)
	for _, jd := range jobDevices {
		if jd.Device.ProductID != nil {
			assignedPerProduct[*jd.Device.ProductID]++
		}
	}

	type RequirementRow struct {
		models.JobProductRequirement
		AssignedCount int `json:"assigned_count"`
	}
	rows := make([]RequirementRow, 0, len(reqs))
	for _, r := range reqs {
		rows = append(rows, RequirementRow{
			JobProductRequirement: r,
			AssignedCount:         assignedPerProduct[r.ProductID],
		})
	}

	c.JSON(http.StatusOK, gin.H{"requirements": rows})
}
```

- [ ] **Step 6: Add GetAvailableDevicesForRequirementAPI handler — add after GetJobRequirementsAPI**

```go
// GetAvailableDevicesForRequirementAPI returns devices of a given product
// that are available for the job's date range.
func (h *JobHandler) GetAvailableDevicesForRequirementAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}
	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.StartDate == nil || job.EndDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job has no date range set"})
		return
	}

	availability, err := h.deviceRepo.GetProductAvailabilityForJob(uint(productID), &job.JobID, job.StartDate, job.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	available := make([]repository.ProductDeviceAvailability, 0)
	for _, d := range availability {
		if d.Available {
			available = append(available, d)
		}
	}

	c.JSON(http.StatusOK, gin.H{"devices": available})
}
```

- [ ] **Step 7: Commit (not yet building — wait for main.go fix in Task 7)**

```bash
git add internal/handlers/job_handler.go
git commit -m "feat: replace auto-assign with requirements, add availability endpoints"
```

---

### Task 7: Wire RequirementRepository in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Initialize RequirementRepository — add after existing repository inits (around line 374)**

Find:
```go
	jobRepo := repository.NewJobRepository(db)
```

Somewhere in that block (add it after the existing repo inits, e.g. after `deviceRepo := ...`):

```go
	requirementRepo := repository.NewRequirementRepository(db)
```

- [ ] **Step 2: Pass requirementRepo to NewJobHandler (line ~375)**

Find:
```go
	jobHandler := handlers.NewJobHandler(jobRepo, jobPackageRepo, deviceRepo, customerRepo, statusRepo, jobCategoryRepo, jobEditSessionRepo, jobHistoryService, rentalEquipmentRepo)
```

Replace with:
```go
	jobHandler := handlers.NewJobHandler(jobRepo, jobPackageRepo, deviceRepo, requirementRepo, customerRepo, statusRepo, jobCategoryRepo, jobEditSessionRepo, jobHistoryService, rentalEquipmentRepo)
```

- [ ] **Step 3: Register 2 new routes — add inside the `apiJobs` group (around line 1285)**

Find:
```go
			apiJobs.GET("/:id/devices", jobHandler.GetJobDevices)
			apiJobs.POST("/:id/devices/:deviceId", jobHandler.AssignDeviceAPI)
			apiJobs.PUT("/:id/devices/:deviceId", jobHandler.UpdateDevicePriceAPI)
			apiJobs.DELETE("/:id/devices/:deviceId", jobHandler.RemoveDeviceAPI)
```

Add after the devices lines:
```go
			apiJobs.GET("/:id/requirements", jobHandler.GetJobRequirementsAPI)
			apiJobs.GET("/:id/products/:product_id/available-devices", jobHandler.GetAvailableDevicesForRequirementAPI)
```

- [ ] **Step 4: Build**

```bash
cd /opt/dev/cores/rentalcore
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: register requirements routes, wire RequirementRepository"
```

---

### Task 8: Frontend — ProductPicker: remove quantity cap

**Files:**
- Modify: `web/src/pages/JobsPage.tsx`

Since requirements are just quantities (not auto-assigned devices), users should be able to specify any quantity — not capped to available stock.

- [ ] **Step 1: Remove the `max` attribute and availability cap in ProductPicker**

Find (around line 88 in JobsPage.tsx):
```tsx
              <input
                type="number" min={0} max={avail} value={cur}
                onChange={(e) => setQty((q) => ({ ...q, [p.id]: Math.min(avail, Math.max(0, Number(e.target.value))) }))}
```

Replace with:
```tsx
              <input
                type="number" min={0} value={cur}
                onChange={(e) => setQty((q) => ({ ...q, [p.id]: Math.max(0, Number(e.target.value)) }))}
```

- [ ] **Step 2: Remove the disabled guard on the "+" button that blocks when avail=0**

Find:
```tsx
            <button
              onClick={() => { if (cur > 0) onSelect(p.id, p.name, cur); }}
              disabled={cur === 0}
```

Replace with:
```tsx
            <button
              onClick={() => { if (cur > 0) onSelect(p.id, p.name, cur); }}
              disabled={cur === 0 || total === 0}
```

(Keep `total === 0` disabled — if there are no devices at all for this product, you can't plan for it either.)

- [ ] **Step 3: Remove the date-validation guard in JobForm.save() that blocks saving requirements without dates**

Find (around line 523):
```tsx
    if (selections.length > 0 && (!form.startDate || !form.endDate)) {
      setError('Bitte Zeitraum setzen bevor Produkte gespeichert werden können.');
      return;
    }
```

Delete those 4 lines. Requirements can be saved without dates (the job just won't have availability data until dates are set, which affects Stage 2, not Stage 1).

- [ ] **Step 4: Build frontend**

```bash
cd /opt/dev/cores/rentalcore
npm run build --prefix web
```

Expected: no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/JobsPage.tsx
git commit -m "feat: allow any requirement quantity, remove availability cap in job form"
```

---

### Task 9: Frontend — RequirementsPanel with device assignment modal

**Files:**
- Modify: `web/src/pages/JobsPage.tsx`

- [ ] **Step 1: Add TypeScript interfaces after the existing `ProductSelection` interface (around line 337)**

```tsx
interface Requirement {
  id: number;
  job_id: number;
  product_id: number;
  quantity: number;
  assigned_count: number;
  product?: { name: string; productid: number };
}

interface AvailableDevice {
  DeviceID: string;
  ProductID: number;
  Status: string;
  CaseID?: number;
  CaseName?: string;
  AssignedToJob: boolean;
  Available: boolean;
}
```

- [ ] **Step 2: Add RequirementsPanel component — add after the `DeviceList` component (after line ~222)**

```tsx
// ── Requirements Panel (Stage 2 device assignment) ────────────

function RequirementsPanel({ jobId, onDeviceAssigned }: { jobId: number; onDeviceAssigned: () => void }) {
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [assigning, setAssigning] = useState<number | null>(null); // product_id being assigned
  const [availableDevices, setAvailableDevices] = useState<AvailableDevice[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [assignError, setAssignError] = useState('');

  const load = useCallback(() => {
    api.get(`/jobs/${jobId}/requirements`)
      .then((r) => setRequirements(r.data.requirements || []))
      .catch(console.error);
  }, [jobId]);

  useEffect(() => { load(); }, [load]);

  const openAssignModal = async (productId: number) => {
    setAssigning(productId);
    setAssignError('');
    setLoadingDevices(true);
    try {
      const r = await api.get(`/jobs/${jobId}/products/${productId}/available-devices`);
      setAvailableDevices(r.data.devices || []);
    } catch {
      setAssignError('Fehler beim Laden der verfügbaren Geräte');
    } finally {
      setLoadingDevices(false);
    }
  };

  const assignDevice = async (deviceId: string) => {
    try {
      await api.post(`/jobs/${jobId}/devices/${deviceId}`, {});
      setAssigning(null);
      load();
      onDeviceAssigned();
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setAssignError(err.response?.data?.error || 'Zuweisung fehlgeschlagen');
    }
  };

  if (requirements.length === 0) return null;

  return (
    <div className="glass-dark rounded-xl border border-white/10 p-5">
      <div className="flex items-center gap-2 mb-4">
        <Package className="w-4 h-4 text-accent-red" />
        <h3 className="font-semibold text-white">Produktbedarf ({requirements.length})</h3>
      </div>
      <div className="space-y-2">
        {requirements.map((req) => {
          const done = req.assigned_count >= req.quantity;
          const productName = req.product?.name || `Produkt ${req.product_id}`;
          return (
            <div key={req.id} className="flex items-center justify-between px-4 py-3 bg-white/5 rounded-lg">
              <div className="flex items-center gap-3 min-w-0">
                <span className={`w-2 h-2 rounded-full flex-shrink-0 ${done ? 'bg-green-400' : 'bg-yellow-400'}`} />
                <span className="text-sm text-white truncate">{productName}</span>
              </div>
              <div className="flex items-center gap-3 flex-shrink-0">
                <span className={`text-xs px-2 py-0.5 rounded-full ${done ? 'bg-green-500/10 text-green-400' : 'bg-yellow-500/10 text-yellow-400'}`}>
                  {req.assigned_count}/{req.quantity}
                </span>
                {!done && (
                  <button
                    onClick={() => openAssignModal(req.product_id)}
                    className="px-3 py-1 text-xs bg-accent-red/80 hover:bg-accent-red text-white rounded-lg transition-colors"
                  >
                    Zuweisen
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Device assign modal */}
      {assigning !== null && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="glass-dark rounded-2xl border border-white/10 p-6 w-full max-w-md space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">Gerät zuweisen</h3>
              <button
                onClick={() => { setAssigning(null); setAssignError(''); }}
                className="p-1.5 hover:bg-white/10 rounded-lg"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            {assignError && <p className="text-sm text-red-400">{assignError}</p>}
            {loadingDevices ? (
              <div className="flex justify-center py-6">
                <div className="w-6 h-6 border-2 border-accent-red/20 border-t-accent-red rounded-full animate-spin" />
              </div>
            ) : availableDevices.length === 0 ? (
              <p className="text-sm text-gray-500 text-center py-4">
                Keine verfügbaren Geräte im gewählten Zeitraum.
              </p>
            ) : (
              <div className="space-y-1 max-h-64 overflow-y-auto">
                {availableDevices.map((d) => (
                  <button
                    key={d.DeviceID}
                    onClick={() => assignDevice(d.DeviceID)}
                    className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-white/10 rounded-lg text-left transition-colors"
                  >
                    <span className="text-sm font-mono text-white">{d.DeviceID}</span>
                    <span className="text-xs text-gray-400">{d.CaseName || 'Lose'}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Wire RequirementsPanel into JobDetail — add before DeviceList**

Find (around line 317):
```tsx
      <DeviceList devices={devices} jobId={id} onChanged={loadData} />
```

Replace with:
```tsx
      <RequirementsPanel jobId={id} onDeviceAssigned={loadData} />
      <DeviceList devices={devices} jobId={id} onChanged={loadData} />
```

- [ ] **Step 4: Build**

```bash
npm run build --prefix web
```

Expected: no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/JobsPage.tsx
git commit -m "feat: add RequirementsPanel with device assignment modal to job detail"
```

---

### Task 10: Build Docker image and push

**Files:**
- Modify: `README.md` — add version entry

- [ ] **Step 1: Check current version in README**

```bash
grep "v5\." README.md | head -5
```

The current latest is v5.3.27. New version: **v5.3.28**.

- [ ] **Step 2: Add version entry to README**

In `README.md`, find the version history section. Add at the top:

```
### **v5.3.28** - Feature: two-stage device availability — job form saves requirements, job detail assigns specific devices
```

- [ ] **Step 3: Git push to GitLab**

```bash
cd /opt/dev/cores/rentalcore
git push origin main
```

- [ ] **Step 4: Build Docker image**

```bash
docker build -t nobentie/rentalcore:5.3.28 .
```

- [ ] **Step 5: Push version tag and latest**

```bash
docker push nobentie/rentalcore:5.3.28
docker tag nobentie/rentalcore:5.3.28 nobentie/rentalcore:latest
docker push nobentie/rentalcore:latest
```

- [ ] **Step 6: Commit README**

```bash
git add README.md
git commit -m "docs: update README for v5.3.28"
git push origin main
```

---

## Self-Review

### Spec coverage

| Requirement | Task |
|-------------|------|
| Store product requirements (job_id, product_id, quantity) at job creation | Tasks 1–3, 5 |
| No auto-device-assignment at job creation | Task 6 (removes resolveProductSelections) |
| Warehouse staff assigns specific devices in preparation step | Task 9 (RequirementsPanel) |
| Simple availability check: is this device in another overlapping job? | Tasks 4, 6 (IsDeviceAvailableForJob + AssignDeviceAPI check) |
| Backend endpoint: GET requirements with assigned count | Task 6 (GetJobRequirementsAPI) |
| Backend endpoint: GET available devices for a product in job period | Task 6 (GetAvailableDevicesForRequirementAPI) |
| Frontend: job form no longer errors on "not enough devices" | Tasks 6 (removed resolveProductSelections), 8 (removed date guard) |
| Frontend: job detail shows fulfilment progress | Task 9 (RequirementsPanel with colored dots) |
| Frontend: warehouse staff can pick a device from modal | Task 9 (DeviceAssignModal inside RequirementsPanel) |

### Potential issues

1. **`sort` import** — `resolveProductSelections` used `"sort"`. After removing it in Task 6, check that `"sort"` isn't used elsewhere in `job_handler.go`. If unused, remove it from imports to avoid compile error.

2. **`lookupProductLabel` uses `h.jobRepo.GetProductName`** — if this method is used elsewhere, don't delete it. If only used by the deleted `resolveProductSelections`, delete it too.

3. **`models.JobProductRequirement` embedded in `RequirementRow`** — Go embedded struct with JSON will include all fields. Verify that `AssignedCount` doesn't conflict with any embedded field name. It doesn't — `JobProductRequirement` has no `AssignedCount` field.

4. **`normalizeProductSelections`** — verify this is still called somewhere (e.g. PDF mapping or packages). If only used by `applyProductSelections`, delete it. Check with `grep normalizeProductSelections internal/handlers/job_handler.go`.
