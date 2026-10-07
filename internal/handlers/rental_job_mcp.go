package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"go-barcode-webapp/internal/jobstatus"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rentalJobFields struct {
	Description      *string  `json:"description"`
	CustomerID       *int64   `json:"customer_id"`
	StatusID         *int64   `json:"status_id"`
	JobCategoryID    *int64   `json:"job_category_id"`
	VenueID          *int64   `json:"venue_id"`
	StartDate        *string  `json:"start_date"`
	EndDate          *string  `json:"end_date"`
	Revenue          *float64 `json:"revenue"`
	Discount         *float64 `json:"discount"`
	DiscountType     *string  `json:"discount_type"`
	MultiplyByDays   *bool    `json:"multiply_by_days"`
	PricesIncludeTax *bool    `json:"prices_include_tax"`
}
type rentalJobRequest struct {
	rentalJobFields
	JobID             int64    `json:"job_id"`
	ExpectedUpdatedAt string   `json:"expected_updated_at"`
	ExpectedContext   string   `json:"expected_context"`
	ClearFields       []string `json:"clear_fields"`
	AllowDuplicate    bool     `json:"allow_duplicate"`
	ConfirmChange     bool     `json:"confirm_change"`
	ConfirmationText  string   `json:"confirmation_text"`
	Preview           bool     `json:"preview"`
}
type RentalJobMCP struct{ db *sql.DB }

func NewRentalJobMCP(db *repository.Database) *RentalJobMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalJobMCP{db: sqlDB}
}

var rentalJobColumns = map[string]string{"description": "description", "customer_id": "customerid", "status_id": "statusid", "job_category_id": "jobcategoryid", "venue_id": "venue_id", "start_date": "startdate", "end_date": "enddate", "revenue": "revenue", "discount": "discount", "discount_type": "discount_type", "multiply_by_days": "multiply_by_days", "prices_include_tax": "prices_include_tax"}

const rentalJobRecordSQL = `jsonb_build_object('job_id',jobid,'jobID',jobid,'job_code',job_code,'revision',revision,'customer_id',customerid,'status_id',statusid,'job_category_id',jobcategoryid,'venue_id',venue_id,'description',description,'start_date',to_char(startdate,'YYYY-MM-DD'),'end_date',to_char(enddate,'YYYY-MM-DD'),'revenue',revenue,'final_revenue',final_revenue,'discount',discount,'discount_type',discount_type,'multiply_by_days',multiply_by_days,'prices_include_tax',prices_include_tax,'created_by',created_by,'updated_by',updated_by,'created_at',created_at,'is_archived',deleted_at IS NOT NULL,'archived_at',deleted_at,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func rentalJobJSON(tx *sql.Tx, query string, args ...any) (map[string]any, error) {
	var raw json.RawMessage
	if err := tx.QueryRow(query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	out := map[string]any{}
	err := json.Unmarshal(raw, &out)
	return out, err
}
func rentalJobNumber(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}
func rentalJobModel(draft map[string]any) (models.Job, error) {
	job := models.Job{CustomerID: uint(rentalJobNumber(draft["customer_id"])), StatusID: uint(rentalJobNumber(draft["status_id"])), Revenue: rentalJobNumber(draft["revenue"]), Discount: rentalJobNumber(draft["discount"])}
	if value, ok := draft["description"].(string); ok {
		value = strings.TrimSpace(value)
		job.Description = &value
	}
	job.DiscountType, _ = draft["discount_type"].(string)
	job.MultiplyByDays, _ = draft["multiply_by_days"].(bool)
	job.PricesIncludeTax, _ = draft["prices_include_tax"].(bool)
	for _, item := range []struct {
		key    string
		target **uint
	}{{"job_category_id", &job.JobCategoryID}, {"venue_id", &job.VenueID}} {
		if value := draft[item.key]; value != nil && rentalJobNumber(value) > 0 {
			id := uint(rentalJobNumber(value))
			*item.target = &id
		}
	}
	for _, item := range []struct {
		key    string
		target **time.Time
	}{{"start_date", &job.StartDate}, {"end_date", &job.EndDate}} {
		if value := draft[item.key]; value != nil && value != "" {
			s, ok := value.(string)
			if !ok {
				return job, fmt.Errorf("%s must be YYYY-MM-DD", item.key)
			}
			date, err := time.Parse("2006-01-02", s)
			if err != nil {
				return job, fmt.Errorf("%s must be YYYY-MM-DD", item.key)
			}
			*item.target = &date
		}
	}
	return job, nil
}

func (h *RentalJobMCP) Change(c *gin.Context) {
	op := c.Param("operation")
	if op == "external-equipment-create" {
		h.createExternalEquipment(c)
		return
	}
	if !map[string]bool{"create": true, "update": true, "archive": true, "restore": true}[op] {
		c.JSON(404, gin.H{"error": "Unknown named job operation"})
		return
	}
	user, ok := GetCurrentUser(c)
	if !ok || user == nil || user.UserID == 0 {
		c.JSON(403, gin.H{"error": "Interactive rental user required"})
		return
	}
	cookie, err := c.Cookie("cores_token")
	if err != nil {
		c.JSON(403, gin.H{"error": "Signed MCP action delegation required"})
		return
	}
	claims := struct {
		UserID    uint   `json:"uid"`
		Scope     string `json:"mcp_scope"`
		Financial bool   `json:"mcp_financial"`
		jwt.RegisteredClaims
	}{}
	tok, err := jwt.ParseWithClaims(cookie, &claims, func(*jwt.Token) (any, error) { return []byte(os.Getenv("CORES_JWT_SECRET")), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	action := op
	if op == "restore" {
		action = "archive"
	}
	if err != nil || !tok.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed user and selected rental action required"})
		return
	}
	var in rentalJobRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 65537))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 65536 || in.JobID < 0 || in.JobID > math.MaxInt32 || (op == "create" && in.JobID != 0) || (op != "create" && in.JobID == 0) {
		c.JSON(400, gin.H{"error": "One bounded typed object and canonical job ID required"})
		return
	}
	preview := in.Preview || !in.ConfirmChange
	raw, _ := json.Marshal(in.rentalJobFields)
	fields := map[string]any{}
	_ = json.Unmarshal(raw, &fields)
	for key, value := range fields {
		if value == nil {
			delete(fields, key)
			continue
		}
		if op == "archive" || op == "restore" {
			c.JSON(400, gin.H{"error": "Lifecycle preserves business fields"})
			return
		}
		if s, ok := value.(string); ok {
			fields[key] = strings.TrimSpace(s)
			if len([]rune(s)) > 10000 {
				c.JSON(400, gin.H{"error": "Job text exceeds field limit"})
				return
			}
		}
		if f, ok := value.(float64); ok {
			limit := 99999999.99
			if key == "revenue" {
				limit = 9999999999.99
			}
			if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || (!strings.HasSuffix(key, "_id") && (f > limit || math.Abs(f*100-math.Round(f*100)) > 0.0001)) {
				c.JSON(400, gin.H{"error": "Job numbers must be finite and in range"})
				return
			}
			if strings.HasSuffix(key, "_id") && (f != math.Trunc(f) || f > math.MaxInt32 || ((key == "customer_id" || key == "status_id") && f == 0)) {
				c.JSON(400, gin.H{"error": "Positive canonical reference ID required"})
				return
			}
		}
	}
	for _, key := range in.ClearFields {
		if !map[string]bool{"start_date": true, "end_date": true, "venue_id": true, "job_category_id": true}[key] || op == "archive" || op == "restore" {
			c.JSON(400, gin.H{"error": "Only nullable job dates/category/venue can be cleared during field updates"})
			return
		}
		if _, exists := fields[key]; exists {
			c.JSON(400, gin.H{"error": "A field cannot be supplied and cleared together"})
			return
		}
		fields[key] = nil
	}
	for _, key := range []string{"venue_id", "job_category_id"} {
		if fields[key] == float64(0) {
			fields[key] = nil
		}
	}
	for _, key := range []string{"start_date", "end_date"} {
		if fields[key] == "" {
			fields[key] = nil
		}
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		masterError(c, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='20s';SET LOCAL TIME ZONE 'UTC'`); err != nil {
		masterError(c, err)
		return
	}
	var admin bool
	if err = tx.QueryRow(`SELECT is_admin AND is_active FROM users WHERE userid=$1 FOR SHARE`, user.UserID).Scan(&admin); err != nil || !admin {
		c.JSON(403, gin.H{"error": "Current rental administrator rights required"})
		return
	}
	financialInput := false
	for _, key := range []string{"revenue", "discount", "discount_type", "multiply_by_days", "prices_include_tax"} {
		if _, exists := fields[key]; exists {
			financialInput = true
		}
	}
	if financialInput && !claims.Financial {
		c.JSON(403, gin.H{"error": "Selected rental financial scope required for commercial fields"})
		return
	}
	receiptID := int64(0)
	if !preview {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if !rentalMasterKeyPattern.MatchString(key) || !rentalMasterContextPattern.MatchString(in.ExpectedContext) {
			c.JSON(428, gin.H{"error": "Idempotency and exact final context required"})
			return
		}
		keyDigest := sha256.Sum256([]byte(key))
		requestBytes, _ := json.Marshal(in)
		requestDigest := sha256.Sum256(requestBytes)
		keyHash, requestHash := hex.EncodeToString(keyDigest[:]), hex.EncodeToString(requestDigest[:])
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, "job."+op, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, "job."+op, keyHash).Scan(&previous, &response, &status); err != nil {
				masterError(c, err)
				return
			}
			if previous != requestHash {
				c.JSON(409, gin.H{"error": "Idempotency payload conflict"})
				return
			}
			var saved struct {
				Effects struct {
					Financial bool `json:"requires_financial_scope"`
				} `json:"effects"`
			}
			if err = json.Unmarshal(response, &saved); err != nil {
				masterError(c, err)
				return
			}
			if saved.Effects.Financial && !claims.Financial {
				c.JSON(403, gin.H{"error": "Selected rental financial scope required for this saved commercial operation"})
				return
			}
			c.Data(status, "application/json", response)
			return
		}
		if err != nil {
			masterError(c, err)
			return
		}
	}
	if _, err = tx.Exec(`LOCK TABLE customers,venues,status,jobcategory IN SHARE MODE;LOCK TABLE jobs IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_devices,job_product_requirements,job_packages,job_positions,job_employees,job_rental_equipment,devices IN SHARE MODE`); err != nil {
		masterError(c, err)
		return
	}
	// Optional owner tables vary between standalone and umbrella deployments.
	// Lock their complete snapshot so insertions cannot evade final-context checks.
	for _, table := range []string{"cases", "warehouse_tasks", "job_edit_sessions", "job_package_reservations", "job_position_devices", "products", "product_packages", "rental_equipment", "employees"} {
		var exists bool
		if err = tx.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			masterError(c, err)
			return
		}
		if exists {
			if _, err = tx.Exec("LOCK TABLE " + table + " IN SHARE MODE"); err != nil {
				masterError(c, err)
				return
			}
		}
	}
	current := map[string]any{}
	draft := map[string]any{}
	required := []string{}
	if op != "create" {
		current, err = rentalJobJSON(tx, "SELECT "+rentalJobRecordSQL+" FROM jobs WHERE jobid=$1", in.JobID)
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Job not found"})
			return
		}
		if err != nil {
			masterError(c, err)
			return
		}
		for key, value := range current {
			draft[key] = value
		}
		archived := current["is_archived"] == true
		if (op == "restore" && !archived) || (op != "restore" && archived) {
			required = append(required, "lifecycle_state")
		}
		if (!preview && in.ExpectedUpdatedAt == "") || (in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != current["updated_at"]) {
			required = append(required, "expected_updated_at")
		}
	} else {
		draft = map[string]any{"description": nil, "customer_id": nil, "status_id": float64(jobstatus.PlanningID), "job_category_id": nil, "venue_id": nil, "start_date": nil, "end_date": nil, "revenue": float64(0), "final_revenue": float64(0), "discount": float64(0), "discount_type": "amount", "multiply_by_days": true, "prices_include_tax": false, "is_archived": false}
	}
	for key, value := range fields {
		draft[key] = value
	}
	if op == "archive" {
		draft["is_archived"] = true
		if id := uint(rentalJobNumber(current["status_id"])); id == jobstatus.PlanningID || id == jobstatus.ConfirmedID {
			draft["status_id"] = float64(jobstatus.CancelledID)
		}
	}
	if op == "restore" {
		draft["is_archived"] = false
	}
	previousStatus := jobstatus.PlanningID
	if op != "create" {
		previousStatus = uint(rentalJobNumber(current["status_id"]))
	}
	job, validationErr := rentalJobModel(draft)
	if validationErr == nil {
		validationErr = validateJobWrite(&job, previousStatus)
	}
	if validationErr != nil {
		required = append(required, "valid_job_fields")
		if job.CustomerID == 0 {
			required = append(required, "customer_id")
		}
		if job.Description == nil || strings.TrimSpace(*job.Description) == "" {
			required = append(required, "description")
		}
		if (job.StartDate == nil) != (job.EndDate == nil) || ((job.StatusID == jobstatus.ConfirmedID || job.StatusID == jobstatus.CompletedID) && job.StartDate == nil) {
			required = append(required, "start_date", "end_date")
		}
	} else {
		draft["description"] = *job.Description
		draft["discount_type"] = job.DiscountType
	}
	if op == "update" && len(fields) == 0 {
		required = append(required, "changed_fields")
	}
	references := map[string]any{}
	for _, ref := range []struct{ key, query string }{{"customer_id", `SELECT jsonb_build_object('id',customerid,'active',NOT COALESCE(is_archived,false),'updated_at',updated_at) FROM customers WHERE customerid=$1`}, {"venue_id", `SELECT jsonb_build_object('id',id,'active',NOT is_archived,'updated_at',updated_at) FROM venues WHERE id=$1`}, {"job_category_id", `SELECT jsonb_build_object('id',jobcategoryid,'name',name) FROM jobcategory WHERE jobcategoryid=$1`}, {"status_id", `SELECT jsonb_build_object('id',statusid,'name',status) FROM status WHERE statusid=$1`}} {
		if draft[ref.key] == nil {
			continue
		}
		record, err := rentalJobJSON(tx, ref.query, int64(rentalJobNumber(draft[ref.key])))
		if err == sql.ErrNoRows {
			required = append(required, ref.key)
			continue
		}
		if err != nil {
			masterError(c, err)
			return
		}
		references[ref.key] = record
		mustActive := op == "create" || op == "restore" || job.StatusID == jobstatus.PlanningID || job.StatusID == jobstatus.ConfirmedID || !reflect.DeepEqual(draft[ref.key], current[ref.key])
		if op != "archive" && mustActive && record["active"] == false {
			required = append(required, "active_"+ref.key)
		}
	}
	dependencies := map[string]any{}
	financialRequired := financialInput
	positions := []models.JobPosition{}
	if op != "create" {
		for _, item := range []struct{ key, query string }{{"devices", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.deviceid),'[]'::jsonb) FROM (SELECT jd.*,dv.status,dv.condition_status,dv.updated_at AS device_version FROM job_devices jd LEFT JOIN devices dv ON dv.deviceid=jd.deviceid WHERE jd.jobid=$1 LIMIT 1001) d`}, {"requirements", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.id),'[]'::jsonb) FROM (SELECT * FROM job_product_requirements WHERE job_id=$1 LIMIT 1001) d`}, {"positions", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.position_id),'[]'::jsonb) FROM (SELECT * FROM job_positions WHERE job_id=$1 LIMIT 1001) d`}, {"packages", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.job_package_id),'[]'::jsonb) FROM (SELECT * FROM job_packages WHERE job_id=$1 LIMIT 1001) d`}, {"staffing", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.employee_id),'[]'::jsonb) FROM (SELECT * FROM job_employees WHERE job_id=$1 LIMIT 1001) d`}, {"external_equipment", `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.equipment_id),'[]'::jsonb) FROM (SELECT * FROM job_rental_equipment WHERE job_id=$1 LIMIT 1001) d`}} {
			var raw json.RawMessage
			if err = tx.QueryRow(item.query, in.JobID).Scan(&raw); err != nil {
				masterError(c, err)
				return
			}
			var rows []any
			if err = json.Unmarshal(raw, &rows); err != nil {
				masterError(c, err)
				return
			}
			dependencies[item.key] = rows
			if len(rows) > 1000 {
				required = append(required, "bounded_job_context")
			}
			if item.key == "positions" {
				for _, value := range rows {
					row := value.(map[string]any)
					if row["deleted_at"] != nil {
						continue
					}
					positions = append(positions, models.JobPosition{Quantity: rentalJobNumber(row["quantity"]), UnitPrice: rentalJobNumber(row["unit_price"]), FollowDayFactor: rentalJobNumber(row["follow_day_factor"]), DiscountPercent: rentalJobNumber(row["discount_percent"]), DiscountAmount: rentalJobNumber(row["discount_amount"]), TaxRate: rentalJobNumber(row["tax_rate"])})
				}
			}
		}
	}
	if op != "create" {
		for _, item := range []struct{ table, key, query string }{
			{"warehouse_tasks", "warehouse_tasks", `SELECT task_id,status,is_archived,updated_at FROM warehouse_tasks WHERE job_id=$1`},
			{"cases", "cases", `SELECT caseid,to_jsonb(c)->>'lifecycle_status' AS lifecycle_status,updated_at FROM cases c WHERE (to_jsonb(c)->>'current_job_id')::bigint=$1`},
			{"job_edit_sessions", "active_editors", `SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes'`},
			{"job_package_reservations", "package_reservations", `SELECT r.* FROM job_package_reservations r JOIN job_packages p ON p.job_package_id=r.job_package_id WHERE p.job_id=$1`},
			{"job_position_devices", "position_devices", `SELECT d.* FROM job_position_devices d JOIN job_positions p ON p.position_id=d.position_id WHERE p.job_id=$1`},
			{"devices", "device_references", `SELECT deviceid AS id,COALESCE(to_jsonb(d)->>'lifecycle_status','active')='active' AS active,updated_at FROM devices d WHERE deviceid IN (SELECT deviceid FROM job_devices WHERE jobid=$1)`},
			{"products", "product_references", `SELECT productid AS id,lifecycle_status='active' AS active,updated_at FROM products WHERE productid IN (SELECT product_id FROM job_product_requirements r WHERE job_id=$1 AND COALESCE(to_jsonb(r)->>'deleted_at','')='' UNION SELECT product_id FROM job_positions WHERE job_id=$1 AND COALESCE(to_jsonb(job_positions)->>'deleted_at','')='' UNION SELECT d.productid FROM devices d JOIN job_devices jd ON jd.deviceid=d.deviceid WHERE jd.jobid=$1)`},
			{"product_packages", "package_references", `SELECT id,is_active AS active,updated_at FROM product_packages WHERE id IN (SELECT package_id FROM job_packages WHERE job_id=$1)`},
			{"employees", "employee_references", `SELECT id,is_active AS active,updated_at FROM employees WHERE id IN (SELECT employee_id FROM job_employees WHERE job_id=$1)`},
			{"rental_equipment", "external_equipment_references", `SELECT id,is_active AS active,updated_at FROM rental_equipment WHERE id IN (SELECT equipment_id FROM job_rental_equipment WHERE job_id=$1)`},
		} {
			var exists bool
			if err = tx.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, item.table).Scan(&exists); err != nil {
				masterError(c, err)
				return
			}
			if !exists {
				continue
			}
			args := []any{in.JobID}
			if item.key == "active_editors" {
				args = append(args, user.UserID)
			}
			var raw json.RawMessage
			query := `SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY to_jsonb(d)::text),'[]'::jsonb) FROM (` + item.query + ` LIMIT 1001) d`
			if err = tx.QueryRow(query, args...).Scan(&raw); err != nil {
				masterError(c, err)
				return
			}
			var rows []any
			if err = json.Unmarshal(raw, &rows); err != nil {
				masterError(c, err)
				return
			}
			dependencies[item.key] = rows
			if strings.HasSuffix(item.key, "_references") && (op == "restore" || op == "update" && (job.StatusID == jobstatus.PlanningID || job.StatusID == jobstatus.ConfirmedID)) {
				for _, value := range rows {
					if value.(map[string]any)["active"] == false {
						required = append(required, "active_"+item.key)
						break
					}
				}
			}
			if len(rows) > 1000 {
				required = append(required, "bounded_job_context")
			}
			if item.key == "active_editors" && len(rows) > 0 {
				required = append(required, "active_editing_session")
			}
		}
		if job.StatusID == jobstatus.PlanningID || job.StatusID == jobstatus.ConfirmedID {
			var raw json.RawMessage
			if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.job_id,d.device_id),'[]'::jsonb) FROM (SELECT DISTINCT j.jobid AS job_id,j.updated_at,j.startdate,j.enddate,jd.deviceid AS device_id FROM jobs j JOIN job_devices jd ON jd.jobid=j.jobid WHERE j.jobid<>$1 AND j.deleted_at IS NULL AND j.statusid IN (1,2) AND jd.deviceid IN (SELECT deviceid FROM job_devices WHERE jobid=$1) AND COALESCE(j.startdate,'-infinity'::date)<=COALESCE($3::date,'infinity'::date) AND COALESCE(j.enddate,'infinity'::date)>=COALESCE($2::date,'-infinity'::date) LIMIT 1001) d`, in.JobID, draft["start_date"], draft["end_date"]).Scan(&raw); err != nil {
				masterError(c, err)
				return
			}
			var conflicts []any
			if err = json.Unmarshal(raw, &conflicts); err != nil {
				masterError(c, err)
				return
			}
			dependencies["overlapping_device_jobs"] = conflicts
			if len(conflicts) > 0 && (op == "restore" || previousStatus != job.StatusID || !reflect.DeepEqual(current["start_date"], draft["start_date"]) || !reflect.DeepEqual(current["end_date"], draft["end_date"])) {
				required = append(required, "device_schedule_conflicts")
			}
		}
	}
	if op == "create" || op == "update" {
		revenue, final := job.Revenue, math.Max(0, job.Revenue-job.Discount)
		if job.DiscountType == "percent" {
			final = math.Max(0, job.Revenue*(1-job.Discount/100))
		}
		if len(positions) > 0 {
			if in.Revenue != nil {
				required = append(required, "revenue_owned_by_positions")
			}
			revenue, final = calculateJobPositionRevenue(job, positions)
		}
		revenue, final = roundAnalyticsMoney(revenue), roundAnalyticsMoney(final)
		draft["revenue"], draft["final_revenue"] = revenue, final
		if math.IsNaN(revenue) || math.IsInf(revenue, 0) || math.IsNaN(final) || math.IsInf(final, 0) || revenue > 9999999999.99 || final > 9999999999.99 {
			required = append(required, "commercial_totals_in_range")
		}
		if op == "update" && (!reflect.DeepEqual(current["revenue"], draft["revenue"]) || !reflect.DeepEqual(current["final_revenue"], draft["final_revenue"])) {
			financialRequired = true
			if !claims.Financial {
				required = append(required, "financial_effect_scope")
			}
		}
	}
	blockers, err := rentalJobJSON(tx, `SELECT rental_job_archive_blockers($1::int)`, in.JobID)
	if err != nil {
		masterError(c, err)
		return
	}
	if op == "archive" {
		for _, value := range blockers {
			if rentalJobNumber(value) > 0 {
				required = append(required, "active_warehouse_dependencies")
				break
			}
		}
	}
	candidates := []any{}
	if op == "create" || op == "update" || op == "restore" {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.job_id),'[]'::jsonb) FROM (SELECT jobid AS job_id,job_code,deleted_at IS NOT NULL AS is_archived,updated_at FROM jobs WHERE customerid=$1 AND lower(trim(description))=lower(trim($2)) AND startdate IS NOT DISTINCT FROM $3::date AND enddate IS NOT DISTINCT FROM $4::date AND jobid<>$5 ORDER BY jobid LIMIT 51) d`, job.CustomerID, draft["description"], draft["start_date"], draft["end_date"], in.JobID).Scan(&raw); err != nil {
			masterError(c, err)
			return
		}
		if err = json.Unmarshal(raw, &candidates); err != nil {
			masterError(c, err)
			return
		}
		for _, candidate := range candidates {
			row := candidate.(map[string]any)
			if row["is_archived"] == true && op == "create" {
				required = append(required, "restoration_required")
			} else if !in.AllowDuplicate {
				required = append(required, "review_duplicate_and_allow_explicitly")
			}
		}
		if len(candidates) > 50 {
			required = append(required, "bounded_duplicate_context")
		}
	}
	diff := map[string]any{}
	for key, value := range draft {
		if !reflect.DeepEqual(current[key], value) {
			diff[key] = map[string]any{"before": current[key], "after": value}
		}
	}
	encoded, _ := json.Marshal(map[string]any{"operation": op, "current": current, "draft": draft, "references": references, "dependencies": dependencies, "blockers": blockers, "candidates": candidates, "allow_duplicate": in.AllowDuplicate})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL JOB %d %s", strings.ToUpper(op), in.JobID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact job/draft-bound confirmation phrase required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "ready_to_create": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "references": references, "archive_blockers": blockers, "possible_duplicates": candidates, "expected_updated_at": current["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase, "effects": map[string]any{"history_retained": true, "external_messages_sent": false, "positions_own_revenue": len(positions) > 0, "issued_devices_enter_return_flow": current["status_id"] != draft["status_id"] && (job.StatusID == jobstatus.CompletedID || job.StatusID == jobstatus.CancelledID)}}
	result["effects"].(map[string]any)["requires_financial_scope"] = financialRequired
	summary := map[string]int{}
	for key, value := range dependencies {
		if rows, ok := value.([]any); ok {
			summary[key] = len(rows)
		}
	}
	result["dependency_counts"] = summary
	result["device_schedule_conflicts"] = dependencies["overlapping_device_jobs"]
	result["active_editors"] = dependencies["active_editors"]
	if validationErr != nil {
		result["validation_error"] = validationErr.Error()
	}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	id := in.JobID
	status := 200
	if op == "create" {
		err = tx.QueryRow(`INSERT INTO jobs(job_code,customerid,statusid,jobcategoryid,venue_id,description,startdate,enddate,revenue,final_revenue,discount,discount_type,multiply_by_days,prices_include_tax,created_by,updated_by) VALUES('PENDING',$1,$2,$3,$4,$5,$6::date,$7::date,$8,$9,$10,$11,$12,$13,$14,$14) RETURNING jobid`, draft["customer_id"], draft["status_id"], draft["job_category_id"], draft["venue_id"], draft["description"], draft["start_date"], draft["end_date"], draft["revenue"], draft["final_revenue"], draft["discount"], draft["discount_type"], draft["multiply_by_days"], draft["prices_include_tax"], user.UserID).Scan(&id)
		if err == nil {
			_, err = tx.Exec(`UPDATE jobs SET job_code=$2 WHERE jobid=$1`, id, fmt.Sprintf("JOB%06d", id))
		}
		status = 201
	} else if op == "update" {
		keys := []string{}
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		sets := []string{}
		args := []any{id}
		for _, key := range keys {
			if key == "revenue" {
				continue
			}
			args = append(args, draft[key])
			sets = append(sets, fmt.Sprintf("%s=$%d", rentalJobColumns[key], len(args)))
		}
		args = append(args, draft["revenue"], draft["final_revenue"], user.UserID)
		sets = append(sets, fmt.Sprintf("revenue=$%d,final_revenue=$%d,updated_by=$%d,revision=revision+1", len(args)-2, len(args)-1, len(args)))
		_, err = tx.Exec("UPDATE jobs SET "+strings.Join(sets, ",")+" WHERE jobid=$1", args...)
	} else {
		if op == "archive" {
			_, err = tx.Exec(`UPDATE jobs SET deleted_at=clock_timestamp() AT TIME ZONE 'UTC',statusid=$2,updated_by=$3,revision=revision+1 WHERE jobid=$1`, id, draft["status_id"], user.UserID)
		} else {
			_, err = tx.Exec(`UPDATE jobs SET deleted_at=NULL,updated_by=$2,revision=revision+1 WHERE jobid=$1`, id, user.UserID)
		}
	}
	if err != nil {
		masterError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, "SELECT "+rentalJobRecordSQL+" FROM jobs WHERE jobid=$1", id)
	if err != nil {
		masterError(c, err)
		return
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": record, "updated_at": record["updated_at"]})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_job',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.job."+op, fmt.Sprint(id), string(before), string(after)); err != nil {
		masterError(c, err)
		return
	}
	historyType := "updated"
	if op == "create" {
		historyType = "created"
	} else if op == "archive" {
		historyType = "deleted"
	} else if current["status_id"] != draft["status_id"] {
		historyType = "status_changed"
	}
	if _, err = tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, user.UserID, historyType, "mcp."+op, string(before), string(after), "MCP/AI "+op); err != nil {
		masterError(c, err)
		return
	}
	operationStatus := map[string]string{"create": "created", "update": "updated", "archive": "archived", "restore": "restored"}[op]
	result = map[string]any{"operation_status": operationStatus, "job": record, "diff": diff, "effects": result["effects"]}
	if op == "create" {
		result["creation_status"] = "created"
	}
	response, _ := json.Marshal(result)
	if _, err = tx.Exec(`UPDATE rental_mcp_mutation_receipts SET response=$2::jsonb,status_code=$3 WHERE id=$1`, receiptID, string(response), status); err != nil {
		masterError(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		masterError(c, err)
		return
	}
	c.Data(status, "application/json", response)
}
