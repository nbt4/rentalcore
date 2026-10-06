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
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rentalPositionRequest struct {
	PositionID             int64    `json:"position_id"`
	JobID                  int64    `json:"job_id"`
	ProductID              int64    `json:"product_id"`
	Description            *string  `json:"description"`
	Quantity               *float64 `json:"quantity"`
	Unit                   *string  `json:"unit"`
	UnitPrice              *float64 `json:"unit_price"`
	FollowDayFactor        *float64 `json:"follow_day_factor"`
	DiscountPercent        *float64 `json:"discount_percent"`
	DiscountAmount         *float64 `json:"discount_amount"`
	TaxRate                *float64 `json:"tax_rate"`
	ManualQuantity         *int64   `json:"manual_quantity"`
	PreserveManualQuantity bool     `json:"preserve_manual_quantity"`
	AllowDuplicate         bool     `json:"allow_duplicate"`
	ExpectedUpdatedAt      string   `json:"expected_updated_at"`
	ExpectedJobUpdatedAt   string   `json:"expected_job_updated_at"`
	ExpectedContext        string   `json:"expected_context"`
	ConfirmChange          bool     `json:"confirm_change"`
	ConfirmationText       string   `json:"confirmation_text"`
	Preview                bool     `json:"preview"`
}
type RentalPositionMCP struct {
	db           *sql.DB
	orm          *gorm.DB
	requirements *repository.RequirementRepository
}

func NewRentalPositionMCP(db *repository.Database) *RentalPositionMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalPositionMCP{db: sqlDB, orm: db.DB, requirements: repository.NewRequirementRepository(db)}
}

const rentalPositionRecordSQL = `jsonb_build_object('position_id',position_id,'job_id',job_id,'product_id',product_id,'position_type',position_type,'service_item_id',service_item_id,'rental_equipment_id',rental_equipment_id,'pdf_extraction_item_id',pdf_extraction_item_id,'description',description,'quantity',quantity,'unit',unit,'unit_price',unit_price,'follow_day_factor',follow_day_factor,'discount_percent',discount_percent,'discount_amount',discount_amount,'tax_rate',tax_rate,'sort_order',sort_order,'created_at',created_at,'is_archived',deleted_at IS NOT NULL,'archived_at',deleted_at,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func (h *RentalPositionMCP) Change(c *gin.Context) {
	op := c.Param("operation")
	if !map[string]bool{"create": true, "update": true, "archive": true}[op] {
		c.JSON(404, gin.H{"error": "Unknown named position operation"})
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
	if err != nil || !tok.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action || !claims.Financial || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user and selected rental action required"})
		return
	}
	var in rentalPositionRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 32769))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 32768 {
		c.JSON(400, gin.H{"error": "One bounded typed position object required"})
		return
	}
	for _, id := range []int64{in.PositionID, in.JobID, in.ProductID} {
		if id < 0 || id > math.MaxInt32 {
			c.JSON(400, gin.H{"error": "Canonical positive reference IDs required"})
			return
		}
	}
	if op == "create" && in.PositionID != 0 || op != "create" && in.PositionID == 0 {
		c.JSON(400, gin.H{"error": "Exact position ID required for existing records; omit for create"})
		return
	}
	if in.ManualQuantity != nil && (*in.ManualQuantity < 0 || *in.ManualQuantity > math.MaxInt32) {
		c.JSON(400, gin.H{"error": "Manual quantity must be a nonnegative bounded integer"})
		return
	}
	if op == "archive" && (in.Quantity != nil || in.UnitPrice != nil || in.Description != nil || in.Unit != nil || in.FollowDayFactor != nil || in.DiscountPercent != nil || in.DiscountAmount != nil || in.TaxRate != nil || in.ManualQuantity != nil || in.PreserveManualQuantity || in.AllowDuplicate) {
		c.JSON(400, gin.H{"error": "Archive preserves every position field and manual demand"})
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
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, "position."+op, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, "position."+op, keyHash).Scan(&previous, &response, &status); err != nil {
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
	if _, err = tx.Exec(`LOCK TABLE jobs IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_product_requirements,job_positions IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_position_devices,job_devices,devices,products,status IN SHARE MODE`); err != nil {
		masterError(c, err)
		return
	}
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
	}
	current := map[string]any{}
	required := []string{}
	jobID, productID := in.JobID, in.ProductID
	if op != "create" {
		current, err = rentalJobJSON(tx, "SELECT "+rentalPositionRecordSQL+" FROM job_positions WHERE position_id=$1", in.PositionID)
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Position not found"})
			return
		}
		if err != nil {
			masterError(c, err)
			return
		}
		jobID, productID = int64(rentalJobNumber(current["job_id"])), int64(rentalJobNumber(current["product_id"]))
		if in.JobID != 0 && in.JobID != jobID || in.ProductID != 0 && in.ProductID != productID {
			c.JSON(400, gin.H{"error": "Job/product identity is immutable"})
			return
		}
		if current["position_type"] != "product" {
			c.JSON(400, gin.H{"error": "These tools manage product positions; other position types retain their native workflows"})
			return
		}
		if current["is_archived"] == true {
			required = append(required, "active_position")
		}
		if !preview && in.ExpectedUpdatedAt == "" || in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != current["updated_at"] {
			required = append(required, "expected_updated_at")
		}
	}
	parent, err := rentalJobJSON(tx, `SELECT jsonb_build_object('job_id',jobid,'job_code',job_code,'description',description,'status_id',statusid,'start_date',startdate,'end_date',enddate,'is_archived',deleted_at IS NOT NULL,'multiply_by_days',multiply_by_days,'prices_include_tax',prices_include_tax,'discount',discount,'discount_type',discount_type,'revenue',revenue,'final_revenue',final_revenue,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) FROM jobs WHERE jobid=$1`, jobID)
	if err == sql.ErrNoRows {
		required = append(required, "job_id")
		parent = map[string]any{}
	} else if err != nil {
		masterError(c, err)
		return
	}
	if parent["is_archived"] == true {
		required = append(required, "restore_job_first")
	}
	if !preview && in.ExpectedJobUpdatedAt == "" || in.ExpectedJobUpdatedAt != "" && in.ExpectedJobUpdatedAt != parent["updated_at"] {
		required = append(required, "expected_job_updated_at")
	}
	product, err := rentalJobJSON(tx, `SELECT jsonb_build_object('product_id',productid,'name',name,'lifecycle_status',lifecycle_status,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) FROM products WHERE productid=$1`, productID)
	if err == sql.ErrNoRows {
		required = append(required, "product_id")
		product = map[string]any{}
	} else if err != nil {
		masterError(c, err)
		return
	}
	if op != "archive" && product["lifecycle_status"] != "active" {
		required = append(required, "active_product")
	}
	dependencies := map[string]any{}
	for _, item := range []struct{ key, query string }{
		{"positions", `SELECT ` + rentalPositionRecordSQL + ` AS record FROM job_positions WHERE job_id=$1`},
		{"requirements", `SELECT to_jsonb(r) AS record FROM job_product_requirements r WHERE job_id=$1`},
		{"position_devices", `SELECT to_jsonb(d) AS record FROM job_position_devices d JOIN job_positions p ON p.position_id=d.position_id WHERE p.job_id=$1`},
		{"assigned_devices", `SELECT jsonb_build_object('device_id',jd.deviceid,'product_id',d.productid,'pack_status',jd.pack_status,'updated_at',d.updated_at) AS record FROM job_devices jd JOIN devices d ON d.deviceid=jd.deviceid WHERE jd.jobid=$1`},
	} {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(record ORDER BY record::text),'[]'::jsonb) FROM (`+item.query+` LIMIT 1001) q`, jobID).Scan(&raw); err != nil {
			masterError(c, err)
			return
		}
		rows := []any{}
		if err = json.Unmarshal(raw, &rows); err != nil {
			masterError(c, err)
			return
		}
		dependencies[item.key] = rows
		if len(rows) > 1000 {
			required = append(required, "bounded_position_context")
		}
	}
	editors := []any{}
	if editorsExist {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.user_id),'[]'::jsonb) FROM (SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes' LIMIT 1001) d`, jobID, user.UserID).Scan(&raw); err != nil {
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
	dependencies["active_editors"] = editors
	draft := map[string]any{}
	for k, v := range current {
		draft[k] = v
	}
	if op == "create" {
		draft = map[string]any{"job_id": float64(jobID), "product_id": float64(productID), "position_type": "product", "quantity": float64(1), "unit": "Stück", "unit_price": float64(0), "description": product["name"], "follow_day_factor": float64(0.5), "discount_percent": float64(0), "discount_amount": float64(0), "tax_rate": float64(19), "sort_order": float64(0), "is_archived": false}
		if in.Quantity == nil {
			required = append(required, "quantity")
		}
		if in.UnitPrice == nil {
			required = append(required, "unit_price")
		}
		for _, value := range dependencies["positions"].([]any) {
			row := value.(map[string]any)
			order := rentalJobNumber(row["sort_order"]) + 1
			if order > rentalJobNumber(draft["sort_order"]) {
				draft["sort_order"] = order
			}
		}
	}
	changed := false
	if op != "archive" {
		for _, field := range []struct {
			key   string
			value *float64
		}{{"quantity", in.Quantity}, {"unit_price", in.UnitPrice}, {"follow_day_factor", in.FollowDayFactor}, {"discount_percent", in.DiscountPercent}, {"discount_amount", in.DiscountAmount}, {"tax_rate", in.TaxRate}} {
			if field.value != nil {
				draft[field.key] = *field.value
				changed = true
			}
		}
		for _, field := range []struct {
			key   string
			value *string
		}{{"description", in.Description}, {"unit", in.Unit}} {
			if field.value != nil {
				draft[field.key] = *field.value
				changed = true
			}
		}
		if op == "update" && !changed && in.ManualQuantity == nil {
			required = append(required, "changed_fields")
		}
	} else {
		draft["is_archived"] = true
	}
	pos := rentalPositionModel(draft)
	validationPos := pos
	if validationPos.ProductID == nil && op == "create" {
		placeholder := uint(1)
		validationPos.ProductID = &placeholder
	}
	if err = validatePosition(&validationPos); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if pos.Quantity > 99999999 || pos.UnitPrice > 9999999999.99 || pos.DiscountAmount > 9999999999.99 || pos.FollowDayFactor > 99.99 || pos.TaxRate > 999.99 || len(pos.Unit) > 50 || strings.TrimSpace(pos.Unit) == "" || len(pos.Description) > 4000 {
		c.JSON(400, gin.H{"error": "Position fields exceed supported limits"})
		return
	}
	for _, n := range []float64{pos.UnitPrice, pos.FollowDayFactor, pos.DiscountPercent, pos.DiscountAmount, pos.TaxRate} {
		if math.Abs(n*100-math.Round(n*100)) > 0.00001 {
			c.JSON(400, gin.H{"error": "Commercial amounts and factors support at most two decimal places"})
			return
		}
	}
	all := []models.JobPosition{}
	positionQuantity := int64(0)
	duplicates := []any{}
	for _, value := range dependencies["positions"].([]any) {
		row := value.(map[string]any)
		if row["is_archived"] == true {
			continue
		}
		if op != "create" && int64(rentalJobNumber(row["position_id"])) == in.PositionID {
			continue
		}
		all = append(all, rentalPositionModel(row))
		if row["position_type"] == "product" && int64(rentalJobNumber(row["product_id"])) == productID {
			positionQuantity += int64(math.Max(1, math.Round(rentalJobNumber(row["quantity"]))))
			duplicates = append(duplicates, row)
		}
	}
	if op != "archive" {
		all = append(all, pos)
		positionQuantity += int64(pos.Quantity)
	}
	if op == "create" && len(duplicates) > 0 && !in.AllowDuplicate {
		required = append(required, "review_duplicate_positions")
	}
	requirement := map[string]any{}
	manual := int64(0)
	for _, value := range dependencies["requirements"].([]any) {
		row := value.(map[string]any)
		if int64(rentalJobNumber(row["product_id"])) == productID {
			requirement = row
			if row["deleted_at"] == nil {
				manual = int64(rentalJobNumber(row["manual_quantity"]))
			}
			break
		}
	}
	originalManual := manual
	if op != "archive" {
		if in.ManualQuantity != nil && in.PreserveManualQuantity {
			c.JSON(400, gin.H{"error": "Choose replacement manual_quantity or preserve_manual_quantity, not both"})
			return
		}
		if manual > 0 && in.ManualQuantity == nil && !in.PreserveManualQuantity {
			required = append(required, "manual_quantity_or_preserve_manual_quantity")
		}
		if in.ManualQuantity != nil {
			manual = *in.ManualQuantity
		}
	}
	total := manual + positionQuantity
	if total > math.MaxInt32 || total < 0 {
		required = append(required, "bounded_material_quantity")
	}
	assigned := int64(0)
	positionAssigned := int64(0)
	for _, value := range dependencies["assigned_devices"].([]any) {
		row := value.(map[string]any)
		if int64(rentalJobNumber(row["product_id"])) == productID && row["pack_status"] != "returned" {
			assigned++
		}
	}
	for _, value := range dependencies["position_devices"].([]any) {
		row := value.(map[string]any)
		if int64(rentalJobNumber(row["position_id"])) == in.PositionID {
			positionAssigned++
		}
	}
	if total < assigned {
		required = append(required, "quantity_not_below_assigned_devices")
	}
	if op == "archive" && positionAssigned > 0 {
		required = append(required, "assigned_position_devices")
	}
	if op == "update" && pos.Quantity < float64(positionAssigned) {
		required = append(required, "quantity_not_below_position_devices")
	}
	job := models.Job{MultiplyByDays: parent["multiply_by_days"] == true, PricesIncludeTax: parent["prices_include_tax"] == true, Discount: rentalJobNumber(parent["discount"])}
	job.DiscountType, _ = parent["discount_type"].(string)
	// Native date parsing and calculation determine the same totals as the UI.
	if value, ok := parent["start_date"].(string); ok {
		date, e := time.Parse("2006-01-02", value)
		if e == nil {
			job.StartDate = &date
		}
	}
	if value, ok := parent["end_date"].(string); ok {
		date, e := time.Parse("2006-01-02", value)
		if e == nil {
			job.EndDate = &date
		}
	}
	revenue, final := calculateJobPositionRevenue(job, all)
	if math.IsNaN(revenue) || math.IsInf(revenue, 0) || revenue > 9999999999.99 || math.IsNaN(final) || math.IsInf(final, 0) {
		required = append(required, "bounded_job_revenue")
	}
	effects := map[string]any{"material_before": requirement, "material_after": map[string]any{"product_id": productID, "quantity": total, "position_quantity": positionQuantity, "manual_quantity": manual, "is_archived": total == 0}, "manual_quantity_before": originalManual, "revenue_before": parent["revenue"], "revenue_after": revenue, "final_revenue_before": parent["final_revenue"], "final_revenue_after": final, "requires_financial_scope": true, "stock_movements": false, "procurement_changes": false, "external_messages_sent": false}
	diff := map[string]any{}
	for k, v := range draft {
		if !reflect.DeepEqual(current[k], v) {
			diff[k] = map[string]any{"before": current[k], "after": v}
		}
	}
	encoded, _ := json.Marshal(map[string]any{"operation": op, "current": current, "draft": draft, "job": parent, "product": product, "dependencies": dependencies, "effects": effects, "allow_duplicate": in.AllowDuplicate, "preserve_manual_quantity": in.PreserveManualQuantity})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL JOB POSITION %d %s", strings.ToUpper(op), in.PositionID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact position/draft-bound confirmation phrase required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "job": parent, "product": product, "possible_duplicates": duplicates, "effects": effects, "active_editors": editors, "expected_updated_at": current["updated_at"], "expected_job_updated_at": parent["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	id := in.PositionID
	status := 200
	if op == "create" {
		err = tx.QueryRow(`INSERT INTO job_positions(job_id,product_id,position_type,description,quantity,unit,unit_price,follow_day_factor,discount_percent,discount_amount,tax_rate,sort_order) VALUES($1,$2,'product',$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING position_id`, jobID, productID, pos.Description, pos.Quantity, pos.Unit, pos.UnitPrice, pos.FollowDayFactor, pos.DiscountPercent, pos.DiscountAmount, pos.TaxRate, pos.SortOrder).Scan(&id)
		status = 201
	} else if op == "update" {
		_, err = tx.Exec(`UPDATE job_positions SET description=$2,quantity=$3,unit=$4,unit_price=$5,follow_day_factor=$6,discount_percent=$7,discount_amount=$8,tax_rate=$9 WHERE position_id=$1`, id, pos.Description, pos.Quantity, pos.Unit, pos.UnitPrice, pos.FollowDayFactor, pos.DiscountPercent, pos.DiscountAmount, pos.TaxRate)
	} else {
		_, err = tx.Exec(`UPDATE job_positions SET deleted_at=clock_timestamp() WHERE position_id=$1`, id)
	}
	if err != nil {
		masterError(c, err)
		return
	}
	// Use the owning native repositories on this exact SQL transaction.
	native := h.orm.Session(&gorm.Session{NewDB: true, SkipDefaultTransaction: true, Context: c.Request.Context()})
	native.Statement.ConnPool = tx
	if err = h.requirements.ReconcileProductPositionRequirement(native, uint(jobID), uint(productID), in.ManualQuantity); err != nil {
		masterError(c, err)
		return
	}
	if err = syncJobRevenue(native, uint(jobID)); err != nil {
		masterError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, "SELECT "+rentalPositionRecordSQL+" FROM job_positions WHERE position_id=$1", id)
	if err != nil {
		masterError(c, err)
		return
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": record, "updated_at": record["updated_at"], "effects": effects})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_job_position',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.job_position."+op, fmt.Sprint(id), string(before), string(after)); err != nil {
		masterError(c, err)
		return
	}
	if _, err = tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,'updated',$3,$4,$5,$6)`, jobID, user.UserID, "mcp.job_position."+op, string(before), string(after), "MCP/AI job position "+op); err != nil {
		masterError(c, err)
		return
	}
	var parentVersion string
	if err = tx.QueryRow(`SELECT to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM jobs WHERE jobid=$1`, jobID).Scan(&parentVersion); err != nil {
		masterError(c, err)
		return
	}
	result = map[string]any{"operation_status": map[string]string{"create": "created", "update": "updated", "archive": "archived"}[op], "position": record, "diff": diff, "job_updated_at": parentVersion, "effects": effects}
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

func rentalPositionModel(row map[string]any) models.JobPosition {
	pos := models.JobPosition{PositionID: uint(rentalJobNumber(row["position_id"])), JobID: uint(rentalJobNumber(row["job_id"])), Quantity: rentalJobNumber(row["quantity"]), UnitPrice: rentalJobNumber(row["unit_price"]), FollowDayFactor: rentalJobNumber(row["follow_day_factor"]), DiscountPercent: rentalJobNumber(row["discount_percent"]), DiscountAmount: rentalJobNumber(row["discount_amount"]), TaxRate: rentalJobNumber(row["tax_rate"]), SortOrder: int(rentalJobNumber(row["sort_order"]))}
	productID := uint(rentalJobNumber(row["product_id"]))
	if productID > 0 {
		pos.ProductID = &productID
	}
	pos.PositionType, _ = row["position_type"].(string)
	pos.Description, _ = row["description"].(string)
	pos.Unit, _ = row["unit"].(string)
	return pos
}
