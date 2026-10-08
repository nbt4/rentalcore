package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rentalJobExternalEquipmentRequest struct {
	JobID                int64  `json:"job_id"`
	EquipmentID          int64  `json:"equipment_id"`
	Quantity             *int64 `json:"quantity"`
	DaysUsed             *int64 `json:"days_used"`
	RepairExisting       bool   `json:"repair_existing,omitempty"`
	Notes                string `json:"notes"`
	ExpectedJobUpdatedAt string `json:"expected_job_updated_at"`
	ExpectedContext      string `json:"expected_context"`
	ConfirmationText     string `json:"confirmation_text"`
	ConfirmChange        bool   `json:"confirm_change"`
	Preview              bool   `json:"preview"`
}

// Reuse the existing named job API route without adding startup or schema logic.
// Both the commercial position and supplier cost ledger use the owning transaction.
func (h *RentalJobMCP) createExternalEquipment(c *gin.Context) {
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
	if err != nil || !token.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:create" || !claims.Financial || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user, rental create action and rental financial access required"})
		return
	}
	var in rentalJobExternalEquipmentRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 8193))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 8192 {
		c.JSON(400, gin.H{"error": "One bounded typed rental assignment object required"})
		return
	}
	if in.JobID < 0 || in.JobID > math.MaxInt32 || in.EquipmentID < 0 || in.EquipmentID > math.MaxInt32 || len(in.Notes) > 500 || in.Quantity != nil && (*in.Quantity < 1 || *in.Quantity > 1000) || in.DaysUsed != nil && (*in.DaysUsed < 1 || *in.DaysUsed > 365) {
		c.JSON(400, gin.H{"error": "Canonical reference IDs, quantity 1–1000, days_used 1–365 and notes up to 500 bytes required"})
		return
	}
	preview := in.Preview || !in.ConfirmChange
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
		const operation = "job.external_equipment.create"
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, operation, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, operation, keyHash).Scan(&previous, &response, &status); err != nil {
				masterError(c, err)
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
			masterError(c, err)
			return
		}
	}
	// Freeze the job, assignments and catalog prices in the same lock order as
	// the existing job workflow; native UI/catalog edits invalidate the preview.
	if _, err = tx.Exec(`LOCK TABLE jobs IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_positions IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_rental_equipment IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE rental_equipment IN SHARE MODE`); err != nil {
		masterError(c, err)
		return
	}
	required := []string{}
	job, err := rentalJobJSON(tx, `SELECT `+rentalJobRecordSQL+` FROM jobs WHERE jobid=$1`, in.JobID)
	if err == sql.ErrNoRows {
		required = append(required, "job_id")
		job = map[string]any{}
	} else if err != nil {
		masterError(c, err)
		return
	}
	if job["is_archived"] == true {
		required = append(required, "restore_job_first")
	}
	if !preview && in.ExpectedJobUpdatedAt == "" || in.ExpectedJobUpdatedAt != "" && in.ExpectedJobUpdatedAt != job["updated_at"] {
		required = append(required, "expected_job_updated_at")
	}
	equipment, err := rentalJobJSON(tx, `SELECT jsonb_build_object('equipment_id',id,'name',name,'supplier',supplier,'category',category,'rental_price',rental_price,'customer_price',customer_price,'is_active',is_active,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) FROM rental_equipment WHERE id=$1`, in.EquipmentID)
	if err == sql.ErrNoRows {
		required = append(required, "equipment_id")
		equipment = map[string]any{}
	} else if err != nil {
		masterError(c, err)
		return
	}
	if equipment["is_active"] != true {
		required = append(required, "active_equipment")
	}
	price, priceOK := equipment["rental_price"].(float64)
	if !priceOK || math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price > 9999999999.99 || math.Abs(price*100-math.Round(price*100)) > 0.00001 {
		required = append(required, "catalog_rental_price")
	}
	customerPrice, customerOK := equipment["customer_price"].(float64)
	if !customerOK || !validRentalPrice(&customerPrice) {
		required = append(required, "catalog_customer_price")
	}
	if in.Quantity == nil {
		required = append(required, "quantity")
	}
	if in.DaysUsed == nil {
		required = append(required, "days_used")
	}
	var raw json.RawMessage
	if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.equipment_id),'[]'::jsonb) FROM (SELECT job_id,equipment_id,quantity,days_used,total_cost,notes,position_id,rental_unit_price,created_at,updated_at FROM job_rental_equipment WHERE job_id=$1 ORDER BY equipment_id LIMIT 1001) d`, in.JobID).Scan(&raw); err != nil {
		masterError(c, err)
		return
	}
	assignments := []any{}
	if err = json.Unmarshal(raw, &assignments); err != nil {
		masterError(c, err)
		return
	}
	if len(assignments) > 1000 {
		required = append(required, "bounded_assignment_context")
	}
	duplicates := []any{}
	for _, value := range assignments {
		row := value.(map[string]any)
		if int64(rentalJobNumber(row["equipment_id"])) == in.EquipmentID {
			duplicates = append(duplicates, row)
		}
	}
	if in.RepairExisting {
		if len(duplicates) != 1 {
			required = append(required, "existing_assignment_required")
		} else {
			old := duplicates[0].(map[string]any)
			if old["position_id"] != nil {
				required = append(required, "assignment_already_linked")
			}
			if in.Quantity != nil && rentalJobNumber(old["quantity"]) != float64(*in.Quantity) || in.DaysUsed != nil && rentalJobNumber(old["days_used"]) != float64(*in.DaysUsed) {
				required = append(required, "exact_existing_quantity_and_days_required")
			}
		}
	} else if len(duplicates) > 0 {
		required = append(required, "equipment_already_assigned")
	}
	if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(p) ORDER BY p.position_id),'[]'::jsonb) FROM (SELECT * FROM job_positions WHERE job_id=$1 ORDER BY position_id LIMIT 1001) p`, in.JobID).Scan(&raw); err != nil {
		masterError(c, err)
		return
	}
	positions := []map[string]any{}
	if err = json.Unmarshal(raw, &positions); err != nil {
		masterError(c, err)
		return
	}
	if len(positions) > 1000 {
		required = append(required, "bounded_position_context")
	}
	all := []models.JobPosition{}
	nextOrder := 0
	for _, p := range positions {
		if p["deleted_at"] != nil {
			continue
		}
		all = append(all, rentalPositionModel(p))
		if order := int(rentalJobNumber(p["sort_order"])) + 1; order > nextOrder {
			nextOrder = order
		}
		if p["position_type"] == "rental" && int64(rentalJobNumber(p["rental_equipment_id"])) == in.EquipmentID {
			required = append(required, "rental_position_already_exists")
		}
	}
	editors := []any{}
	var editorsExist bool
	if err = tx.QueryRow(`SELECT to_regclass('job_edit_sessions') IS NOT NULL`).Scan(&editorsExist); err != nil {
		masterError(c, err)
		return
	}
	if editorsExist {
		if _, err = tx.Exec(`LOCK TABLE job_edit_sessions IN SHARE MODE`); err != nil {
			masterError(c, err)
			return
		}
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.user_id),'[]'::jsonb) FROM (SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes' LIMIT 1001) d`, in.JobID, user.UserID).Scan(&raw); err != nil {
			masterError(c, err)
			return
		}
		if err = json.Unmarshal(raw, &editors); err != nil {
			masterError(c, err)
			return
		}
		if len(editors) > 0 {
			required = append(required, "active_editing_session")
		}
	}
	totalCost := any(nil)
	if priceOK && in.Quantity != nil && in.DaysUsed != nil && !in.RepairExisting {
		// PostgreSQL numeric arithmetic matches the native quantity/day settings
		// without losing cents through binary floating point multiplication.
		var cost float64
		if err = tx.QueryRow(`SELECT round(re.rental_price*$2::numeric*CASE WHEN COALESCE(j.multiply_by_days,true) THEN $3::numeric ELSE 1 END,2) FROM rental_equipment re JOIN jobs j ON j.jobid=$4 WHERE re.id=$1`, in.EquipmentID, *in.Quantity, *in.DaysUsed, in.JobID).Scan(&cost); err != nil && err != sql.ErrNoRows {
			masterError(c, err)
			return
		}
		if err == nil {
			totalCost = cost
			if cost < 0 || cost > 9999999999.99 || math.IsNaN(cost) || math.IsInf(cost, 0) {
				required = append(required, "bounded_total_cost")
			}
		}
	}

	effectiveNotes := in.Notes
	supplierUnitPrice := price
	if in.RepairExisting && len(duplicates) == 1 {
		old := duplicates[0].(map[string]any)
		effectiveNotes, _ = old["notes"].(string)
		if in.Notes != "" && in.Notes != effectiveNotes {
			required = append(required, "preserve_existing_assignment_notes")
		}
		totalCost = old["total_cost"]
		dayFactor := 1.0
		if job["multiply_by_days"] == true {
			dayFactor = math.Max(rentalJobNumber(old["days_used"]), 1)
		}
		if rentalJobNumber(old["quantity"]) <= 0 || rentalJobNumber(totalCost) < 0 || rentalJobNumber(totalCost) > 9999999999.99 {
			required = append(required, "valid_existing_supplier_cost")
		} else {
			supplierUnitPrice = rentalJobNumber(totalCost) / rentalJobNumber(old["quantity"]) / dayFactor
		}
		if value, ok := old["rental_unit_price"].(float64); ok {
			supplierUnitPrice = value
		}
	}
	jobModel := models.Job{MultiplyByDays: job["multiply_by_days"] == true, PricesIncludeTax: job["prices_include_tax"] == true, Discount: rentalJobNumber(job["discount"])}
	jobModel.DiscountType, _ = job["discount_type"].(string)
	for _, date := range []struct {
		key    string
		target **time.Time
	}{{"start_date", &jobModel.StartDate}, {"end_date", &jobModel.EndDate}} {
		if value, ok := job[date.key].(string); ok {
			if parsed, e := time.Parse("2006-01-02", value); e == nil {
				*date.target = &parsed
			}
		}
	}
	quantity := float64(0)
	if in.Quantity != nil {
		quantity = float64(*in.Quantity)
	}
	equipmentID := uint(in.EquipmentID)
	pos := models.JobPosition{JobID: uint(in.JobID), PositionType: "rental", RentalEquipmentID: &equipmentID, Description: fmt.Sprint(equipment["name"]), Quantity: quantity, Unit: "Stück", UnitPrice: customerPrice, FollowDayFactor: 0, TaxRate: 19, SortOrder: nextOrder}
	beforeRevenue, beforeFinal := calculateJobPositionRevenue(jobModel, all)
	revenue, final := calculateJobPositionRevenue(jobModel, append(all, pos))
	subtotal := 0.0
	for _, p := range append(all, pos) {
		subtotal += positionInvoiceRevenue(p, jobModel.MultiplyByDays, positionEventDays(jobModel.StartDate, jobModel.EndDate))
	}
	rawAmounts := splitInvoiceRevenue(positionInvoiceRevenue(pos, jobModel.MultiplyByDays, positionEventDays(jobModel.StartDate, jobModel.EndDate)), pos.TaxRate, jobModel.PricesIncludeTax)
	pos.LineNet = roundAnalyticsMoney(rawAmounts.Net)
	pos.LineGross = roundAnalyticsMoney(rawAmounts.Gross)
	amounts := rawAmounts.scale(jobDiscountFactor(subtotal, jobModel.Discount, jobModel.DiscountType))
	if revenue > 9999999999.99 || final > 9999999999.99 || math.IsNaN(revenue) || math.IsInf(revenue, 0) {
		required = append(required, "bounded_job_revenue")
	}
	draft := map[string]any{"job_id": in.JobID, "equipment_id": in.EquipmentID, "quantity": in.Quantity, "days_used": in.DaysUsed, "total_cost": totalCost, "notes": effectiveNotes, "repair_existing": in.RepairExisting, "position": pos}
	effects := map[string]any{"rental_cost_after": totalCost, "rental_price_source": "rental_equipment.rental_price", "customer_price_source": "rental_equipment.customer_price", "rental_price": equipment["rental_price"], "supplier_unit_price": supplierUnitPrice, "customer_price": equipment["customer_price"], "multiply_by_days": job["multiply_by_days"], "commercial_positions_created": true, "job_revenue_recalculated": true, "customer_sales_net": roundAnalyticsMoney(amounts.Net), "customer_sales_gross": roundAnalyticsMoney(amounts.Gross), "expected_margin": roundAnalyticsMoney(amounts.Net - rentalJobNumber(totalCost)), "revenue_before": job["revenue"], "revenue_after": revenue, "final_revenue_before": job["final_revenue"], "final_revenue_after": final, "job_revenue_change": roundAnalyticsMoney(revenue - rentalJobNumber(job["revenue"])), "job_final_revenue_change": roundAnalyticsMoney(final - rentalJobNumber(job["final_revenue"])), "position_revenue_change": roundAnalyticsMoney(revenue - beforeRevenue), "position_final_revenue_change": roundAnalyticsMoney(final - beforeFinal), "supplier_cost_preserved": in.RepairExisting, "stock_movements": false, "procurement_changes": false, "external_messages_sent": false}
	encoded, _ := json.Marshal(map[string]any{"operation": "job.external_equipment.create", "draft": draft, "job": job, "equipment": equipment, "assignments": assignments, "positions": positions, "active_editors": editors, "effects": effects})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	verb := "CREATE"
	if in.RepairExisting {
		verb = "REPAIR"
	}
	phrase := fmt.Sprintf("%s RENTAL JOB EXTERNAL EQUIPMENT %d/%d %s", verb, in.JobID, in.EquipmentID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact job/equipment/draft-bound confirmation phrase required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "draft": draft, "job": job, "equipment": equipment, "existing_assignments": assignments, "possible_duplicates": duplicates, "active_editors": editors, "effects": effects, "expected_job_updated_at": job["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	native := h.orm.Session(&gorm.Session{NewDB: true, SkipDefaultTransaction: true, Context: c.Request.Context()})
	native.Statement.ConnPool = tx
	if err = createRentalPosition(native, &pos, *in.DaysUsed, in.Notes, in.RepairExisting); err != nil {
		masterError(c, err)
		return
	}
	if err = syncJobRevenue(native, uint(in.JobID)); err != nil {
		masterError(c, err)
		return
	}
	position, err := rentalJobJSON(tx, "SELECT "+rentalPositionRecordSQL+" FROM job_positions WHERE position_id=$1", pos.PositionID)
	if err != nil {
		masterError(c, err)
		return
	}
	var jobVersion string
	if err = tx.QueryRow(`UPDATE jobs SET updated_at=clock_timestamp(),updated_by=$2 WHERE jobid=$1 RETURNING to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, in.JobID, user.UserID).Scan(&jobVersion); err != nil {
		masterError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, `SELECT to_jsonb(d) FROM job_rental_equipment d WHERE job_id=$1 AND equipment_id=$2`, in.JobID, in.EquipmentID)
	if err != nil {
		masterError(c, err)
		return
	}
	before := []byte(`{}`)
	if in.RepairExisting && len(duplicates) == 1 {
		before, _ = json.Marshal(duplicates[0])
	}
	historyDescription := "MCP/AI external rental equipment assignment"
	if in.RepairExisting {
		historyDescription = "MCP/AI rental position repair"
	}
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": record, "position": position, "effects": effects, "job_updated_at": jobVersion})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,'rental.job_external_equipment.create','rental_job_external_equipment',$2,$3::jsonb,$4::jsonb,CURRENT_TIMESTAMP)`, user.UserID, fmt.Sprintf("%d/%d", in.JobID, in.EquipmentID), string(before), string(after)); err != nil {
		masterError(c, err)
		return
	}
	if _, err = tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,'updated','mcp.job_external_equipment.create','',$3,$4)`, in.JobID, user.UserID, string(after), historyDescription); err != nil {
		masterError(c, err)
		return
	}
	result = map[string]any{"operation_status": "created", "assignment": record, "position": position, "job_updated_at": jobVersion, "effects": effects}
	response, _ := json.Marshal(result)
	if _, err = tx.Exec(`UPDATE rental_mcp_mutation_receipts SET response=$2::jsonb,status_code=201 WHERE id=$1`, receiptID, string(response)); err != nil {
		masterError(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		masterError(c, err)
		return
	}
	c.Data(201, "application/json", response)
}
