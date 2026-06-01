# Employees, Skills & M365 Calendar Integration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mitarbeiter- und Skills-Verwaltung mit Admin-UI einführen, Mitarbeiter Jobs zuweisbar machen und bei jeder Job-Erstellung/-Änderung automatisch einen Termin im M365-Gruppenkalender `events@tsunami-events.de` anlegen/aktualisieren, der zugewiesene Mitarbeiter als erforderliche Teilnehmer enthält.

**Architecture:** Vier sequenzielle Schichten: (1) Skills-CRUD, (2) Employee-CRUD mit Skills-Zuweisung, (3) Job→Employee-Zuweisung, (4) M365-Kalender-Sync. Der bestehende `GraphClient` wird per Go-Embedding wiederverwendet — kein zweiter OAuth2-Flow. Kalender-Event-IDs werden in der `jobs`-Tabelle gespeichert, damit Updates/Deletes idempotent sind. Der `CalendarSyncService` wird dem `JobHandler` als Interface übergeben, damit er ohne M365-Config-Pflicht auch nil sein kann.

**Tech Stack:** Go 1.21 + Gin, GORM v2, PostgreSQL, Microsoft Graph API v1.0 (Calendars-Scope), React 18 + TypeScript, Tailwind CSS

---

## Dateiübersicht

### Neu erstellen
| Datei | Verantwortlichkeit |
|---|---|
| `migrations/041_skills_employees.sql` | Tabellen: `skills` (inkl. AV-Seed), `employees`, `employee_skills` |
| `migrations/042_job_employees.sql` | Tabelle: `job_employees`, Spalte `m365_event_id` auf `jobs` |
| `internal/models/employee.go` | Structs: `Skill`, `Employee`, `JobEmployee` |
| `internal/repository/skill_repository.go` | CRUD für Skills |
| `internal/repository/employee_repository.go` | CRUD für Employees inkl. Skills-Preload |
| `internal/repository/job_employee_repository.go` | Mitarbeiter einem Job zuweisen/entfernen |
| `internal/handlers/skill_handler.go` | HTTP-Handler für Skills-API |
| `internal/handlers/employee_handler.go` | HTTP-Handler für Employees-API |
| `internal/sync/m365/calendar_client.go` | Graph-API-Calls für Kalender-Events (Create/Update/Delete) |
| `internal/sync/m365/calendar_sync.go` | `CalendarSyncService` — baut Event-Body, ruft CalendarClient auf |
| `web/src/pages/SkillsPage.tsx` | Admin-Seite: Skills-Liste + Inline-Formular |
| `web/src/pages/EmployeesPage.tsx` | Admin-Seite: Mitarbeiter-Liste + Detailformular + Skills-Tags |

### Modifizieren
| Datei | Änderung |
|---|---|
| `internal/config/config.go` | `M365Config` um `CalendarMailbox` + `AppBaseURL` erweitern |
| `internal/models/models.go` | `Job`-Struct um `M365EventID *string` erweitern |
| `internal/handlers/job_handler.go` | Interface + Feld `calendarSync`; Sync nach Create/Update/Delete aufrufen; JobEmployee-Endpunkte |
| `cmd/server/main.go` | Repos/Handler/CalendarSyncService initialisieren; Routes registrieren |
| `web/src/App.tsx` | Routen `/admin/skills` + `/admin/employees` |
| `web/src/components/Layout.tsx` | Admin-Navigationsbereich mit Skills + Employees |
| `web/src/pages/JobsPage.tsx` | Bearbeiter-Zuweisung im Job-Detailbereich |
| `web/src/lib/api.ts` | Typen + API-Funktionen für Skills, Employees, JobEmployees |

---

## Task 1: Datenbankmigrationen

**Files:**
- Create: `rentalcore/migrations/041_skills_employees.sql`
- Create: `rentalcore/migrations/042_job_employees.sql`

- [ ] **Schritt 1: Migration 041 schreiben**

```sql
-- migrations/041_skills_employees.sql

CREATE TABLE skills (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT skills_name_unique UNIQUE (name)
);

CREATE TABLE employees (
    id           BIGSERIAL PRIMARY KEY,
    first_name   VARCHAR(100) NOT NULL,
    last_name    VARCHAR(100) NOT NULL,
    email        VARCHAR(255),
    phone        VARCHAR(50),
    mobile       VARCHAR(50),
    street       VARCHAR(255),
    house_number VARCHAR(20),
    zip          VARCHAR(20),
    city         VARCHAR(100),
    country      VARCHAR(100) NOT NULL DEFAULT 'Deutschland',
    date_of_birth DATE,
    iban         VARCHAR(50),
    notes        TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_email_unique UNIQUE (email)
);

CREATE TABLE employee_skills (
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id    BIGINT NOT NULL REFERENCES skills(id)    ON DELETE CASCADE,
    PRIMARY KEY (employee_id, skill_id)
);

-- Seed: Standard AV-Branche Skills
INSERT INTO skills (name, category) VALUES
  -- Audio
  ('FOH-Mischung',            'Audio'),
  ('Monitoring-Mischung',     'Audio'),
  ('PA-System',               'Audio'),
  ('Mikrofonierung',          'Audio'),
  ('Playback',                'Audio'),
  ('Intercom',                'Audio'),
  -- Licht
  ('Lichttechnik',            'Licht'),
  ('Moving Lights',           'Licht'),
  ('LED-Steuerung',           'Licht'),
  ('grandMA2',                'Licht'),
  ('grandMA3',                'Licht'),
  ('Haze / Fog',              'Licht'),
  ('Followspot',              'Licht'),
  -- Video
  ('Projektionstechnik',      'Video'),
  ('LED-Wall',                'Video'),
  ('Kameratechnik',           'Video'),
  ('Video-Switching',         'Video'),
  ('Live-Streaming',          'Video'),
  ('Screen-Management',       'Video'),
  -- Rigging
  ('Rigging',                 'Rigging'),
  ('Anschlagmittel',          'Rigging'),
  ('Traversensysteme',        'Rigging'),
  ('Flugplanung',             'Rigging'),
  -- Bühne
  ('Bühnenaufbau',            'Bühne'),
  ('Bühnenabbau',             'Bühne'),
  ('Traversenbau',            'Bühne'),
  ('Kabellegen',              'Bühne'),
  -- Projekt
  ('Projektmanagement',       'Projekt'),
  ('Technische Leitung',      'Projekt'),
  ('Veranstaltungsplanung',   'Projekt'),
  ('CAD-Planung',             'Projekt'),
  -- Fahrzeug / Logistik
  ('Führerschein Klasse B',   'Fahrzeug'),
  ('Führerschein Klasse BE',  'Fahrzeug'),
  ('Führerschein Klasse C',   'Fahrzeug'),
  ('Gabelstapler',            'Fahrzeug'),
  ('Hubarbeitsbühne',         'Fahrzeug');
```

- [ ] **Schritt 2: Migration 042 schreiben**

```sql
-- migrations/042_job_employees.sql

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS m365_event_id VARCHAR(255);

CREATE TABLE job_employees (
    job_id      BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    role        VARCHAR(100),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, employee_id)
);

CREATE INDEX idx_job_employees_job_id      ON job_employees(job_id);
CREATE INDEX idx_job_employees_employee_id ON job_employees(employee_id);
```

- [ ] **Schritt 3: Migrationen auf docker03 einspielen**

```bash
ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore" \
  < migrations/041_skills_employees.sql

ssh noah@docker03 "docker exec -i postgres psql -U rentalcore -d rentalcore" \
  < migrations/042_job_employees.sql
```

Erwartete Ausgabe:
```
CREATE TABLE
CREATE TABLE
CREATE TABLE
INSERT 0 37
ALTER TABLE
CREATE TABLE
CREATE INDEX
CREATE INDEX
```

- [ ] **Schritt 4: Commit**

```bash
git add migrations/041_skills_employees.sql migrations/042_job_employees.sql
git commit -m "feat: add skills, employees, job_employees tables and m365_event_id"
```

---

## Task 2: Go-Modelle

**Files:**
- Create: `internal/models/employee.go`
- Modify: `internal/models/models.go`

- [ ] **Schritt 1: employee.go erstellen**

```go
// internal/models/employee.go
package models

import "time"

type Skill struct {
	ID          uint      `json:"id"          gorm:"primaryKey"`
	Name        string    `json:"name"        gorm:"not null;uniqueIndex;size:100"`
	Category    string    `json:"category"    gorm:"not null;default:''"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Skill) TableName() string { return "skills" }

type Employee struct {
	ID          uint       `json:"id"           gorm:"primaryKey"`
	FirstName   string     `json:"first_name"   gorm:"not null;size:100"`
	LastName    string     `json:"last_name"    gorm:"not null;size:100"`
	Email       *string    `json:"email"        gorm:"uniqueIndex;size:255"`
	Phone       *string    `json:"phone"        gorm:"size:50"`
	Mobile      *string    `json:"mobile"       gorm:"size:50"`
	Street      *string    `json:"street"       gorm:"size:255"`
	HouseNumber *string    `json:"house_number" gorm:"size:20"`
	ZIP         *string    `json:"zip"          gorm:"size:20"`
	City        *string    `json:"city"         gorm:"size:100"`
	Country     *string    `json:"country"      gorm:"default:'Deutschland'"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	IBAN        *string    `json:"iban"         gorm:"size:50"`
	Notes       *string    `json:"notes"`
	IsActive    bool       `json:"is_active"    gorm:"default:true"`
	Skills      []Skill    `json:"skills"       gorm:"many2many:employee_skills;"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Employee) TableName() string { return "employees" }

func (e Employee) DisplayName() string {
	return e.FirstName + " " + e.LastName
}

type JobEmployee struct {
	JobID      uint      `json:"job_id"      gorm:"primaryKey"`
	EmployeeID uint      `json:"employee_id" gorm:"primaryKey"`
	Role       *string   `json:"role"        gorm:"size:100"`
	CreatedAt  time.Time `json:"created_at"`
	Employee   Employee  `json:"employee"    gorm:"foreignKey:EmployeeID"`
}

func (JobEmployee) TableName() string { return "job_employees" }
```

- [ ] **Schritt 2: `M365EventID` im Job-Struct ergänzen**

In `internal/models/models.go` die `Job`-Struct um ein Feld erweitern. Das Feld kommt direkt nach `MultiplyByDays bool`:

```go
M365EventID *string `json:"m365_event_id,omitempty" gorm:"column:m365_event_id;size:255"`
```

- [ ] **Schritt 3: Commit**

```bash
git add internal/models/employee.go internal/models/models.go
git commit -m "feat: add Skill, Employee, JobEmployee models and m365_event_id on Job"
```

---

## Task 3: Repositories

**Files:**
- Create: `internal/repository/skill_repository.go`
- Create: `internal/repository/employee_repository.go`
- Create: `internal/repository/job_employee_repository.go`

- [ ] **Schritt 1: SkillRepository**

```go
// internal/repository/skill_repository.go
package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type SkillRepository struct{ db *gorm.DB }

func NewSkillRepository(db *gorm.DB) *SkillRepository {
	return &SkillRepository{db: db}
}

func (r *SkillRepository) List() ([]models.Skill, error) {
	var skills []models.Skill
	err := r.db.Order("category, name").Find(&skills).Error
	return skills, err
}

func (r *SkillRepository) GetByID(id uint) (*models.Skill, error) {
	var s models.Skill
	err := r.db.First(&s, id).Error
	return &s, err
}

func (r *SkillRepository) Create(s *models.Skill) error {
	return r.db.Create(s).Error
}

func (r *SkillRepository) Update(s *models.Skill) error {
	return r.db.Save(s).Error
}

func (r *SkillRepository) Delete(id uint) error {
	return r.db.Delete(&models.Skill{}, id).Error
}
```

- [ ] **Schritt 2: EmployeeRepository**

```go
// internal/repository/employee_repository.go
package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type EmployeeRepository struct{ db *gorm.DB }

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) List() ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Preload("Skills").Order("last_name, first_name").Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) ListActive() ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Preload("Skills").Where("is_active = true").
		Order("last_name, first_name").Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetByID(id uint) (*models.Employee, error) {
	var e models.Employee
	err := r.db.Preload("Skills").First(&e, id).Error
	return &e, err
}

func (r *EmployeeRepository) Create(e *models.Employee, skillIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(e).Error; err != nil {
			return err
		}
		return r.replaceSkills(tx, e, skillIDs)
	})
}

func (r *EmployeeRepository) Update(e *models.Employee, skillIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(e).Error; err != nil {
			return err
		}
		return r.replaceSkills(tx, e, skillIDs)
	})
}

func (r *EmployeeRepository) Delete(id uint) error {
	return r.db.Delete(&models.Employee{}, id).Error
}

func (r *EmployeeRepository) replaceSkills(tx *gorm.DB, e *models.Employee, skillIDs []uint) error {
	if err := tx.Exec("DELETE FROM employee_skills WHERE employee_id = ?", e.ID).Error; err != nil {
		return err
	}
	if len(skillIDs) == 0 {
		return nil
	}
	var skills []models.Skill
	if err := tx.Where("id IN ?", skillIDs).Find(&skills).Error; err != nil {
		return err
	}
	return tx.Model(e).Association("Skills").Replace(skills)
}
```

- [ ] **Schritt 3: JobEmployeeRepository**

```go
// internal/repository/job_employee_repository.go
package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type JobEmployeeRepository struct{ db *gorm.DB }

func NewJobEmployeeRepository(db *gorm.DB) *JobEmployeeRepository {
	return &JobEmployeeRepository{db: db}
}

func (r *JobEmployeeRepository) ListForJob(jobID uint) ([]models.JobEmployee, error) {
	var je []models.JobEmployee
	err := r.db.Preload("Employee.Skills").Where("job_id = ?", jobID).Find(&je).Error
	return je, err
}

func (r *JobEmployeeRepository) Assign(jobID, employeeID uint, role *string) error {
	je := models.JobEmployee{JobID: jobID, EmployeeID: employeeID, Role: role}
	return r.db.Where(models.JobEmployee{JobID: jobID, EmployeeID: employeeID}).
		FirstOrCreate(&je).Error
}

func (r *JobEmployeeRepository) Remove(jobID, employeeID uint) error {
	return r.db.Where("job_id = ? AND employee_id = ?", jobID, employeeID).
		Delete(&models.JobEmployee{}).Error
}
```

- [ ] **Schritt 4: Commit**

```bash
git add internal/repository/skill_repository.go \
        internal/repository/employee_repository.go \
        internal/repository/job_employee_repository.go
git commit -m "feat: add skill, employee and job_employee repositories"
```

---

## Task 4: HTTP-Handler

**Files:**
- Create: `internal/handlers/skill_handler.go`
- Create: `internal/handlers/employee_handler.go`
- Modify: `internal/handlers/job_handler.go`

- [ ] **Schritt 1: SkillHandler**

```go
// internal/handlers/skill_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

type SkillHandler struct {
	repo *repository.SkillRepository
}

func NewSkillHandler(repo *repository.SkillRepository) *SkillHandler {
	return &SkillHandler{repo: repo}
}

func (h *SkillHandler) List(c *gin.Context) {
	skills, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, skills)
}

func (h *SkillHandler) Create(c *gin.Context) {
	var s models.Skill
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Create(&s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}

func (h *SkillHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	s, err := h.repo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.ID = uint(id)
	if err := h.repo.Update(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *SkillHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Schritt 2: EmployeeHandler**

```go
// internal/handlers/employee_handler.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

type EmployeeHandler struct {
	repo *repository.EmployeeRepository
}

func NewEmployeeHandler(repo *repository.EmployeeRepository) *EmployeeHandler {
	return &EmployeeHandler{repo: repo}
}

type employeeRequest struct {
	models.Employee
	SkillIDs []uint `json:"skill_ids"`
}

func (h *EmployeeHandler) List(c *gin.Context) {
	employees, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, employees)
}

func (h *EmployeeHandler) ListActive(c *gin.Context) {
	employees, err := h.repo.ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, employees)
}

func (h *EmployeeHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	e, err := h.repo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Create(&req.Employee, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req.Employee)
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Employee.ID = uint(id)
	if err := h.repo.Update(&req.Employee, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req.Employee)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Schritt 3: JobHandler erweitern — Interface + Felder**

In `internal/handlers/job_handler.go` direkt **über** der `JobHandler`-Struct-Definition einfügen:

```go
// CalendarSyncServiceInterface ermöglicht nil-sicheres Einbinden ohne M365-Pflicht.
type CalendarSyncServiceInterface interface {
	SyncJobEvent(jobID uint)
	DeleteJobEvent(jobID uint)
}
```

In der `JobHandler`-Struct folgende Felder ergänzen (nach den bestehenden Feldern):

```go
jobEmployeeRepo *repository.JobEmployeeRepository
calendarSync    CalendarSyncServiceInterface
```

In `NewJobHandler(...)` die Parameter-Liste um folgende zwei Einträge am Ende erweitern (vor der schließenden Klammer):

```go
jobEmployeeRepo *repository.JobEmployeeRepository,
calendarSync    CalendarSyncServiceInterface,
```

Im `return`-Block des `NewJobHandler` ergänzen:

```go
jobEmployeeRepo: jobEmployeeRepo,
calendarSync:    calendarSync,
```

- [ ] **Schritt 4: Kalender-Sync in CreateJob/CreateJobAPI aufrufen**

In `CreateJob` und `CreateJobAPI` jeweils direkt nach dem erfolgreichen DB-Create (nach dem `AfterCreate`-Hook-Aufruf, vor dem Response):

```go
if h.calendarSync != nil {
    go h.calendarSync.SyncJobEvent(job.JobID)
}
```

- [ ] **Schritt 5: Kalender-Sync in UpdateJob aufrufen**

In `UpdateJob` (und `UpdateJobAPI` falls vorhanden) direkt nach dem erfolgreichen `db.Save`:

```go
if h.calendarSync != nil {
    go h.calendarSync.SyncJobEvent(job.JobID)
}
```

- [ ] **Schritt 6: Kalender-Delete in DeleteJob aufrufen**

In `DeleteJob` direkt **vor** dem DB-Delete (damit m365_event_id noch verfügbar ist):

```go
if h.calendarSync != nil {
    h.calendarSync.DeleteJobEvent(job.JobID) // synchron, nicht goroutine
}
```

- [ ] **Schritt 7: Job-Employee-Endpunkte ans Ende von job_handler.go anhängen**

```go
func (h *JobHandler) ListJobEmployees(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	entries, err := h.jobEmployeeRepo.ListForJob(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *JobHandler) AssignEmployee(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	var body struct {
		EmployeeID uint    `json:"employee_id"`
		Role       *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.EmployeeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee_id required"})
		return
	}
	if err := h.jobEmployeeRepo.Assign(uint(jobID), body.EmployeeID, body.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(uint(jobID))
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *JobHandler) RemoveEmployee(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	employeeID, err := strconv.ParseUint(c.Param("employeeId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
		return
	}
	if err := h.jobEmployeeRepo.Remove(uint(jobID), uint(employeeID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(uint(jobID))
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Schritt 8: Commit**

```bash
git add internal/handlers/skill_handler.go \
        internal/handlers/employee_handler.go \
        internal/handlers/job_handler.go
git commit -m "feat: add skill/employee handlers and job-employee assignment endpoints"
```

---

## Task 5: M365 Kalender-Client

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/sync/m365/calendar_client.go`

- [ ] **Schritt 1: Config erweitern**

In `internal/config/config.go` in der `M365Config`-Struct nach `SyncInterval` ergänzen:

```go
CalendarMailbox string // M365_CALENDAR_MAILBOX (default: events@tsunami-events.de)
AppBaseURL      string // APP_BASE_URL (z.B. https://rentalcore.tsunami-events.de)
```

In der `LoadFromEnv()`-Methode ergänzen:

```go
if v := os.Getenv("M365_CALENDAR_MAILBOX"); v != "" {
    c.CalendarMailbox = v
} else {
    c.CalendarMailbox = "events@tsunami-events.de"
}
if v := os.Getenv("APP_BASE_URL"); v != "" {
    c.AppBaseURL = v
}
```

- [ ] **Schritt 2: CalendarClient schreiben**

```go
// internal/sync/m365/calendar_client.go
package m365

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CalendarEvent ist das Graph-API-Objekt für einen Kalendertermin.
type CalendarEvent struct {
	Subject   string        `json:"subject"`
	Body      EventBody     `json:"body"`
	Start     EventDateTime `json:"start"`
	End       EventDateTime `json:"end"`
	Attendees []Attendee    `json:"attendees,omitempty"`
}

type EventBody struct {
	ContentType string `json:"contentType"` // "HTML"
	Content     string `json:"content"`
}

type EventDateTime struct {
	DateTime string `json:"dateTime"` // "2026-05-14T00:00:00"
	TimeZone string `json:"timeZone"` // "Europe/Berlin"
}

type Attendee struct {
	EmailAddress EmailAddr `json:"emailAddress"`
	Type         string    `json:"type"` // "required"
}

type createdEventResponse struct {
	ID string `json:"id"`
}

// CalendarClient bettet GraphClient ein und verwendet seinen Token-Cache.
type CalendarClient struct {
	gc      *GraphClient
	mailbox string
}

func NewCalendarClient(gc *GraphClient, mailbox string) *CalendarClient {
	return &CalendarClient{gc: gc, mailbox: mailbox}
}

func (c *CalendarClient) CreateEvent(event CalendarEvent) (string, error) {
	resp, err := c.gc.doRequest("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events", c.mailbox),
		event,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create event HTTP %d: %s", resp.StatusCode, body)
	}
	var result createdEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}
	return result.ID, nil
}

func (c *CalendarClient) UpdateEvent(eventID string, event CalendarEvent) error {
	resp, err := c.gc.doRequest("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", c.mailbox, eventID),
		event,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update event HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (c *CalendarClient) DeleteEvent(eventID string) error {
	resp, err := c.gc.doRequest("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", c.mailbox, eventID),
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete event HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}
```

- [ ] **Schritt 3: Commit**

```bash
git add internal/sync/m365/calendar_client.go internal/config/config.go
git commit -m "feat: add M365 CalendarClient and CalendarMailbox config"
```

---

## Task 6: CalendarSyncService

**Files:**
- Create: `internal/sync/m365/calendar_sync.go`

- [ ] **Schritt 1: CalendarSyncService schreiben**

```go
// internal/sync/m365/calendar_sync.go
package m365

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"gorm.io/gorm"
)

type CalendarSyncService struct {
	client  *CalendarClient
	jobRepo *repository.JobRepository
	posRepo *repository.PositionRepository
	empRepo *repository.JobEmployeeRepository
	db      *gorm.DB
	baseURL string
}

func NewCalendarSyncService(
	client *CalendarClient,
	jobRepo *repository.JobRepository,
	posRepo *repository.PositionRepository,
	empRepo *repository.JobEmployeeRepository,
	db *gorm.DB,
	baseURL string,
) *CalendarSyncService {
	return &CalendarSyncService{
		client:  client,
		jobRepo: jobRepo,
		posRepo: posRepo,
		empRepo: empRepo,
		db:      db,
		baseURL: baseURL,
	}
}

// SyncJobEvent erstellt oder aktualisiert den Kalendertermin für einen Job.
// Wird als goroutine aufgerufen — nie direkt Result prüfen.
func (s *CalendarSyncService) SyncJobEvent(jobID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		log.Printf("[CalendarSync] job %d not found: %v", jobID, err)
		return
	}
	if job.StartDate == nil {
		log.Printf("[CalendarSync] job %d has no start date, skipping", jobID)
		return
	}

	event, err := s.buildEvent(job)
	if err != nil {
		log.Printf("[CalendarSync] build event for job %d: %v", jobID, err)
		return
	}

	if job.M365EventID != nil && *job.M365EventID != "" {
		if err := s.client.UpdateEvent(*job.M365EventID, *event); err != nil {
			log.Printf("[CalendarSync] update event for job %d: %v", jobID, err)
		}
		return
	}

	eventID, err := s.client.CreateEvent(*event)
	if err != nil {
		log.Printf("[CalendarSync] create event for job %d: %v", jobID, err)
		return
	}
	if err := s.db.Model(&models.Job{}).Where("jobid = ?", jobID).
		Update("m365_event_id", eventID).Error; err != nil {
		log.Printf("[CalendarSync] save event id for job %d: %v", jobID, err)
	}
}

// DeleteJobEvent löscht den Kalendertermin und leert m365_event_id.
func (s *CalendarSyncService) DeleteJobEvent(jobID uint) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil || job.M365EventID == nil || *job.M365EventID == "" {
		return
	}
	if err := s.client.DeleteEvent(*job.M365EventID); err != nil {
		log.Printf("[CalendarSync] delete event for job %d: %v", jobID, err)
		return
	}
	s.db.Model(&models.Job{}).Where("jobid = ?", jobID).Update("m365_event_id", nil)
}

func (s *CalendarSyncService) buildEvent(job *models.Job) (*CalendarEvent, error) {
	positions, err := s.posRepo.ListForJob(job.JobID)
	if err != nil {
		return nil, fmt.Errorf("load positions: %w", err)
	}
	jEmployees, err := s.empRepo.ListForJob(job.JobID)
	if err != nil {
		return nil, fmt.Errorf("load employees: %w", err)
	}

	desc := ""
	if job.Description != nil {
		desc = *job.Description
	}
	customerName := job.Customer.GetDisplayName()
	subject := fmt.Sprintf("%s - %s (%s)", desc, customerName, job.JobCode)

	body := s.buildBody(positions, job.JobID)

	start := job.StartDate.Format("2006-01-02") + "T00:00:00"
	var end string
	if job.EndDate != nil {
		end = job.EndDate.Add(24 * time.Hour).Format("2006-01-02") + "T00:00:00"
	} else {
		end = job.StartDate.Add(24 * time.Hour).Format("2006-01-02") + "T00:00:00"
	}

	var attendees []Attendee
	for _, je := range jEmployees {
		if je.Employee.Email != nil && *je.Employee.Email != "" {
			attendees = append(attendees, Attendee{
				EmailAddress: EmailAddr{Address: *je.Employee.Email},
				Type:         "required",
			})
		}
	}

	return &CalendarEvent{
		Subject:   subject,
		Body:      EventBody{ContentType: "HTML", Content: body},
		Start:     EventDateTime{DateTime: start, TimeZone: "Europe/Berlin"},
		End:       EventDateTime{DateTime: end, TimeZone: "Europe/Berlin"},
		Attendees: attendees,
	}, nil
}

func (s *CalendarSyncService) buildBody(positions []models.JobPosition, jobID uint) string {
	type section struct {
		label string
		types []string
	}
	sections := []section{
		{"Dienstleistungen", []string{"service"}},
		{"Produkte", []string{"product"}},
		{"Mietprodukte", []string{"rental"}},
	}

	var sb strings.Builder
	for i, sec := range sections {
		if i > 0 {
			sb.WriteString("<br>")
		}
		sb.WriteString(fmt.Sprintf("<b>%s</b><br>", sec.label))
		found := false
		for _, p := range positions {
			if sliceContains(sec.types, p.PositionType) {
				name := p.Description
				if name == "" && p.ProductID != nil {
					name = fmt.Sprintf("Produkt #%d", *p.ProductID)
				}
				qty := p.Quantity
				if qty == 0 {
					qty = 1
				}
				sb.WriteString(fmt.Sprintf("%s – %.0fx<br>", name, qty))
				found = true
			}
		}
		if !found {
			sb.WriteString("–<br>")
		}
	}

	if s.baseURL != "" {
		sb.WriteString(fmt.Sprintf(
			"<br><a href=\"%s/jobs/%d\">Job in RentalCore öffnen</a>",
			s.baseURL, jobID,
		))
	}
	return sb.String()
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
```

- [ ] **Schritt 2: Commit**

```bash
git add internal/sync/m365/calendar_sync.go
git commit -m "feat: add CalendarSyncService with job event build and sync logic"
```

---

## Task 7: main.go — Initialisierung & Routes

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Schritt 1: Neue Repos initialisieren**

Im Bereich der Repo-Initialisierungen (nach `customerRepo :=`) ergänzen:

```go
skillRepo       := repository.NewSkillRepository(db)
employeeRepo    := repository.NewEmployeeRepository(db)
jobEmployeeRepo := repository.NewJobEmployeeRepository(db)
```

- [ ] **Schritt 2: PositionRepository-Variable prüfen**

Sicherstellen, dass `positionRepo` bereits initialisiert ist (für `CalendarSyncService`). Falls die Variable anders heißt (z.B. `posRepo`), den korrekten Namen im nächsten Schritt verwenden.

```bash
grep -n "PositionRepository\|positionRepo\|posRepo" cmd/server/main.go | head -10
```

- [ ] **Schritt 3: CalendarSyncService initialisieren**

Im `if cfg.M365.IsConfigured()` Block direkt nach dem bestehenden `svc := m365sync.NewSyncService(...)`:

```go
var calendarSync handlers.CalendarSyncServiceInterface
if cfg.M365.CalendarMailbox != "" {
    calendarClient := m365sync.NewCalendarClient(graphClient, cfg.M365.CalendarMailbox)
    calendarSync = m365sync.NewCalendarSyncService(
        calendarClient,
        jobRepo,
        positionRepo,      // ggf. posRepo — aus Schritt 2 prüfen
        jobEmployeeRepo,
        db,
        cfg.M365.AppBaseURL,
    )
    log.Printf("M365 calendar sync: initialized for %s", cfg.M365.CalendarMailbox)
}
```

- [ ] **Schritt 4: Handler initialisieren**

```go
skillHandler    := handlers.NewSkillHandler(skillRepo)
employeeHandler := handlers.NewEmployeeHandler(employeeRepo)
```

- [ ] **Schritt 5: NewJobHandler-Aufruf anpassen**

Den bestehenden `handlers.NewJobHandler(...)` Aufruf um die beiden neuen Parameter am Ende erweitern:

```go
jobHandler := handlers.NewJobHandler(
    // ... alle bisherigen Parameter unverändert ...,
    jobEmployeeRepo, // NEU
    calendarSync,    // NEU (nil wenn M365 nicht konfiguriert)
)
```

- [ ] **Schritt 6: API-Routen registrieren**

Im API-Routen-Block (`api := router.Group("/api/v1")`):

```go
// Skills
api.GET("/skills",        skillHandler.List)
api.POST("/skills",       skillHandler.Create)
api.PUT("/skills/:id",    skillHandler.Update)
api.DELETE("/skills/:id", skillHandler.Delete)

// Employees
api.GET("/employees",        employeeHandler.List)
api.GET("/employees/active", employeeHandler.ListActive)
api.GET("/employees/:id",    employeeHandler.GetByID)
api.POST("/employees",       employeeHandler.Create)
api.PUT("/employees/:id",    employeeHandler.Update)
api.DELETE("/employees/:id", employeeHandler.Delete)

// Job-Employee-Zuweisung
api.GET("/jobs/:id/employees",                jobHandler.ListJobEmployees)
api.POST("/jobs/:id/employees",               jobHandler.AssignEmployee)
api.DELETE("/jobs/:id/employees/:employeeId", jobHandler.RemoveEmployee)
```

- [ ] **Schritt 7: Backend kompilieren**

```bash
cd /opt/dev/cores/rentalcore
go build ./...
```

Alle Compile-Fehler beheben (fehlende Imports, falsche Parameterzahl). Erst wenn `go build ./...` fehlerfrei durchläuft, weitermachen.

- [ ] **Schritt 8: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire skill/employee/calendar handlers and routes in main.go"
```

---

## Task 8: Frontend — API-Typen, Navigation, Routen

**Files:**
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/components/Layout.tsx`
- Modify: `web/src/App.tsx`

- [ ] **Schritt 1: api.ts erweitern**

Am Ende von `web/src/lib/api.ts` anfügen:

```typescript
// ── Skills ───────────────────────────────────────────────────────────────
export interface Skill {
  id: number;
  name: string;
  category: string;
  description?: string;
}

export const fetchSkills = () =>
  api.get<Skill[]>('/skills').then(r => r.data);

export const createSkill = (data: Partial<Skill>) =>
  api.post<Skill>('/skills', data).then(r => r.data);

export const updateSkill = (id: number, data: Partial<Skill>) =>
  api.put<Skill>(`/skills/${id}`, data).then(r => r.data);

export const deleteSkill = (id: number) =>
  api.delete(`/skills/${id}`);

// ── Employees ────────────────────────────────────────────────────────────
export interface Employee {
  id: number;
  first_name: string;
  last_name: string;
  email?: string;
  phone?: string;
  mobile?: string;
  street?: string;
  house_number?: string;
  zip?: string;
  city?: string;
  country?: string;
  date_of_birth?: string;
  iban?: string;
  notes?: string;
  is_active: boolean;
  skills: Skill[];
}

export interface EmployeeRequest extends Omit<Employee, 'id' | 'skills'> {
  skill_ids: number[];
}

export const fetchEmployees = () =>
  api.get<Employee[]>('/employees').then(r => r.data);

export const fetchActiveEmployees = () =>
  api.get<Employee[]>('/employees/active').then(r => r.data);

export const fetchEmployee = (id: number) =>
  api.get<Employee>(`/employees/${id}`).then(r => r.data);

export const createEmployee = (data: EmployeeRequest) =>
  api.post<Employee>('/employees', data).then(r => r.data);

export const updateEmployee = (id: number, data: EmployeeRequest) =>
  api.put<Employee>(`/employees/${id}`, data).then(r => r.data);

export const deleteEmployee = (id: number) =>
  api.delete(`/employees/${id}`);

// ── Job-Employees ─────────────────────────────────────────────────────────
export interface JobEmployee {
  job_id: number;
  employee_id: number;
  role?: string;
  employee: Employee;
}

export const fetchJobEmployees = (jobId: number) =>
  api.get<JobEmployee[]>(`/jobs/${jobId}/employees`).then(r => r.data);

export const assignJobEmployee = (jobId: number, employeeId: number, role?: string) =>
  api.post(`/jobs/${jobId}/employees`, { employee_id: employeeId, role });

export const removeJobEmployee = (jobId: number, employeeId: number) =>
  api.delete(`/jobs/${jobId}/employees/${employeeId}`);
```

- [ ] **Schritt 2: Layout.tsx — Admin-Navigation**

In `web/src/components/Layout.tsx` den bestehenden Lucide-Import um `Users` und `Star` erweitern (falls noch nicht vorhanden):

```typescript
import { ..., Users, Star } from 'lucide-react';
```

Direkt nach dem bestehenden `navItems.map(...)` JSX-Block eine Admin-Sektion einfügen:

```tsx
{/* Admin */}
<div className="mt-6">
  <p className="px-3 mb-1 text-xs font-semibold uppercase tracking-wider text-gray-500">
    Administration
  </p>
  {[
    { path: '/admin/employees', icon: Users, label: 'Mitarbeiter' },
    { path: '/admin/skills',    icon: Star,  label: 'Skills'      },
  ].map(item => (
    <NavLink
      key={item.path}
      to={item.path}
      className={({ isActive }) =>
        `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
          isActive
            ? 'bg-blue-600 text-white'
            : 'text-gray-300 hover:bg-gray-700 hover:text-white'
        }`
      }
    >
      <item.icon size={18} />
      {item.label}
    </NavLink>
  ))}
</div>
```

- [ ] **Schritt 3: App.tsx — Routen registrieren**

```typescript
// Imports ergänzen:
import SkillsPage    from './pages/SkillsPage';
import EmployeesPage from './pages/EmployeesPage';

// Im <Routes>-Block innerhalb der Layout-Route ergänzen:
<Route path="/admin/skills"    element={<SkillsPage />}    />
<Route path="/admin/employees" element={<EmployeesPage />} />
```

- [ ] **Schritt 4: Commit**

```bash
git add web/src/lib/api.ts web/src/components/Layout.tsx web/src/App.tsx
git commit -m "feat: add skills/employee API types, admin nav and routes"
```

---

## Task 9: SkillsPage

**Files:**
- Create: `web/src/pages/SkillsPage.tsx`

- [ ] **Schritt 1: Datei erstellen**

```tsx
// web/src/pages/SkillsPage.tsx
import { useState, useEffect } from 'react';
import { Plus, Pencil, Trash2, X, Check } from 'lucide-react';
import { Skill, fetchSkills, createSkill, updateSkill, deleteSkill } from '../lib/api';

const CATEGORIES = [
  'Audio', 'Licht', 'Video', 'Rigging', 'Bühne', 'Projekt', 'Fahrzeug', 'Sonstiges',
];

export default function SkillsPage() {
  const [skills, setSkills]     = useState<Skill[]>([]);
  const [editing, setEditing]   = useState<Skill | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm]         = useState({ name: '', category: 'Audio', description: '' });

  useEffect(() => { load(); }, []);

  async function load() { setSkills(await fetchSkills()); }

  async function save() {
    if (!form.name.trim()) return;
    if (editing) {
      await updateSkill(editing.id, form);
    } else {
      await createSkill(form);
    }
    setShowForm(false);
    setEditing(null);
    setForm({ name: '', category: 'Audio', description: '' });
    load();
  }

  async function remove(id: number) {
    if (!confirm('Skill wirklich löschen?')) return;
    await deleteSkill(id);
    load();
  }

  function startEdit(s: Skill) {
    setEditing(s);
    setForm({ name: s.name, category: s.category, description: s.description ?? '' });
    setShowForm(true);
  }

  const grouped = CATEGORIES.map(cat => ({
    cat,
    items: skills.filter(s => s.category === cat),
  })).filter(g => g.items.length > 0);

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Skills</h1>
        <button
          onClick={() => {
            setEditing(null);
            setForm({ name: '', category: 'Audio', description: '' });
            setShowForm(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
        >
          <Plus size={16} /> Skill hinzufügen
        </button>
      </div>

      {showForm && (
        <div className="bg-gray-800 border border-gray-700 rounded-xl p-4 mb-6 space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-gray-400 mb-1">Name</label>
              <input
                className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                placeholder="z.B. FOH-Mischung"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">Kategorie</label>
              <select
                className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                value={form.category}
                onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
              >
                {CATEGORIES.map(c => <option key={c}>{c}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className="block text-xs text-gray-400 mb-1">Beschreibung (optional)</label>
            <input
              className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
              value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
            />
          </div>
          <div className="flex gap-2">
            <button onClick={save}
              className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white rounded text-sm">
              <Check size={14} /> Speichern
            </button>
            <button onClick={() => setShowForm(false)}
              className="flex items-center gap-1 px-3 py-1.5 bg-gray-600 hover:bg-gray-500 text-white rounded text-sm">
              <X size={14} /> Abbrechen
            </button>
          </div>
        </div>
      )}

      <div className="space-y-6">
        {grouped.map(({ cat, items }) => (
          <div key={cat}>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-gray-400 mb-2">{cat}</h2>
            <div className="bg-gray-800 rounded-xl border border-gray-700 divide-y divide-gray-700">
              {items.map(s => (
                <div key={s.id} className="flex items-center justify-between px-4 py-3">
                  <div>
                    <span className="text-white text-sm font-medium">{s.name}</span>
                    {s.description && (
                      <p className="text-gray-400 text-xs mt-0.5">{s.description}</p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => startEdit(s)}
                      className="p-1.5 text-gray-400 hover:text-blue-400 rounded">
                      <Pencil size={14} />
                    </button>
                    <button onClick={() => remove(s.id)}
                      className="p-1.5 text-gray-400 hover:text-red-400 rounded">
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Schritt 2: Commit**

```bash
git add web/src/pages/SkillsPage.tsx
git commit -m "feat: add SkillsPage admin UI"
```

---

## Task 10: EmployeesPage

**Files:**
- Create: `web/src/pages/EmployeesPage.tsx`

- [ ] **Schritt 1: Datei erstellen**

```tsx
// web/src/pages/EmployeesPage.tsx
import { useState, useEffect } from 'react';
import { Plus, Pencil, Trash2, X, Check, ChevronDown, ChevronUp } from 'lucide-react';
import {
  Employee, EmployeeRequest, Skill,
  fetchEmployees, fetchSkills, createEmployee, updateEmployee, deleteEmployee,
} from '../lib/api';

const EMPTY: EmployeeRequest = {
  first_name: '', last_name: '', email: undefined, phone: undefined, mobile: undefined,
  street: undefined, house_number: undefined, zip: undefined, city: undefined,
  country: 'Deutschland', date_of_birth: undefined, iban: undefined, notes: undefined,
  is_active: true, skill_ids: [],
};

export default function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [allSkills, setAllSkills] = useState<Skill[]>([]);
  const [showForm, setShowForm]   = useState(false);
  const [editId, setEditId]       = useState<number | null>(null);
  const [form, setForm]           = useState<EmployeeRequest>(EMPTY);
  const [expanded, setExpanded]   = useState<number | null>(null);

  useEffect(() => { load(); }, []);

  async function load() {
    const [emps, skills] = await Promise.all([fetchEmployees(), fetchSkills()]);
    setEmployees(emps);
    setAllSkills(skills);
  }

  async function save() {
    if (!form.first_name.trim() || !form.last_name.trim()) return;
    if (editId !== null) {
      await updateEmployee(editId, form);
    } else {
      await createEmployee(form);
    }
    setShowForm(false);
    setEditId(null);
    setForm(EMPTY);
    load();
  }

  async function remove(id: number) {
    if (!confirm('Mitarbeiter wirklich löschen?')) return;
    await deleteEmployee(id);
    load();
  }

  function startEdit(e: Employee) {
    setEditId(e.id);
    setForm({ ...e, skill_ids: e.skills.map(s => s.id) });
    setShowForm(true);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function toggleSkill(id: number) {
    setForm(f => ({
      ...f,
      skill_ids: f.skill_ids.includes(id)
        ? f.skill_ids.filter(s => s !== id)
        : [...f.skill_ids, id],
    }));
  }

  const byCategory = allSkills.reduce<Record<string, Skill[]>>((acc, s) => {
    (acc[s.category] ??= []).push(s);
    return acc;
  }, {});

  const TEXT_FIELDS: { key: keyof EmployeeRequest; label: string }[] = [
    { key: 'email',        label: 'E-Mail' },
    { key: 'phone',        label: 'Telefon' },
    { key: 'mobile',       label: 'Mobil' },
    { key: 'street',       label: 'Straße' },
    { key: 'house_number', label: 'Hausnummer' },
    { key: 'zip',          label: 'PLZ' },
    { key: 'city',         label: 'Stadt' },
    { key: 'country',      label: 'Land' },
    { key: 'iban',         label: 'IBAN' },
  ];

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Mitarbeiter</h1>
        <button
          onClick={() => { setEditId(null); setForm(EMPTY); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
        >
          <Plus size={16} /> Mitarbeiter anlegen
        </button>
      </div>

      {showForm && (
        <div className="bg-gray-800 border border-gray-700 rounded-xl p-5 mb-6 space-y-4">
          <h2 className="text-white font-semibold">
            {editId ? 'Mitarbeiter bearbeiten' : 'Neuer Mitarbeiter'}
          </h2>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-gray-400 mb-1">Vorname *</label>
              <input
                className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                value={form.first_name}
                onChange={e => setForm(f => ({ ...f, first_name: e.target.value }))}
              />
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">Nachname *</label>
              <input
                className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                value={form.last_name}
                onChange={e => setForm(f => ({ ...f, last_name: e.target.value }))}
              />
            </div>
            {TEXT_FIELDS.map(({ key, label }) => (
              <div key={key}>
                <label className="block text-xs text-gray-400 mb-1">{label}</label>
                <input
                  className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                  value={(form[key] as string) ?? ''}
                  onChange={e =>
                    setForm(f => ({ ...f, [key]: e.target.value || undefined }))
                  }
                />
              </div>
            ))}
            <div>
              <label className="block text-xs text-gray-400 mb-1">Geburtsdatum</label>
              <input
                type="date"
                className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm"
                value={form.date_of_birth ?? ''}
                onChange={e =>
                  setForm(f => ({ ...f, date_of_birth: e.target.value || undefined }))
                }
              />
            </div>
          </div>

          <div>
            <label className="block text-xs text-gray-400 mb-1">Notizen</label>
            <textarea
              rows={2}
              className="w-full bg-gray-700 text-white rounded px-3 py-2 text-sm resize-none"
              value={form.notes ?? ''}
              onChange={e => setForm(f => ({ ...f, notes: e.target.value || undefined }))}
            />
          </div>

          <div>
            <label className="block text-xs text-gray-400 mb-2">Skills</label>
            <div className="space-y-2">
              {Object.entries(byCategory).map(([cat, catSkills]) => (
                <div key={cat}>
                  <p className="text-xs text-gray-500 mb-1">{cat}</p>
                  <div className="flex flex-wrap gap-2">
                    {catSkills.map(s => {
                      const active = form.skill_ids.includes(s.id);
                      return (
                        <button
                          key={s.id}
                          type="button"
                          onClick={() => toggleSkill(s.id)}
                          className={`px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
                            active
                              ? 'bg-blue-600 text-white'
                              : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                          }`}
                        >
                          {s.name}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
            <input
              type="checkbox"
              checked={form.is_active}
              onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))}
              className="rounded"
            />
            Aktiv
          </label>

          <div className="flex gap-2">
            <button onClick={save}
              className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 text-white rounded text-sm">
              <Check size={14} /> Speichern
            </button>
            <button onClick={() => setShowForm(false)}
              className="flex items-center gap-1 px-3 py-1.5 bg-gray-600 hover:bg-gray-500 text-white rounded text-sm">
              <X size={14} /> Abbrechen
            </button>
          </div>
        </div>
      )}

      <div className="bg-gray-800 rounded-xl border border-gray-700 divide-y divide-gray-700">
        {employees.length === 0 && (
          <p className="text-gray-400 text-sm p-6 text-center">Noch keine Mitarbeiter angelegt.</p>
        )}
        {employees.map(e => (
          <div key={e.id}>
            <div
              className="flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-gray-750"
              onClick={() => setExpanded(expanded === e.id ? null : e.id)}
            >
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-blue-700 flex items-center justify-center text-xs font-bold text-white">
                  {e.first_name[0]}{e.last_name[0]}
                </div>
                <div>
                  <p className="text-white text-sm font-medium">
                    {e.first_name} {e.last_name}
                  </p>
                  <p className="text-gray-400 text-xs">{e.email ?? '—'}</p>
                </div>
                {!e.is_active && (
                  <span className="ml-2 px-2 py-0.5 bg-gray-700 text-gray-400 text-xs rounded-full">
                    inaktiv
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={ev => { ev.stopPropagation(); startEdit(e); }}
                  className="p-1.5 text-gray-400 hover:text-blue-400 rounded"
                >
                  <Pencil size={14} />
                </button>
                <button
                  onClick={ev => { ev.stopPropagation(); remove(e.id); }}
                  className="p-1.5 text-gray-400 hover:text-red-400 rounded"
                >
                  <Trash2 size={14} />
                </button>
                {expanded === e.id
                  ? <ChevronUp size={16} className="text-gray-400" />
                  : <ChevronDown size={16} className="text-gray-400" />
                }
              </div>
            </div>

            {expanded === e.id && (
              <div className="px-6 pb-4 space-y-1.5 bg-gray-750">
                {e.phone  && <p className="text-sm text-gray-300">📞 {e.phone}</p>}
                {e.mobile && <p className="text-sm text-gray-300">📱 {e.mobile}</p>}
                {(e.street || e.city) && (
                  <p className="text-sm text-gray-300">
                    📍 {e.street} {e.house_number}, {e.zip} {e.city}
                  </p>
                )}
                {e.skills.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 pt-1">
                    {e.skills.map(s => (
                      <span key={s.id}
                        className="px-2 py-0.5 bg-blue-900/50 text-blue-300 text-xs rounded-full">
                        {s.name}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Schritt 2: Commit**

```bash
git add web/src/pages/EmployeesPage.tsx
git commit -m "feat: add EmployeesPage with skills multi-select and accordion detail"
```

---

## Task 11: Bearbeiter-Zuweisung in JobsPage

**Files:**
- Modify: `web/src/pages/JobsPage.tsx`

- [ ] **Schritt 1: Imports ergänzen**

Am Ende der bestehenden Imports in `JobsPage.tsx`:

```typescript
import {
  JobEmployee, fetchJobEmployees, fetchActiveEmployees,
  assignJobEmployee, removeJobEmployee, Employee,
} from '../lib/api';
import { X as XIcon } from 'lucide-react';
```

- [ ] **Schritt 2: State hinzufügen**

Im Komponenten-Body, nach den bestehenden State-Deklarationen:

```typescript
const [jobEmployees, setJobEmployees]         = useState<JobEmployee[]>([]);
const [activeEmployees, setActiveEmployees]   = useState<Employee[]>([]);
const [employeeSelectId, setEmployeeSelectId] = useState<number>(0);
```

- [ ] **Schritt 3: Effekt zum Laden der Job-Bearbeiter**

Nach dem bestehenden `useEffect` für den ausgewählten Job:

```typescript
useEffect(() => {
  if (!selectedJob) return;
  fetchJobEmployees(selectedJob.JobID).then(setJobEmployees);
  fetchActiveEmployees().then(setActiveEmployees);
}, [selectedJob?.JobID]);
```

- [ ] **Schritt 4: Handler-Funktionen**

```typescript
async function handleAssignEmployee() {
  if (!selectedJob || !employeeSelectId) return;
  await assignJobEmployee(selectedJob.JobID, employeeSelectId);
  setJobEmployees(await fetchJobEmployees(selectedJob.JobID));
  setEmployeeSelectId(0);
}

async function handleRemoveEmployee(employeeId: number) {
  if (!selectedJob) return;
  await removeJobEmployee(selectedJob.JobID, employeeId);
  setJobEmployees(await fetchJobEmployees(selectedJob.JobID));
}
```

- [ ] **Schritt 5: Bearbeiter-Panel im Job-Detail JSX**

Im JSX des Job-Detailbereichs, nach dem Positions-Panel (nach `<JobPositionsPanel .../>` bzw. dem letzten bestehenden Panel):

```tsx
{/* Bearbeiter */}
<div className="bg-gray-800 rounded-xl border border-gray-700 p-4">
  <h3 className="text-white font-semibold mb-3">Bearbeiter</h3>

  <div className="flex gap-2 mb-3">
    <select
      className="flex-1 bg-gray-700 text-white rounded px-3 py-2 text-sm"
      value={employeeSelectId}
      onChange={e => setEmployeeSelectId(Number(e.target.value))}
    >
      <option value={0}>Mitarbeiter auswählen…</option>
      {activeEmployees
        .filter(emp => !jobEmployees.some(je => je.employee_id === emp.id))
        .map(emp => (
          <option key={emp.id} value={emp.id}>
            {emp.first_name} {emp.last_name}
          </option>
        ))
      }
    </select>
    <button
      onClick={handleAssignEmployee}
      disabled={!employeeSelectId}
      className="px-3 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-40 text-white rounded text-sm"
    >
      Zuweisen
    </button>
  </div>

  {jobEmployees.length === 0 ? (
    <p className="text-gray-400 text-sm">Noch keine Bearbeiter zugewiesen.</p>
  ) : (
    <ul className="space-y-2">
      {jobEmployees.map(je => (
        <li key={je.employee_id}
          className="flex items-center justify-between bg-gray-700 rounded-lg px-3 py-2">
          <div>
            <p className="text-white text-sm">
              {je.employee.first_name} {je.employee.last_name}
            </p>
            {je.employee.email && (
              <p className="text-gray-400 text-xs">{je.employee.email}</p>
            )}
          </div>
          <button
            onClick={() => handleRemoveEmployee(je.employee_id)}
            className="text-gray-400 hover:text-red-400 p-1"
          >
            <XIcon size={14} />
          </button>
        </li>
      ))}
    </ul>
  )}
</div>
```

- [ ] **Schritt 6: Commit**

```bash
git add web/src/pages/JobsPage.tsx
git commit -m "feat: add employee assignment panel to job detail view"
```

---

## Task 12: Frontend bauen & Docker deployen

**Files:**
- Modify: `README.md`

- [ ] **Schritt 1: Frontend bauen**

```bash
cd /opt/dev/cores/rentalcore/web
npm run build
```

Erwartete Ausgabe: `dist/` befüllt, keine TypeScript-Fehler. Bei Fehlern beheben und erneut bauen.

- [ ] **Schritt 2: Aktuelle Versionsnummer ermitteln**

```bash
grep -m1 -E "v[0-9]+\.[0-9]+\.[0-9]+" /opt/dev/cores/rentalcore/README.md
```

Nächste Patch-Version notieren (z.B. `v5.3.44` → `v5.3.45`).

- [ ] **Schritt 3: README-Versionsnummer aktualisieren und alles committen**

```bash
cd /opt/dev/cores/rentalcore
# Versionsnummer in README.md manuell auf neue Version setzen
git add web/dist README.md
git commit -m "chore: bump to vX.X.X, add employees/skills/calendar integration"
git push
```

- [ ] **Schritt 4: Docker-Image bauen und pushen**

```bash
docker build -t nobentie/rentalcore:X.X.X .
docker push nobentie/rentalcore:X.X.X
docker tag nobentie/rentalcore:X.X.X nobentie/rentalcore:latest
docker push nobentie/rentalcore:latest
```

- [ ] **Schritt 5: Produktions-Logs prüfen (read-only)**

```bash
ssh noah@docker03 "docker logs --tail=60 rentalcore 2>&1 | grep -E 'error|CalendarSync|skill|employee'"
```

Erwartetes Ergebnis: keine Fehler. `M365 calendar sync: initialized for events@tsunami-events.de` sollte erscheinen (wenn `M365_CALENDAR_MAILBOX` gesetzt ist).

---

## Spec-Abgleich (Selbstreview)

| Anforderung | Task |
|---|---|
| Termin bei Job-Anlage in `events@tsunami-events.de` | Task 5 + 6 + 7 |
| Titel: `[Beschreibung] - [Kunde] ([JobCode])` | Task 6 `buildEvent()` |
| Drei Abschnitte in Beschreibung (Dienstleistungen / Produkte / Mietprodukte) | Task 6 `buildBody()` |
| Format je Zeile: `[Name] – [Anzahl]x` | Task 6 `buildBody()` |
| Link zum Job am Ende der Beschreibung | Task 6 `buildBody()` |
| Termin aktualisieren bei Job-Änderung | Task 6 `SyncJobEvent` → UpdateEvent |
| Termin löschen bei Job-Löschung | Task 6 `DeleteJobEvent` |
| Bearbeiter einem Job zuweisen | Task 3 + 4 + 11 |
| Bearbeiter aus „Employees" auswählen | Task 3 + 10 |
| Bearbeiter als erforderliche Teilnehmer im Termin | Task 6 `buildEvent()` attendees |
| Bearbeiter-Zuweisung/-Entfernung löst Kalender-Update aus | Task 4 `AssignEmployee` / `RemoveEmployee` |
| Neuer Admin-Tab „Mitarbeiter" | Task 8 + 10 |
| Neuer Admin-Tab „Skills" | Task 8 + 9 |
| Skills auswählbar bei Mitarbeiter | Task 10 |
| Skills vorbelegt mit AV-Branche Standard | Task 1 Seed (37 Skills) |
| Kontaktfelder beim Mitarbeiter (Standard) | Task 2 `Employee`-Struct + Task 10 Form |
