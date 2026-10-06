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

	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rentalRequirementRequest struct {
	RequirementID        int64  `json:"requirement_id"`
	JobID                int64  `json:"job_id"`
	ProductID            int64  `json:"product_id"`
	Quantity             *int64 `json:"quantity"`
	ManualQuantity       *int64 `json:"manual_quantity"`
	ExpectedUpdatedAt    string `json:"expected_updated_at"`
	ExpectedJobUpdatedAt string `json:"expected_job_updated_at"`
	ExpectedContext      string `json:"expected_context"`
	ConfirmChange        bool   `json:"confirm_change"`
	ConfirmationText     string `json:"confirmation_text"`
	Preview              bool   `json:"preview"`
}
type RentalRequirementMCP struct{ db *sql.DB }

func NewRentalRequirementMCP(db *repository.Database) *RentalRequirementMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalRequirementMCP{db: sqlDB}
}

const rentalRequirementRecordSQL = `jsonb_build_object('requirement_id',id,'id',id,'job_id',job_id,'product_id',product_id,'quantity',quantity,'manual_quantity',manual_quantity,'position_quantity',position_quantity,'created_at',created_at,'is_archived',deleted_at IS NOT NULL,'archived_at',deleted_at,'updated_at',to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func (h *RentalRequirementMCP) Change(c *gin.Context) {
	op := c.Param("operation")
	if !map[string]bool{"create": true, "update": true, "archive": true, "restore": true}[op] {
		c.JSON(404, gin.H{"error": "Unknown named requirement operation"})
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
		UserID uint   `json:"uid"`
		Scope  string `json:"mcp_scope"`
		jwt.RegisteredClaims
	}{}
	tok, err := jwt.ParseWithClaims(cookie, &claims, func(*jwt.Token) (any, error) { return []byte(os.Getenv("CORES_JWT_SECRET")), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	action := op
	if op == "restore" {
		action = "archive"
	}
	if err != nil || !tok.Valid || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user and selected rental action required"})
		return
	}
	var in rentalRequirementRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 32769))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 32768 {
		c.JSON(400, gin.H{"error": "One bounded typed requirement object required"})
		return
	}
	for _, id := range []int64{in.RequirementID, in.JobID, in.ProductID} {
		if id < 0 || id > math.MaxInt32 {
			c.JSON(400, gin.H{"error": "Canonical positive reference IDs required"})
			return
		}
	}
	if op == "create" && in.RequirementID != 0 || op != "create" && in.RequirementID == 0 {
		c.JSON(400, gin.H{"error": "Exact requirement ID required for existing records; omit for create"})
		return
	}
	for _, q := range []*int64{in.Quantity, in.ManualQuantity} {
		if q != nil && (*q < 0 || *q > math.MaxInt32) {
			c.JSON(400, gin.H{"error": "Requirement quantities must be nonnegative bounded integers"})
			return
		}
	}
	if (op == "archive" || op == "restore") && (in.Quantity != nil || in.ManualQuantity != nil) {
		c.JSON(400, gin.H{"error": "Requirement lifecycle preserves every business field"})
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
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, "requirement."+op, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, "requirement."+op, keyHash).Scan(&previous, &response, &status); err != nil {
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
	if _, err = tx.Exec(`LOCK TABLE jobs IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_product_requirements IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE job_positions,job_devices,devices,products,status IN SHARE MODE`); err != nil {
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
		current, err = rentalJobJSON(tx, "SELECT "+rentalRequirementRecordSQL+" FROM job_product_requirements WHERE id=$1", in.RequirementID)
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Requirement not found"})
			return
		}
		if err != nil {
			masterError(c, err)
			return
		}
		jobID, productID = int64(rentalJobNumber(current["job_id"])), int64(rentalJobNumber(current["product_id"]))
		if in.JobID != 0 && in.JobID != jobID || in.ProductID != 0 && in.ProductID != productID {
			c.JSON(400, gin.H{"error": "Job/product identity is immutable; archive the original and create the reviewed new line"})
			return
		}
		archived := current["is_archived"] == true
		if op == "restore" && !archived || op != "restore" && archived {
			required = append(required, "lifecycle_state")
		}
		if !preview && in.ExpectedUpdatedAt == "" || in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != current["updated_at"] {
			required = append(required, "expected_updated_at")
		}
	}
	parent, err := rentalJobJSON(tx, `SELECT jsonb_build_object('job_id',jobid,'job_code',job_code,'description',description,'status_id',statusid,'start_date',startdate,'end_date',enddate,'is_archived',deleted_at IS NOT NULL,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) FROM jobs WHERE jobid=$1`, jobID)
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
	positionQuantity := int64(0)
	assigned := 0
	for _, item := range []struct{ key, query string }{
		{"positions", `SELECT position_id,product_id,position_type,quantity,updated_at FROM job_positions WHERE job_id=$1 AND product_id=$2 AND position_type='product' AND COALESCE(to_jsonb(job_positions)->>'deleted_at','')=''`},
		{"assigned_devices", `SELECT jd.deviceid,jd.pack_status,d.lifecycle_status,d.status,d.updated_at FROM job_devices jd JOIN devices d ON d.deviceid=jd.deviceid WHERE jd.jobid=$1 AND d.productid=$2`},
	} {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY to_jsonb(d)::text),'[]'::jsonb) FROM (`+item.query+` LIMIT 1001) d`, jobID, productID).Scan(&raw); err != nil {
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
			required = append(required, "bounded_requirement_context")
		}
		for _, value := range rows {
			row := value.(map[string]any)
			if item.key == "positions" {
				q := rentalJobNumber(row["quantity"])
				if math.IsNaN(q) || math.IsInf(q, 0) || q > math.MaxInt32 {
					required = append(required, "valid_position_quantities")
					continue
				}
				positionQuantity += int64(math.Max(1, math.Round(q)))
			} else if row["pack_status"] != "returned" {
				assigned++
			}
		}
	}
	editors := []any{}
	if editorsExist {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.user_id),'[]'::jsonb) FROM (SELECT user_id,last_seen FROM job_edit_sessions WHERE job_id=$1 AND user_id<>$2 AND last_seen>CURRENT_TIMESTAMP-INTERVAL '2 minutes' ORDER BY user_id LIMIT 1001) d`, jobID, user.UserID).Scan(&raw); err != nil {
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
	for key, value := range current {
		draft[key] = value
	}
	if op == "create" {
		draft = map[string]any{"job_id": float64(jobID), "product_id": float64(productID), "quantity": float64(0), "manual_quantity": float64(0), "position_quantity": float64(positionQuantity), "is_archived": false}
	}
	manual := int64(rentalJobNumber(draft["manual_quantity"]))
	total := int64(rentalJobNumber(draft["quantity"]))
	if op == "create" || op == "update" {
		if op == "create" && in.Quantity == nil && in.ManualQuantity == nil {
			required = append(required, "quantity_or_manual_quantity")
		}
		if op == "update" && in.Quantity == nil && in.ManualQuantity == nil {
			required = append(required, "changed_fields")
		}
		if in.ManualQuantity != nil {
			manual = *in.ManualQuantity
		}
		total = manual + positionQuantity
		if in.Quantity != nil {
			total = *in.Quantity
			if total < positionQuantity {
				required = append(required, "quantity_not_below_positions")
			} else {
				if in.ManualQuantity != nil && manual != total-positionQuantity {
					required = append(required, "consistent_quantity_sources")
				}
				manual = total - positionQuantity
			}
		}
		draft["quantity"], draft["manual_quantity"], draft["position_quantity"] = float64(total), float64(manual), float64(positionQuantity)
	}
	if total < 1 || total > math.MaxInt32 || manual < 0 || positionQuantity > math.MaxInt32 {
		required = append(required, "positive_bounded_total_quantity")
	}
	if op == "archive" {
		draft["is_archived"] = true
		if positionQuantity > 0 {
			required = append(required, "commercial_position_contributions")
		}
		if assigned > 0 {
			required = append(required, "assigned_equipment")
		}
	}
	if op == "restore" {
		draft["is_archived"] = false
		if int64(rentalJobNumber(current["position_quantity"])) != positionQuantity {
			required = append(required, "restore_position_sources_first")
		}
	}
	if op != "archive" && total < int64(assigned) {
		required = append(required, "quantity_not_below_assigned_devices")
	}
	candidates := []any{}
	if op == "create" {
		var raw json.RawMessage
		if err = tx.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(d) ORDER BY d.requirement_id),'[]'::jsonb) FROM (SELECT id AS requirement_id,quantity,manual_quantity,position_quantity,deleted_at IS NOT NULL AS is_archived,updated_at FROM job_product_requirements WHERE job_id=$1 AND product_id=$2 LIMIT 2) d`, jobID, productID).Scan(&raw); err != nil {
			masterError(c, err)
			return
		}
		if err = json.Unmarshal(raw, &candidates); err != nil {
			masterError(c, err)
			return
		}
		for _, value := range candidates {
			if value.(map[string]any)["is_archived"] == true {
				required = append(required, "restoration_required")
			} else {
				required = append(required, "requirement_already_exists")
			}
		}
	}
	diff := map[string]any{}
	for key, value := range draft {
		if !reflect.DeepEqual(current[key], value) {
			diff[key] = map[string]any{"before": current[key], "after": value}
		}
	}
	encoded, _ := json.Marshal(map[string]any{"operation": op, "current": current, "draft": draft, "job": parent, "product": product, "dependencies": dependencies, "duplicates": candidates})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL REQUIREMENT %d %s", strings.ToUpper(op), in.RequirementID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact requirement/draft-bound confirmation phrase required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "ready_to_create": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "job": parent, "product": product, "possible_duplicates": candidates, "position_quantity": positionQuantity, "assigned_device_count": assigned, "active_editors": editors, "expected_updated_at": current["updated_at"], "expected_job_updated_at": parent["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase, "effects": map[string]any{"material_demand_active": op != "archive", "identity_and_original_fields_retained": op == "archive" || op == "restore", "stock_movements": false, "financial_changes": false, "external_messages_sent": false}}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	id := in.RequirementID
	status := 200
	if op == "create" {
		err = tx.QueryRow(`INSERT INTO job_product_requirements(job_id,product_id,quantity,manual_quantity,position_quantity) VALUES($1,$2,$3,$4,$5) RETURNING id`, jobID, productID, total, manual, positionQuantity).Scan(&id)
		status = 201
	} else if op == "update" {
		_, err = tx.Exec(`UPDATE job_product_requirements SET quantity=$2,manual_quantity=$3,position_quantity=$4 WHERE id=$1`, id, total, manual, positionQuantity)
	} else if op == "archive" {
		_, err = tx.Exec(`UPDATE job_product_requirements SET deleted_at=clock_timestamp() WHERE id=$1`, id)
	} else {
		_, err = tx.Exec(`UPDATE job_product_requirements SET deleted_at=NULL WHERE id=$1`, id)
	}
	if err != nil {
		masterError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, "SELECT "+rentalRequirementRecordSQL+" FROM job_product_requirements WHERE id=$1", id)
	if err != nil {
		masterError(c, err)
		return
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": record, "updated_at": record["updated_at"]})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_requirement',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.requirement."+op, fmt.Sprint(id), string(before), string(after)); err != nil {
		masterError(c, err)
		return
	}
	if _, err = tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES($1,$2,'updated',$3,$4,$5,$6)`, jobID, user.UserID, "mcp.requirement."+op, string(before), string(after), "MCP/AI requirement "+op); err != nil {
		masterError(c, err)
		return
	}
	var parentVersion string
	if err = tx.QueryRow(`SELECT to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM jobs WHERE jobid=$1`, jobID).Scan(&parentVersion); err != nil {
		masterError(c, err)
		return
	}
	result = map[string]any{"operation_status": map[string]string{"create": "created", "update": "updated", "archive": "archived", "restore": "restored"}[op], "requirement": record, "diff": diff, "job_updated_at": parentVersion, "effects": result["effects"]}
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
