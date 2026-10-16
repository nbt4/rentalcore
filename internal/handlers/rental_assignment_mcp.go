package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-barcode-webapp/internal/models"
	"io"
	"math"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type rentalAssignmentRequest struct {
	JobID                     int64    `json:"job_id"`
	EquipmentID               int64    `json:"equipment_id"`
	PositionID                int64    `json:"position_id"`
	Quantity                  *int64   `json:"quantity,omitempty"`
	DaysUsed                  *int64   `json:"days_used,omitempty"`
	RentalUnitPrice           *float64 `json:"rental_unit_price,omitempty"`
	CustomerPrice             *float64 `json:"customer_price,omitempty"`
	Description               *string  `json:"description,omitempty"`
	Notes                     *string  `json:"notes,omitempty"`
	Unit                      *string  `json:"unit,omitempty"`
	FollowDayFactor           *float64 `json:"follow_day_factor,omitempty"`
	DiscountPercent           *float64 `json:"discount_percent,omitempty"`
	DiscountAmount            *float64 `json:"discount_amount,omitempty"`
	TaxRate                   *float64 `json:"tax_rate,omitempty"`
	SortOrder                 *int64   `json:"sort_order,omitempty"`
	ExpectedUpdatedAt         string   `json:"expected_updated_at,omitempty"`
	ExpectedPositionUpdatedAt string   `json:"expected_position_updated_at,omitempty"`
	ExpectedJobUpdatedAt      string   `json:"expected_job_updated_at,omitempty"`
	ExpectedContext           string   `json:"expected_context,omitempty"`
	ConfirmationText          string   `json:"confirmation_text,omitempty"`
	ConfirmChange             bool     `json:"confirm_change,omitempty"`
	Preview                   bool     `json:"preview,omitempty"`
}

const rentalAssignmentRecordSQL = `jsonb_build_object('job_id',c.job_id,'equipment_id',c.equipment_id,'position_id',c.position_id,'quantity',c.quantity,'days_used',c.days_used,'rental_unit_price',c.rental_unit_price,'rental_unit_price_exact',c.rental_unit_price::text,'total_cost',c.total_cost,'notes',c.notes,'created_at',c.created_at,'updated_at',to_char(c.updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func validateRentalAssignment(in rentalAssignmentRequest, op string) error {
	for _, id := range []int64{in.JobID, in.EquipmentID, in.PositionID} {
		if id < 1 || id > math.MaxInt32 {
			return fmt.Errorf("Exact positive job, equipment and position IDs required")
		}
	}
	if in.Quantity != nil && (*in.Quantity < 1 || *in.Quantity > 1000) || in.DaysUsed != nil && (*in.DaysUsed < 1 || *in.DaysUsed > 365) {
		return fmt.Errorf("Quantity1–1000 and rental days1–365 required")
	}
	for _, price := range []*float64{in.RentalUnitPrice, in.CustomerPrice, in.DiscountAmount} {
		if price != nil && !validRentalPrice(price) {
			return fmt.Errorf("Nonnegative bounded prices with two decimals required")
		}
	}
	for _, v := range []*float64{in.DiscountPercent, in.TaxRate, in.FollowDayFactor} {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 100 || math.Abs(*v*100-math.Round(*v*100)) > 0.00001) {
			return fmt.Errorf("Percentages and day factors must be finite0–100")
		}
	}
	if in.FollowDayFactor != nil && *in.FollowDayFactor > 99.99 {
		return fmt.Errorf("Follow-day factor must fit NUMERIC(4,2)")
	}
	if in.SortOrder != nil && (*in.SortOrder < 0 || *in.SortOrder > math.MaxInt32) {
		return fmt.Errorf("Bounded nonnegative sort order required")
	}
	if in.Description != nil && utf8.RuneCountInString(*in.Description) > 10000 || in.Notes != nil && len(*in.Notes) > 500 || in.Unit != nil && (strings.TrimSpace(*in.Unit) == "" || utf8.RuneCountInString(*in.Unit) > 50) {
		return fmt.Errorf("Bounded description, notes and nonempty unit required")
	}
	if op != "update" && (in.Quantity != nil || in.DaysUsed != nil || in.RentalUnitPrice != nil || in.CustomerPrice != nil || in.Description != nil || in.Notes != nil || in.Unit != nil || in.FollowDayFactor != nil || in.DiscountPercent != nil || in.DiscountAmount != nil || in.TaxRate != nil || in.SortOrder != nil) {
		return fmt.Errorf("Lifecycle retains all business fields; update separately")
	}
	return nil
}

func (h *RentalJobMCP) changeExternalEquipment(c *gin.Context, op string) {
	if op != "update" && op != "archive" && op != "restore" {
		c.JSON(404, gin.H{"error": "Unknown named assignment operation"})
		return
	}
	action := op
	if op == "restore" {
		action = "archive"
	}
	user, ok := GetCurrentUser(c)
	if !ok || user == nil || user.UserID == 0 {
		c.JSON(403, gin.H{"error": "Interactive rental user required"})
		return
	}
	cookie, err := c.Cookie("cores_token")
	secret := os.Getenv("CORES_JWT_SECRET")
	if err != nil || secret == "" {
		c.JSON(403, gin.H{"error": "Signed MCP action delegation required"})
		return
	}
	claims := struct {
		UserID    uint   `json:"uid"`
		Scope     string `json:"mcp_scope"`
		Financial bool   `json:"mcp_financial"`
		jwt.RegisteredClaims
	}{}
	token, err := jwt.ParseWithClaims(cookie, &claims, func(*jwt.Token) (any, error) { return []byte(secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action || !claims.Financial || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user, selected rental action and rental financial access required"})
		return
	}
	var in rentalAssignmentRequest
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 256*1024+1))
	if err != nil || len(raw) > 256*1024 {
		c.JSON(400, gin.H{"error": "One bounded assignment object required"})
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF {
		c.JSON(400, gin.H{"error": "One closed typed assignment object required"})
		return
	}
	var supplied map[string]json.RawMessage
	if err = json.Unmarshal(raw, &supplied); err != nil {
		c.JSON(400, gin.H{"error": "Assignment object required"})
		return
	}
	for key, value := range supplied {
		if string(value) == "null" {
			c.JSON(400, gin.H{"error": "Omit fields to preserve; null is not an update value: " + key})
			return
		}
	}
	if err = validateRentalAssignment(in, op); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	preview := in.Preview || !in.ConfirmChange
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='20s';SET LOCAL TIME ZONE 'UTC'`); err != nil {
		rentalCatalogError(c, err)
		return
	}
	var admin bool
	if err = tx.QueryRow(`SELECT is_admin AND is_active FROM users WHERE userid=$1 FOR SHARE`, user.UserID).Scan(&admin); err != nil || !admin {
		c.JSON(403, gin.H{"error": "Current rental administrator rights required"})
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
		operation := "job.external_equipment." + op
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, operation, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, operation, keyHash).Scan(&previous, &response, &status); err != nil {
				rentalCatalogError(c, err)
				return
			}
			if previous != requestHash {
				c.JSON(409, gin.H{"error": "Idempotency key already binds a different request"})
				return
			}
			c.Data(status, "application/json", response)
			return
		}
		if err != nil {
			rentalCatalogError(c, err)
			return
		}
	}
	// Every later lock is NOWAIT: native child writers update parent job versions.
	// A collision must abort this short preview/mutation, never wait in reverse order.
	if _, err = tx.Exec(`LOCK TABLE rental_equipment,status IN SHARE MODE NOWAIT;LOCK TABLE jobs,job_positions,job_rental_equipment,job_position_devices IN SHARE ROW EXCLUSIVE MODE NOWAIT`); err != nil {
		rentalCatalogError(c, err)
		return
	}
	for _, lock := range []struct {
		sql  string
		args []any
	}{
		{`SELECT jobid FROM jobs WHERE jobid=$1 FOR UPDATE NOWAIT`, []any{in.JobID}},
		{`SELECT position_id FROM job_positions WHERE position_id=$1 FOR UPDATE NOWAIT`, []any{in.PositionID}},
		{`SELECT job_id FROM job_rental_equipment WHERE job_id=$1 AND equipment_id=$2 FOR UPDATE NOWAIT`, []any{in.JobID, in.EquipmentID}},
	} {
		if _, err = tx.Exec(lock.sql, lock.args...); err != nil {
			rentalCatalogError(c, err)
			return
		}
	}
	assignment, err := rentalJobJSON(tx, `SELECT `+rentalAssignmentRecordSQL+` FROM job_rental_equipment c WHERE job_id=$1 AND equipment_id=$2`, in.JobID, in.EquipmentID)
	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "Assignment not found"})
		return
	}
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	if assignment["position_id"] == nil {
		c.JSON(409, gin.H{"error": "Repair the unlinked legacy assignment with the existing explicit create/repair workflow first"})
		return
	}
	if int64(rentalJobNumber(assignment["position_id"])) != in.PositionID {
		c.JSON(400, gin.H{"error": "Exact retained assignment position required"})
		return
	}
	position, err := rentalJobJSON(tx, `SELECT `+rentalPositionRecordSQL+` FROM job_positions WHERE position_id=$1 AND job_id=$2 AND rental_equipment_id=$3 AND position_type='rental'`, in.PositionID, in.JobID, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	job, err := rentalJobJSON(tx, `SELECT `+rentalJobRecordSQL+` FROM jobs WHERE jobid=$1`, in.JobID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	equipment, err := rentalJobJSON(tx, `SELECT jsonb_build_object('equipment_id',id,'name',name,'supplier',supplier,'is_active',is_active,'updated_at',updated_at) FROM rental_equipment WHERE id=$1`, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	required := []string{}
	if job["is_archived"] == true {
		required = append(required, "restore_job_first")
	}
	archived := position["is_archived"] == true
	if op == "restore" && !archived {
		required = append(required, "already_active")
	}
	if op != "restore" && archived {
		required = append(required, "restore_assignment_first")
	}
	if op != "archive" && equipment["is_active"] != true {
		required = append(required, "restore_catalog_first")
	}
	for _, v := range []struct {
		provided string
		current  any
		key      string
	}{{in.ExpectedUpdatedAt, assignment["updated_at"], "expected_updated_at"}, {in.ExpectedPositionUpdatedAt, position["updated_at"], "expected_position_updated_at"}, {in.ExpectedJobUpdatedAt, job["updated_at"], "expected_job_updated_at"}} {
		if !preview && v.provided == "" || v.provided != "" && v.provided != v.current {
			required = append(required, v.key)
		}
	}
	draft := map[string]any{}
	for k, v := range assignment {
		draft[k] = v
	}
	posDraft := map[string]any{}
	for k, v := range position {
		posDraft[k] = v
	}
	for _, f := range []struct {
		key   string
		value *float64
	}{{"unit_price", in.CustomerPrice}, {"follow_day_factor", in.FollowDayFactor}, {"discount_percent", in.DiscountPercent}, {"discount_amount", in.DiscountAmount}, {"tax_rate", in.TaxRate}} {
		if f.value != nil {
			posDraft[f.key] = *f.value
		}
	}
	if in.Quantity != nil {
		draft["quantity"] = *in.Quantity
		posDraft["quantity"] = *in.Quantity
	}
	if in.DaysUsed != nil {
		draft["days_used"] = *in.DaysUsed
	}
	if in.RentalUnitPrice != nil {
		draft["rental_unit_price"] = *in.RentalUnitPrice
		draft["rental_unit_price_exact"] = fmt.Sprintf("%.2f", *in.RentalUnitPrice)
	}
	if in.Notes != nil {
		draft["notes"] = *in.Notes
	}
	if in.Description != nil {
		posDraft["description"] = *in.Description
	}
	if in.Unit != nil {
		posDraft["unit"] = *in.Unit
	}
	if in.SortOrder != nil {
		posDraft["sort_order"] = *in.SortOrder
	}
	posDraft["is_archived"] = op == "archive"
	var cost float64
	// Preserve NUMERIC snapshot precision when no explicit supplier price is supplied.
	if err = tx.QueryRow(`SELECT round(COALESCE($3::numeric,c.rental_unit_price)*$4::numeric*CASE WHEN j.multiply_by_days THEN $5::numeric ELSE 1 END,2) FROM job_rental_equipment c JOIN jobs j ON j.jobid=c.job_id WHERE c.job_id=$1 AND c.equipment_id=$2`, in.JobID, in.EquipmentID, in.RentalUnitPrice, draft["quantity"], draft["days_used"]).Scan(&cost); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if op == "archive" {
		draft["total_cost"] = assignment["total_cost"]
	} else {
		draft["total_cost"] = cost
	}
	if math.IsNaN(cost) || math.IsInf(cost, 0) || cost < 0 || cost > 9999999999.99 {
		required = append(required, "bounded_total_cost")
	}
	pos := rentalPositionModel(posDraft)
	equipID := uint(in.EquipmentID)
	pos.RentalEquipmentID = &equipID
	if err = validatePosition(&pos); err != nil {
		required = append(required, "valid_position_fields")
	}
	var rawContext json.RawMessage
	if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(`+rentalPositionRecordSQL+` ORDER BY position_id),'[]'::jsonb) FROM (SELECT * FROM job_positions WHERE job_id=$1 ORDER BY position_id LIMIT 1001) p`, in.JobID).Scan(&rawContext); err != nil {
		rentalCatalogError(c, err)
		return
	}
	positions := []map[string]any{}
	if err = json.Unmarshal(rawContext, &positions); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if len(positions) > 1000 {
		required = append(required, "bounded_position_context")
	}
	all := []models.JobPosition{}
	beforeAll := []models.JobPosition{}
	for _, p := range positions {
		id := int64(rentalJobNumber(p["position_id"]))
		if p["is_archived"] != true {
			beforeAll = append(beforeAll, rentalPositionModel(p))
		}
		if id == in.PositionID {
			continue
		}
		if p["is_archived"] != true {
			all = append(all, rentalPositionModel(p))
			if p["position_type"] == "rental" && int64(rentalJobNumber(p["rental_equipment_id"])) == in.EquipmentID {
				required = append(required, "duplicate_active_assignment")
			}
		}
	}
	if op != "archive" {
		all = append(all, pos)
	}
	var devices int
	if err = tx.QueryRow(`SELECT count(*) FROM job_position_devices WHERE position_id=$1`, in.PositionID).Scan(&devices); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if devices > 0 {
		required = append(required, "remove_position_devices_first")
	}
	editors := []any{}
	var hasEditors bool
	if err = tx.QueryRow(`SELECT to_regclass('job_edit_sessions') IS NOT NULL`).Scan(&hasEditors); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if hasEditors {
		if _, err = tx.Exec(`LOCK TABLE job_edit_sessions IN SHARE MODE NOWAIT`); err != nil {
			rentalCatalogError(c, err)
			return
		}
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY user_id),'[]'::jsonb) FROM (SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes' LIMIT 1001)e`, in.JobID, user.UserID).Scan(&rawContext); err != nil {
			rentalCatalogError(c, err)
			return
		}
		if err = json.Unmarshal(rawContext, &editors); err != nil {
			rentalCatalogError(c, err)
			return
		}
		if len(editors) > 0 {
			required = append(required, "active_editing_session")
		}
	}
	jobModel := rentalAssignmentJobModel(job)
	revenue, final := calculateJobPositionRevenue(jobModel, all)
	beforeRevenue, beforeFinal := calculateJobPositionRevenue(jobModel, beforeAll)
	if revenue > 9999999999.99 || final > 9999999999.99 || math.IsNaN(revenue) || math.IsInf(revenue, 0) || math.IsNaN(final) || math.IsInf(final, 0) {
		required = append(required, "bounded_job_revenue")
	}
	activeCostBefore := rentalJobNumber(assignment["total_cost"])
	if archived {
		activeCostBefore = 0
	}
	activeCostAfter := cost
	if op == "archive" {
		activeCostAfter = 0
	}
	effects := map[string]any{"supplier_cost_before": activeCostBefore, "supplier_cost_after": activeCostAfter, "supplier_cost_change": roundAnalyticsMoney(activeCostAfter - activeCostBefore), "captured_supplier_price_preserved": in.RentalUnitPrice == nil, "captured_customer_price_preserved": in.CustomerPrice == nil, "revenue_before": job["revenue"], "revenue_after": revenue, "final_revenue_before": job["final_revenue"], "final_revenue_after": final, "position_revenue_change": roundAnalyticsMoney(revenue - beforeRevenue), "position_final_revenue_change": roundAnalyticsMoney(final - beforeFinal), "stock_movements": false, "procurement_changes": false, "external_messages_sent": false, "restore_uses_current_job_day_setting": op == "restore"}
	beforeSales := rentalAssignmentSales(jobModel, rentalPositionModel(position), beforeAll, archived)
	afterSales := rentalAssignmentSales(jobModel, pos, all, op == "archive")
	effects["customer_sales_net_before"] = roundAnalyticsMoney(beforeSales.Net)
	effects["customer_sales_net_after"] = roundAnalyticsMoney(afterSales.Net)
	effects["customer_sales_gross_after"] = roundAnalyticsMoney(afterSales.Gross)
	effects["expected_margin_before"] = roundAnalyticsMoney(beforeSales.Net - activeCostBefore)
	effects["expected_margin_after"] = roundAnalyticsMoney(afterSales.Net - activeCostAfter)
	diff := map[string]any{}
	for k, v := range draft {
		if !rentalCatalogEqual(v, assignment[k]) {
			diff["assignment."+k] = map[string]any{"before": assignment[k], "after": v}
		}
	}
	for k, v := range posDraft {
		if !rentalCatalogEqual(v, position[k]) {
			diff["position."+k] = map[string]any{"before": position[k], "after": v}
		}
	}
	if op == "update" && len(diff) == 0 {
		required = append(required, "changed_fields")
	}
	encoded, _ := json.Marshal(map[string]any{"operation": op, "assignment": assignment, "position": position, "job": job, "equipment": equipment, "draft": draft, "position_draft": posDraft, "positions": positions, "devices": devices, "editors": editors, "effects": effects})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL JOB ASSIGNMENT %d/%d POSITION %d %s", strings.ToUpper(op), in.JobID, in.EquipmentID, in.PositionID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact record/draft-bound confirmation required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "current": assignment, "position": position, "draft": draft, "position_draft": posDraft, "diff": diff, "job": job, "equipment": equipment, "active_editors": editors, "effects": effects, "expected_updated_at": assignment["updated_at"], "expected_position_updated_at": position["updated_at"], "expected_job_updated_at": job["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	if op == "update" {
		if _, err = tx.Exec(`UPDATE job_positions SET description=$2,quantity=$3,unit=$4,unit_price=$5,follow_day_factor=$6,discount_percent=$7,discount_amount=$8,tax_rate=$9,sort_order=$10 WHERE position_id=$1`, in.PositionID, pos.Description, pos.Quantity, pos.Unit, pos.UnitPrice, pos.FollowDayFactor, pos.DiscountPercent, pos.DiscountAmount, pos.TaxRate, pos.SortOrder); err != nil {
			rentalCatalogError(c, err)
			return
		}
	} else {
		if _, err = tx.Exec(`UPDATE job_positions SET deleted_at=CASE WHEN $2 THEN clock_timestamp() ELSE NULL END WHERE position_id=$1`, in.PositionID, op == "archive"); err != nil {
			rentalCatalogError(c, err)
			return
		}
	}
	if op != "archive" {
		if _, err = tx.Exec(`UPDATE job_rental_equipment SET days_used=$3,rental_unit_price=COALESCE($4::numeric,rental_unit_price),notes=$5 WHERE job_id=$1 AND equipment_id=$2`, in.JobID, in.EquipmentID, draft["days_used"], in.RentalUnitPrice, draft["notes"]); err != nil {
			rentalCatalogError(c, err)
			return
		}
	}
	native := h.orm.Session(&gorm.Session{NewDB: true, SkipDefaultTransaction: true, Context: c.Request.Context()})
	native.Statement.ConnPool = tx
	if err = syncJobRevenue(native, uint(in.JobID)); err != nil {
		rentalCatalogError(c, err)
		return
	}
	afterAssignment, err := rentalJobJSON(tx, `SELECT `+rentalAssignmentRecordSQL+` FROM job_rental_equipment c WHERE job_id=$1 AND equipment_id=$2`, in.JobID, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	afterPosition, err := rentalJobJSON(tx, `SELECT `+rentalPositionRecordSQL+` FROM job_positions WHERE position_id=$1`, in.PositionID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	var jobVersion string
	if err = tx.QueryRow(`UPDATE jobs SET updated_by=$2,updated_at=clock_timestamp() WHERE jobid=$1 RETURNING to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, in.JobID, user.UserID).Scan(&jobVersion); err != nil {
		rentalCatalogError(c, err)
		return
	}
	before, _ := json.Marshal(map[string]any{"assignment": assignment, "position": position, "job": job})
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": afterAssignment, "position": afterPosition, "effects": effects, "job_updated_at": jobVersion})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_job_external_equipment',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.job_external_equipment."+op, fmt.Sprintf("%d/%d", in.JobID, in.EquipmentID), string(before), string(after)); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if _, err = tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,'updated',$3,$4,$5,$6)`, in.JobID, user.UserID, "mcp.job_external_equipment."+op, string(before), string(after), "MCP/AI retained external rental assignment "+op); err != nil {
		rentalCatalogError(c, err)
		return
	}
	statuses := map[string]string{"update": "updated", "archive": "archived", "restore": "restored"}
	response, _ := json.Marshal(map[string]any{"operation_status": statuses[op], "assignment": afterAssignment, "position": afterPosition, "job_updated_at": jobVersion, "effects": effects})
	if _, err = tx.Exec(`UPDATE rental_mcp_mutation_receipts SET response=$2::jsonb,status_code=200 WHERE id=$1`, receiptID, string(response)); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		rentalCatalogError(c, err)
		return
	}
	c.Data(200, "application/json", response)
}

func rentalAssignmentJobModel(job map[string]any) models.Job {
	out := models.Job{MultiplyByDays: job["multiply_by_days"] == true, PricesIncludeTax: job["prices_include_tax"] == true, Discount: rentalJobNumber(job["discount"])}
	out.DiscountType, _ = job["discount_type"].(string)
	for _, v := range []struct {
		key    string
		target **time.Time
	}{{"start_date", &out.StartDate}, {"end_date", &out.EndDate}} {
		if s, ok := job[v.key].(string); ok {
			if d, e := time.Parse("2006-01-02", s); e == nil {
				*v.target = &d
			}
		}
	}
	return out
}

func rentalAssignmentSales(job models.Job, pos models.JobPosition, all []models.JobPosition, inactive bool) revenueAmounts {
	if inactive {
		return revenueAmounts{}
	}
	subtotal := 0.0
	for _, p := range all {
		subtotal += positionInvoiceRevenue(p, job.MultiplyByDays, positionEventDays(job.StartDate, job.EndDate))
	}
	return splitInvoiceRevenue(positionInvoiceRevenue(pos, job.MultiplyByDays, positionEventDays(job.StartDate, job.EndDate)), pos.TaxRate, job.PricesIncludeTax).scale(jobDiscountFactor(subtotal, job.Discount, job.DiscountType))
}
