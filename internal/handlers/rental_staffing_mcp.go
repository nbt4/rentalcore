package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type RentalStaffingMCP struct {
	db       *sql.DB
	calendar CalendarSyncServiceInterface
}

func NewRentalStaffingMCP(db *repository.Database, calendar CalendarSyncServiceInterface) *RentalStaffingMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalStaffingMCP{db: sqlDB, calendar: calendar}
}

type rentalStaffingRequest struct {
	JobID                int64   `json:"job_id"`
	EmployeeID           int64   `json:"employee_id"`
	Role                 *string `json:"role,omitempty"`
	AllowOverlap         bool    `json:"allow_overlap,omitempty"`
	ExpectedUpdatedAt    string  `json:"expected_updated_at,omitempty"`
	ExpectedJobUpdatedAt string  `json:"expected_job_updated_at,omitempty"`
	ExpectedContext      string  `json:"expected_context,omitempty"`
	ConfirmationText     string  `json:"confirmation_text,omitempty"`
	ConfirmChange        bool    `json:"confirm_change,omitempty"`
	Preview              bool    `json:"preview,omitempty"`
}

const rentalStaffingRecordSQL = `jsonb_build_object('job_id',je.job_id,'employee_id',je.employee_id,'role',je.role,'created_at',je.created_at,'is_archived',je.deleted_at IS NOT NULL,'archived_at',je.deleted_at,'updated_at',to_char(je.updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`
const rentalStaffingJobSQL = `jsonb_build_object('job_id',j.jobid,'job_code',j.job_code,'description',j.description,'start_date',to_char(j.startdate,'YYYY-MM-DD'),'end_date',to_char(j.enddate,'YYYY-MM-DD'),'status_id',j.statusid,'status',s.status,'is_archived',j.deleted_at IS NOT NULL,'updated_at',to_char(j.updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func (h *RentalStaffingMCP) Change(c *gin.Context) {
	op := c.Param("operation")
	if !map[string]bool{"create": true, "update": true, "archive": true, "restore": true}[op] {
		c.JSON(404, gin.H{"error": "Unknown named staffing operation"})
		return
	}
	user, ok := GetCurrentUser(c)
	if !ok || user == nil || user.UserID == 0 {
		c.JSON(403, gin.H{"error": "Interactive rental user required"})
		return
	}
	cookie, e := c.Cookie("cores_token")
	secret := os.Getenv("CORES_JWT_SECRET")
	if e != nil || secret == "" {
		c.JSON(403, gin.H{"error": "Signed MCP action delegation required"})
		return
	}
	claims := struct {
		UserID uint   `json:"uid"`
		Scope  string `json:"mcp_scope"`
		jwt.RegisteredClaims
	}{}
	token, e := jwt.ParseWithClaims(cookie, &claims, func(*jwt.Token) (any, error) { return []byte(secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	action := op
	if op == "restore" {
		action = "archive"
	}
	if e != nil || !token.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user and selected rental action required"})
		return
	}
	raw, e := io.ReadAll(io.LimitReader(c.Request.Body, 32769))
	if e != nil || len(raw) > 32768 {
		c.JSON(400, gin.H{"error": "One bounded staffing object required"})
		return
	}
	var in rentalStaffingRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&in); e != nil || dec.Decode(new(any)) != io.EOF {
		c.JSON(400, gin.H{"error": "One closed typed staffing object required"})
		return
	}
	var supplied map[string]json.RawMessage
	if json.Unmarshal(raw, &supplied) != nil || supplied == nil {
		c.JSON(400, gin.H{"error": "Staffing object required"})
		return
	}
	for _, v := range supplied {
		if string(v) == "null" {
			c.JSON(400, gin.H{"error": "Omit role to preserve, use empty string to clear; null is invalid"})
			return
		}
	}
	if in.JobID < 1 || in.EmployeeID < 1 || in.JobID > math.MaxInt32 || in.EmployeeID > math.MaxInt32 || in.Role != nil && (utf8.RuneCountInString(*in.Role) > 100 || strings.ContainsRune(*in.Role, 0)) {
		c.JSON(400, gin.H{"error": "Exact bounded job/employee IDs and role up to100 characters required"})
		return
	}
	if in.Role != nil && (op == "archive" || op == "restore") || op == "update" && in.Role == nil || in.AllowOverlap && op != "create" && op != "restore" {
		c.JSON(400, gin.H{"error": "Update requires role; archive/restore preserve role; overlap acknowledgement is only for create/restore"})
		return
	}
	preview := in.Preview || !in.ConfirmChange
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='20s';SET LOCAL TIME ZONE 'UTC'`); e != nil {
		rentalCatalogError(c, e)
		return
	}
	var admin bool
	if e = tx.QueryRow(`SELECT is_active AND is_admin FROM users WHERE userid=$1 FOR SHARE`, user.UserID).Scan(&admin); e != nil || !admin {
		c.JSON(403, gin.H{"error": "Current rental administrator required"})
		return
	}
	receiptID := int64(0)
	if !preview {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if !rentalMasterKeyPattern.MatchString(key) || !rentalMasterContextPattern.MatchString(in.ExpectedContext) {
			c.JSON(428, gin.H{"error": "Idempotency and exact final context required"})
			return
		}
		kh := sha256.Sum256([]byte(key))
		request, _ := json.Marshal(in)
		rh := sha256.Sum256(request)
		keyHash, requestHash := hex.EncodeToString(kh[:]), hex.EncodeToString(rh[:])
		operation := "job.staffing." + op
		e = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, operation, keyHash, requestHash).Scan(&receiptID)
		if e == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if e = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, operation, keyHash).Scan(&previous, &response, &status); e != nil {
				rentalCatalogError(c, e)
				return
			}
			if previous != requestHash {
				c.JSON(409, gin.H{"error": "Idempotency key binds a different request"})
				return
			}
			c.Data(status, "application/json", response)
			return
		}
		if e != nil {
			rentalCatalogError(c, e)
			return
		}
	}
	// Child writers touch the parent. NOWAIT table and row locks prevent reverse-order waits.
	if _, e = tx.Exec(`LOCK TABLE employees,skills,employee_skills,status IN SHARE MODE NOWAIT;LOCK TABLE jobs,job_employees,job_edit_sessions IN SHARE ROW EXCLUSIVE MODE NOWAIT`); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if _, e = tx.Exec(`SELECT jobid FROM jobs WHERE jobid=$1 FOR UPDATE NOWAIT`, in.JobID); e != nil {
		rentalCatalogError(c, e)
		return
	}
	current, e := rentalJobJSON(tx, `SELECT `+rentalStaffingRecordSQL+` FROM job_employees je WHERE job_id=$1 AND employee_id=$2 FOR UPDATE NOWAIT`, in.JobID, in.EmployeeID)
	if e != nil && e != sql.ErrNoRows {
		rentalCatalogError(c, e)
		return
	}
	if e == sql.ErrNoRows {
		current = nil
	}
	job, e := rentalJobJSON(tx, `SELECT `+rentalStaffingJobSQL+` FROM jobs j JOIN status s ON s.statusid=j.statusid WHERE j.jobid=$1`, in.JobID)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	employee, e := rentalJobJSON(tx, `SELECT jsonb_build_object('employee_id',e.id,'name',trim(concat_ws(' ',e.first_name,e.last_name)),'is_active',e.is_active,'updated_at',e.updated_at,'skills',COALESCE((SELECT jsonb_agg(jsonb_build_object('skill_id',sk.id,'name',sk.name,'category',sk.category) ORDER BY sk.id) FROM employee_skills es JOIN skills sk ON sk.id=es.skill_id WHERE es.employee_id=e.id),'[]'::jsonb)) FROM employees e WHERE id=$1`, in.EmployeeID)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	required := []string{}
	if job["is_archived"] == true {
		required = append(required, "restore_job_first")
	}
	if op != "archive" && employee["is_active"] != true {
		required = append(required, "activate_employee_first")
	}
	if op == "create" && current != nil {
		required = append(required, "assignment_already_exists_use_update_or_restore")
	}
	if op != "create" && current == nil {
		c.JSON(404, gin.H{"error": "Staffing assignment not found"})
		return
	}
	if current != nil {
		if op == "restore" && current["is_archived"] != true {
			required = append(required, "already_active")
		}
		if op != "restore" && op != "create" && current["is_archived"] == true {
			required = append(required, "restore_assignment_first")
		}
	}
	for _, v := range []struct {
		provided string
		current  any
		key      string
	}{{in.ExpectedJobUpdatedAt, job["updated_at"], "expected_job_updated_at"}, {in.ExpectedUpdatedAt, current["updated_at"], "expected_updated_at"}} {
		if v.key == "expected_updated_at" && op == "create" {
			if v.provided != "" {
				required = append(required, v.key)
			}
			continue
		}
		if !preview && v.provided == "" || v.provided != "" && v.provided != v.current {
			required = append(required, v.key)
		}
	}
	draft := map[string]any{"job_id": in.JobID, "employee_id": in.EmployeeID, "role": nil, "is_archived": op == "archive"}
	if current != nil {
		for k, v := range current {
			draft[k] = v
		}
		draft["is_archived"] = op == "archive"
	}
	if in.Role != nil {
		draft["role"] = *in.Role
	}
	diff := map[string]any{}
	for _, k := range []string{"role", "is_archived"} {
		if !rentalCatalogEqual(current[k], draft[k]) {
			diff[k] = map[string]any{"before": current[k], "after": draft[k]}
		}
	}
	if op == "update" && len(diff) == 0 {
		required = append(required, "changed_fields")
	}
	var conflictRaw, editorRaw json.RawMessage
	if e = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(x) ORDER BY x.job_id),'[]'::jsonb) FROM (SELECT j.jobid AS job_id,j.job_code,to_char(j.startdate,'YYYY-MM-DD') AS start_date,to_char(j.enddate,'YYYY-MM-DD') AS end_date,to_char(j.updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') AS job_updated_at FROM job_employees je JOIN jobs j ON j.jobid=je.job_id JOIN status s ON s.statusid=j.statusid JOIN jobs target ON target.jobid=$1 WHERE je.employee_id=$2 AND je.deleted_at IS NULL AND j.jobid<>$1 AND j.deleted_at IS NULL AND NOT rental_master_job_is_closed(s.status) AND j.startdate<=COALESCE(target.enddate,target.startdate) AND COALESCE(j.enddate,j.startdate)>=target.startdate ORDER BY j.jobid LIMIT 101)x`, in.JobID, in.EmployeeID).Scan(&conflictRaw); e != nil {
		rentalCatalogError(c, e)
		return
	}
	conflicts := []any{}
	if e = json.Unmarshal(conflictRaw, &conflicts); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if (op == "create" || op == "restore") && len(conflicts) > 0 && !in.AllowOverlap {
		required = append(required, "review_conflicts_and_allow_overlap")
	}
	if len(conflicts) > 100 {
		required = append(required, "bounded_conflict_context")
	}
	if e = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(x) ORDER BY user_id),'[]'::jsonb) FROM (SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes' LIMIT 1001)x`, in.JobID, user.UserID).Scan(&editorRaw); e != nil {
		rentalCatalogError(c, e)
		return
	}
	editors := []any{}
	if e = json.Unmarshal(editorRaw, &editors); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if len(editors) > 0 {
		required = append(required, "active_editing_session")
	}
	effects := map[string]any{"calendar_sync_after_commit": h.calendar != nil, "calendar_delivery": "best_effort_after_commit_not_atomic_with_database", "job_dates_apply_to_assignment": true, "financial_changes": false, "allow_overlap": in.AllowOverlap}
	contextRaw, _ := json.Marshal(map[string]any{"operation": op, "current": current, "draft": draft, "job": job, "employee": employee, "conflicts": conflicts, "editors": editors, "effects": effects})
	sum := sha256.Sum256(contextRaw)
	fingerprint := hex.EncodeToString(sum[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL STAFFING %d/%d %s", strings.ToUpper(op), in.JobID, in.EmployeeID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact record/draft-bound confirmation required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "job": job, "employee": employee, "schedule_conflicts": conflicts, "active_editors": editors, "effects": effects, "expected_updated_at": current["updated_at"], "expected_job_updated_at": job["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	switch op {
	case "create":
		_, e = tx.Exec(`INSERT INTO job_employees(job_id,employee_id,role) VALUES($1,$2,$3)`, in.JobID, in.EmployeeID, in.Role)
	case "update":
		_, e = tx.Exec(`UPDATE job_employees SET role=$3 WHERE job_id=$1 AND employee_id=$2`, in.JobID, in.EmployeeID, in.Role)
	default:
		_, e = tx.Exec(`UPDATE job_employees SET deleted_at=CASE WHEN $3 THEN clock_timestamp() AT TIME ZONE 'UTC' ELSE NULL END WHERE job_id=$1 AND employee_id=$2`, in.JobID, in.EmployeeID, op == "archive")
	}
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	after, e := rentalJobJSON(tx, `SELECT `+rentalStaffingRecordSQL+` FROM job_employees je WHERE job_id=$1 AND employee_id=$2`, in.JobID, in.EmployeeID)
	if e != nil {
		rentalCatalogError(c, e)
		return
	}
	var jobVersion string
	if e = tx.QueryRow(`UPDATE jobs SET updated_by=$2 WHERE jobid=$1 RETURNING to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, in.JobID, user.UserID).Scan(&jobVersion); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if e = auditStaffing(tx, user.UserID, in.JobID, in.EmployeeID, op, "MCP/AI", current, after); e != nil {
		rentalCatalogError(c, e)
		return
	}
	status := 200
	if op == "create" {
		status = 201
	}
	response, _ := json.Marshal(map[string]any{"operation_status": map[string]string{"create": "created", "update": "updated", "archive": "archived", "restore": "restored"}[op], "assignment": after, "job_updated_at": jobVersion, "effects": effects})
	if _, e = tx.Exec(`UPDATE rental_mcp_mutation_receipts SET response=$2::jsonb,status_code=$3 WHERE id=$1`, receiptID, string(response), status); e != nil {
		rentalCatalogError(c, e)
		return
	}
	if e = tx.Commit(); e != nil {
		rentalCatalogError(c, e)
		return
	}
	h.syncCalendar(in.JobID, in.EmployeeID, op)
	c.Data(status, "application/json", response)
}
func auditStaffing(tx *sql.Tx, user uint, job, employee int64, op, origin string, before, after map[string]any) error {
	old, _ := json.Marshal(before)
	fresh, _ := json.Marshal(map[string]any{"origin": origin, "after": after})
	if _, e := tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_staffing',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user, "rental.staffing."+op, fmt.Sprintf("%d/%d", job, employee), string(old), string(fresh)); e != nil {
		return e
	}
	_, e := tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,'updated',$3,$4,$5,$6)`, job, user, "staffing."+op, string(old), string(fresh), origin+" retained staffing "+op)
	return e
}
func (h *RentalStaffingMCP) syncCalendar(job, employee int64, op string) {
	if h.calendar != nil {
		go func() {
			if op == "archive" {
				h.calendar.DeleteEmployeeEvent(uint(job), uint(employee))
			}
			h.calendar.SyncJobEvent(uint(job))
		}()
	}
}
