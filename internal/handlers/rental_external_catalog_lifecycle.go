package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rentalExternalCatalogChange struct {
	EquipmentID       int64    `json:"equipment_id"`
	Name              *string  `json:"name"`
	Category          *string  `json:"category"`
	Supplier          *string  `json:"supplier"`
	SupplierID        *int64   `json:"supplier_id"`
	RentalPrice       *float64 `json:"rental_price"`
	CustomerPrice     *float64 `json:"customer_price"`
	Description       *string  `json:"description"`
	Notes             *string  `json:"notes"`
	AllowSimilar      bool     `json:"allow_similar"`
	ExpectedUpdatedAt string   `json:"expected_updated_at"`
	ExpectedContext   string   `json:"expected_context"`
	ConfirmationText  string   `json:"confirmation_text"`
	ConfirmChange     bool     `json:"confirm_change"`
	Preview           bool     `json:"preview"`
}

var rentalExternalCatalogColumns = map[string]string{"name": "name", "category": "category", "supplier": "supplier", "supplier_id": "supplier_id", "rental_price": "rental_price", "customer_price": "customer_price", "description": "description", "notes": "notes"}

func rentalCatalogError(c *gin.Context, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "23505" || pg.Code == "55P03" || pg.Code == "40P01" || pg.Code == "40001") {
		c.JSON(409, gin.H{"error": "Catalog state, dependencies or identity changed; prepare the operation again", "code": "external_catalog_conflict"})
		return
	}
	masterError(c, err)
}

func (h *RentalExternalEquipmentMCP) Change(c *gin.Context) {
	op := c.Param("operation")
	if op != "update" && op != "archive" && op != "restore" {
		c.JSON(404, gin.H{"error": "Unknown named external rental catalog operation"})
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
	cookie, _ := c.Cookie("cores_token")
	claims := struct {
		UID       uint   `json:"uid"`
		Scope     string `json:"mcp_scope"`
		Financial bool   `json:"mcp_financial"`
		jwt.RegisteredClaims
	}{}
	token, err := jwt.ParseWithClaims(cookie, &claims, func(*jwt.Token) (any, error) { return []byte(os.Getenv("CORES_JWT_SECRET")), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if os.Getenv("CORES_JWT_SECRET") == "" || err != nil || !token.Valid || claims.UID != user.UserID || claims.Scope != "cores:rental:"+action || !claims.Financial || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.JSON(403, gin.H{"error": "Signed real user, matching rental action and separate rental financial access required"})
		return
	}
	var raw json.RawMessage
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 256*1024+1))
	if err = decoder.Decode(&raw); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 256*1024 {
		c.JSON(400, gin.H{"error": "One bounded catalog object required"})
		return
	}
	var in rentalExternalCatalogChange
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || in.EquipmentID < 1 || in.EquipmentID > math.MaxInt32 {
		c.JSON(400, gin.H{"error": "Closed catalog fields and exact positive equipment_id required"})
		return
	}
	supplied := map[string]json.RawMessage{}
	_ = json.Unmarshal(raw, &supplied)
	fields := map[string]any{}
	for key, value := range supplied {
		if _, exists := rentalExternalCatalogColumns[key]; !exists {
			continue
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || op != "update" {
			c.JSON(400, gin.H{"error": "Lifecycle preserves fields; update uses omitted fields to preserve and empty optional strings or supplier_id=0 to clear"})
			return
		}
	}
	for _, field := range []struct {
		key   string
		value *string
		limit int
	}{{"name", in.Name, 255}, {"supplier", in.Supplier, 255}, {"category", in.Category, 100}, {"description", in.Description, 10000}, {"notes", in.Notes, 10000}} {
		if field.value != nil {
			v := strings.TrimSpace(*field.value)
			if len([]rune(v)) > field.limit {
				c.JSON(400, gin.H{"error": "Catalog field exceeds documented length"})
				return
			}
			fields[field.key] = v
		}
	}
	if in.SupplierID != nil {
		if *in.SupplierID < 0 || *in.SupplierID > math.MaxInt32 {
			c.JSON(400, gin.H{"error": "Canonical supplier_id required"})
			return
		}
		fields["supplier_id"] = nil
		if *in.SupplierID != 0 {
			fields["supplier_id"] = *in.SupplierID
		}
	}
	for _, field := range []struct {
		key   string
		value *float64
	}{{"rental_price", in.RentalPrice}, {"customer_price", in.CustomerPrice}} {
		if field.value != nil {
			if !validRentalPrice(field.value) || *field.value > 99999999.99 {
				c.JSON(400, gin.H{"error": "Prices require nonnegative values with two decimals up to 99999999.99"})
				return
			}
			fields[field.key] = *field.value
		}
	}
	if op != "update" && in.AllowSimilar {
		c.JSON(400, gin.H{"error": "allow_similar is only an identity-update acknowledgement"})
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
		c.JSON(403, gin.H{"error": "Current active Rental administrator required"})
		return
	}
	receiptID := int64(0)
	if !preview {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if !rentalMasterKeyPattern.MatchString(key) || !rentalMasterContextPattern.MatchString(in.ExpectedContext) || in.ExpectedUpdatedAt == "" {
			c.JSON(428, gin.H{"error": "Durable key and exact final record/context versions required"})
			return
		}
		kd := sha256.Sum256([]byte(key))
		requestBytes, _ := json.Marshal(in)
		rd := sha256.Sum256(requestBytes)
		keyHash, requestHash := hex.EncodeToString(kd[:]), hex.EncodeToString(rd[:])
		err = tx.QueryRow(`INSERT INTO rental_mcp_mutation_receipts(user_id,operation,key_hash,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`, user.UserID, "external_equipment."+op, keyHash, requestHash).Scan(&receiptID)
		if err == sql.ErrNoRows {
			var previous string
			var response json.RawMessage
			var status int
			if err = tx.QueryRow(`SELECT request_hash,response,status_code FROM rental_mcp_mutation_receipts WHERE user_id=$1 AND operation=$2 AND key_hash=$3`, user.UserID, "external_equipment."+op, keyHash).Scan(&previous, &response, &status); err != nil {
				rentalCatalogError(c, err)
				return
			}
			if previous != requestHash {
				c.JSON(409, gin.H{"error": "Idempotency key binds a different request"})
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
	// Freeze the catalog before resolving references. Never wait for subsequent
	// jobs/custom rows: native child writers touch parent versions after taking
	// child locks, so NOWAIT avoids introducing a reverse lock dependency.
	if _, err = tx.Exec(`LOCK TABLE customers,status IN SHARE MODE NOWAIT;LOCK TABLE rental_equipment IN SHARE ROW EXCLUSIVE MODE NOWAIT`); err != nil {
		rentalCatalogError(c, err)
		return
	}
	var lockedID int64
	if err = tx.QueryRow(`SELECT id FROM rental_equipment WHERE id=$1 FOR UPDATE NOWAIT`, in.EquipmentID).Scan(&lockedID); err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "External rental catalog item not found"})
		return
	} else if err != nil {
		rentalCatalogError(c, err)
		return
	}
	referenceRows, err := tx.Query(`SELECT j.jobid FROM jobs j WHERE j.jobid IN (SELECT job_id FROM job_positions WHERE rental_equipment_id=$1 UNION SELECT job_id FROM job_rental_equipment WHERE equipment_id=$1) ORDER BY j.jobid LIMIT 1001 FOR SHARE OF j NOWAIT`, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	referenceJobs := []int64{}
	for referenceRows.Next() {
		var id int64
		if err = referenceRows.Scan(&id); err != nil {
			break
		}
		referenceJobs = append(referenceJobs, id)
	}
	if err == nil {
		err = referenceRows.Err()
	}
	referenceRows.Close()
	if err != nil {
		rentalCatalogError(c, err)
		return
	}

	current, err := rentalJobJSON(tx, `SELECT `+rentalExternalEquipmentRecordSQL+` FROM rental_equipment WHERE id=$1`, in.EquipmentID)
	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "External rental catalog item not found"})
		return
	}
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	required := []string{}
	if len(referenceJobs) > 1000 {
		required = append(required, "bounded_dependencies")
	}
	if in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != current["updated_at"] {
		required = append(required, "expected_updated_at")
	}
	draft := map[string]any{}
	for k, v := range current {
		draft[k] = v
	}
	for k, v := range fields {
		draft[k] = v
	}
	if op == "update" && current["is_active"] != true {
		required = append(required, "restore_before_editing")
	}
	if op == "archive" {
		if current["is_active"] != true {
			required = append(required, "already_archived")
		}
		draft["is_active"] = false
	}
	if op == "restore" {
		if current["is_active"] != false {
			required = append(required, "already_active")
		}
		draft["is_active"] = true
	}
	supplier := map[string]any{}
	if draft["supplier_id"] != nil {
		supplier, err = rentalJobJSON(tx, `SELECT jsonb_build_object('supplier_id',customerid,'name',COALESCE(NULLIF(companyname,''),NULLIF(name,''),trim(concat_ws(' ',firstname,lastname))),'is_supplier',is_supplier,'is_archived',COALESCE(is_archived,false),'updated_at',updated_at) FROM customers WHERE customerid=$1`, draft["supplier_id"])
		if err == sql.ErrNoRows {
			supplier = map[string]any{}
		} else if err != nil {
			rentalCatalogError(c, err)
			return
		}
		if op != "archive" && (supplier["is_supplier"] != true || supplier["is_archived"] != false) {
			required = append(required, "active_supplier_id")
		}
		if _, changed := fields["supplier_id"]; changed && supplier["name"] != nil {
			if in.Supplier != nil && !strings.EqualFold(fmt.Sprint(draft["supplier"]), fmt.Sprint(supplier["name"])) {
				required = append(required, "supplier_name_matches_id")
			} else {
				draft["supplier"] = supplier["name"]
				fields["supplier"] = supplier["name"]
			}
		} else if in.Supplier != nil && supplier["name"] != nil && !strings.EqualFold(fmt.Sprint(draft["supplier"]), fmt.Sprint(supplier["name"])) {
			required = append(required, "supplier_name_matches_id")
		}
	}
	if op == "update" {
		for _, key := range []string{"name", "category", "supplier"} {
			v, _ := draft[key].(string)
			if strings.TrimSpace(v) == "" {
				required = append(required, key)
			}
		}
		for _, key := range []string{"rental_price", "customer_price"} {
			v, exists := draft[key].(float64)
			if !exists || !validRentalPrice(&v) || v > 99999999.99 {
				required = append(required, key)
			}
		}
	}
	deps, err := rentalJobJSON(tx, `SELECT jsonb_build_object('active_jobs',rental_external_equipment_active_jobs($1::int),'ledger_count',(SELECT count(*) FROM job_rental_equipment WHERE equipment_id=$1),'position_count',(SELECT count(*) FROM job_positions WHERE rental_equipment_id=$1))`, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	activeJobs, _ := deps["active_jobs"].([]any)
	identityChanged := false
	for _, key := range []string{"name", "supplier", "supplier_id"} {
		if !rentalCatalogEqual(current[key], draft[key]) {
			identityChanged = true
		}
	}
	if len(activeJobs) > 0 && (op == "archive" || identityChanged) {
		required = append(required, "active_jobs")
	}
	if len(activeJobs) > 1000 {
		required = append(required, "bounded_dependencies")
	}
	custom := []map[string]any{}
	var customExists bool
	if err = tx.QueryRow(`SELECT to_regclass('rental_equipment_field_values') IS NOT NULL AND to_regclass('rental_equipment_field_definitions') IS NOT NULL`).Scan(&customExists); err != nil {
		rentalCatalogError(c, err)
		return
	}
	if customExists {
		if _, err = tx.Exec(`LOCK TABLE rental_equipment_field_definitions,rental_equipment_field_values IN SHARE MODE NOWAIT`); err != nil {
			rentalCatalogError(c, err)
			return
		}
		rows, e := tx.Query(`SELECT jsonb_build_object('value',to_jsonb(v),'definition',to_jsonb(d)) FROM rental_equipment_field_values v JOIN rental_equipment_field_definitions d ON d.id=v.field_definition_id WHERE v.equipment_id=$1 ORDER BY v.id LIMIT 1001`, in.EquipmentID)
		if e != nil {
			rentalCatalogError(c, e)
			return
		}
		for rows.Next() {
			var b json.RawMessage
			if e = rows.Scan(&b); e != nil {
				break
			}
			var item map[string]any
			if e = json.Unmarshal(b, &item); e != nil {
				break
			}
			custom = append(custom, item)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			rentalCatalogError(c, e)
			return
		}
		if len(custom) > 1000 {
			required = append(required, "bounded_custom_fields")
		}
	}
	candidates := []map[string]any{}
	if op == "update" && identityChanged {
		rows, e := tx.Query(`SELECT `+rentalExternalEquipmentRecordSQL+` FROM rental_equipment WHERE id<>$1 AND name ILIKE $2 ESCAPE '\' ORDER BY id LIMIT 51`, in.EquipmentID, "%"+escapeRentalMasterLike(fmt.Sprint(draft["name"]))+"%")
		if e != nil {
			rentalCatalogError(c, e)
			return
		}
		for rows.Next() {
			var b json.RawMessage
			if e = rows.Scan(&b); e != nil {
				break
			}
			var item map[string]any
			if e = json.Unmarshal(b, &item); e != nil {
				break
			}
			exact := strings.EqualFold(strings.TrimSpace(fmt.Sprint(item["name"])), fmt.Sprint(draft["name"])) && strings.EqualFold(strings.TrimSpace(fmt.Sprint(item["supplier"])), fmt.Sprint(draft["supplier"]))
			item["exact"] = exact
			candidates = append(candidates, item)
			if exact {
				required = append(required, "reuse_existing_equipment")
			}
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			rentalCatalogError(c, e)
			return
		}
		if len(candidates) > 50 {
			required = append(required, "narrow_duplicate_identity")
		}
		if len(candidates) > 0 && !in.AllowSimilar {
			required = append(required, "review_similar_and_allow_explicitly")
		}
	}
	diff := map[string]map[string]any{}
	for k, v := range draft {
		if !rentalCatalogEqual(current[k], v) {
			diff[k] = map[string]any{"before": current[k], "after": v}
		}
	}
	if op == "update" && len(diff) == 0 {
		required = append(required, "changes")
	}
	binding, _ := json.Marshal(map[string]any{"operation": op, "current": current, "draft": draft, "supplier": supplier, "dependencies": deps, "custom_fields": custom, "candidates": candidates, "allow_similar": in.AllowSimilar})
	digest := sha256.Sum256(binding)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL EXTERNAL EQUIPMENT %d %s", strings.ToUpper(op), in.EquipmentID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact record/draft-bound confirmation phrase required"})
		return
	}
	effects := map[string]any{"history_retained": true, "job_prices_changed": false, "supplier_costs_changed": false, "job_assignments_changed": false, "stock_movements": false, "custom_fields_preserved": true}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "supplier": supplier, "dependencies": deps, "custom_fields": custom, "possible_duplicates": candidates, "expected_updated_at": current["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase, "effects": effects}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	if op == "update" {
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		args := []any{in.EquipmentID}
		sets := []string{}
		for _, key := range keys {
			args = append(args, fields[key])
			sets = append(sets, rentalExternalCatalogColumns[key]+fmt.Sprintf("=$%d", len(args)))
		}
		_, err = tx.Exec(`UPDATE rental_equipment SET `+strings.Join(sets, ",")+` WHERE id=$1`, args...)
	} else {
		_, err = tx.Exec(`UPDATE rental_equipment SET is_active=$2 WHERE id=$1`, in.EquipmentID, op == "restore")
	}
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, `SELECT `+rentalExternalEquipmentRecordSQL+` FROM rental_equipment WHERE id=$1`, in.EquipmentID)
	if err != nil {
		rentalCatalogError(c, err)
		return
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "before": current, "after": record, "effects": effects})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,'rental_external_equipment',$3,$4::jsonb,$5::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.external_equipment."+op, fmt.Sprint(in.EquipmentID), string(before), string(after)); err != nil {
		rentalCatalogError(c, err)
		return
	}
	status := map[string]string{"update": "updated", "archive": "archived", "restore": "restored"}[op]
	response, _ := json.Marshal(map[string]any{"operation_status": status, "equipment": record, "effects": effects})
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

func rentalCatalogEqual(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
