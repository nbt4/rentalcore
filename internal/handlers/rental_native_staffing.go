package handlers

import (
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// Native requests retain the same identity/history as guided MCP writers. All
// locks, business writes and audit/history entries share one transaction.
func (h *JobHandler) changeNativeStaffing(c *gin.Context, job, employee int64, role *string, archive bool) {
	user, ok := GetCurrentUser(c)
	if !ok || user == nil || user.UserID == 0 {
		c.JSON(403, gin.H{"error": "Interactive user required"})
		return
	}
	if role != nil && (utf8.RuneCountInString(*role) > 100 || strings.ContainsRune(*role, 0)) {
		c.JSON(400, gin.H{"error": "Role exceeds100 characters"})
		return
	}
	db, e := h.jobRepo.GetDB().DB.DB()
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	tx, e := db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='20s';SET LOCAL TIME ZONE 'UTC';LOCK TABLE employees IN SHARE MODE NOWAIT;LOCK TABLE jobs,job_employees IN SHARE ROW EXCLUSIVE MODE NOWAIT`); e != nil {
		rentalCatalogError(c, e)
		return
	}
	var activeJob bool
	if e = tx.QueryRow(`SELECT deleted_at IS NULL FROM jobs WHERE jobid=$1 FOR UPDATE NOWAIT`, job).Scan(&activeJob); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if !activeJob {
		c.JSON(409, gin.H{"error": "Restore job before changing staffing"})
		return
	}
	before, e := rentalJobJSON(tx, `SELECT `+rentalStaffingRecordSQL+` FROM job_employees je WHERE job_id=$1 AND employee_id=$2 FOR UPDATE NOWAIT`, job, employee)
	if e != nil && e != sql.ErrNoRows {
		rentalCatalogError(c, e)
		return
	}
	if e == sql.ErrNoRows {
		before = nil
	}
	op := "create"
	if archive {
		if before == nil || before["is_archived"] == true {
			c.Status(204)
			return
		}
		op = "archive"
		_, e = tx.Exec(`UPDATE job_employees SET deleted_at=clock_timestamp() AT TIME ZONE 'UTC' WHERE job_id=$1 AND employee_id=$2`, job, employee)
	} else {
		var active bool
		if e = tx.QueryRow(`SELECT is_active FROM employees WHERE id=$1`, employee).Scan(&active); e != nil {
			rentalCatalogError(c, e)
			return
		}
		if !active {
			c.JSON(409, gin.H{"error": "Active employee required"})
			return
		}
		if before == nil {
			_, e = tx.Exec(`INSERT INTO job_employees(job_id,employee_id,role) VALUES($1,$2,$3)`, job, employee, role)
		} else if before["is_archived"] == true {
			op = "restore"
			_, e = tx.Exec(`UPDATE job_employees SET deleted_at=NULL WHERE job_id=$1 AND employee_id=$2`, job, employee)
			if e == nil && role != nil && !rentalCatalogEqual(before["role"], *role) {
				_, e = tx.Exec(`UPDATE job_employees SET role=$3 WHERE job_id=$1 AND employee_id=$2`, job, employee, role)
			}
		} else {
			c.JSON(201, gin.H{"ok": true})
			return
		}
	}
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	after, e := rentalJobJSON(tx, `SELECT `+rentalStaffingRecordSQL+` FROM job_employees je WHERE job_id=$1 AND employee_id=$2`, job, employee)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	if e = auditStaffing(tx, user.UserID, job, employee, op, "UI", before, after); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if _, e = tx.Exec(`UPDATE jobs SET updated_by=$2 WHERE jobid=$1`, job, user.UserID); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if e = tx.Commit(); e != nil {
		rentalCatalogError(c, e)
		return
	}
	(&RentalStaffingMCP{calendar: h.calendarSync}).syncCalendar(job, employee, op)
	if archive {
		c.Status(204)
	} else {
		c.JSON(201, gin.H{"ok": true})
	}
}
