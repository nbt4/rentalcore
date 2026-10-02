package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go-barcode-webapp/internal/repository"
)

// The public endpoints are closed customer/venue workflows, never SQL or a
// generic record writer. Wire field names and columns are fixed below.
type rentalMasterFields struct {
	Name         *string `json:"name"`
	CompanyName  *string `json:"company_name"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Street       *string `json:"street"`
	HouseNumber  *string `json:"house_number"`
	ZIP          *string `json:"zip"`
	City         *string `json:"city"`
	FederalState *string `json:"federal_state"`
	Country      *string `json:"country"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	CustomerType *string `json:"customer_type"`
	IsCustomer   *bool   `json:"is_customer"`
	IsSupplier   *bool   `json:"is_supplier"`
	ContactName  *string `json:"contact_name"`
	Notes        *string `json:"notes"`
}
type rentalMasterRequest struct {
	rentalMasterFields
	ID                int64  `json:"id"`
	AuditID           int64  `json:"audit_id,omitempty"`
	ExpectedUpdatedAt string `json:"expected_updated_at"`
	ExpectedContext   string `json:"expected_context"`
	AllowDuplicate    bool   `json:"allow_duplicate"`
	ConfirmChange     bool   `json:"confirm_change"`
	ConfirmationText  string `json:"confirmation_text"`
	Preview           bool   `json:"preview"`
}
type RentalMasterMCP struct{ db *sql.DB }

func NewRentalMasterMCP(db *repository.Database) *RentalMasterMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalMasterMCP{db: sqlDB}
}
func (h *RentalMasterMCP) Customer(c *gin.Context) { h.change(c, "customer") }
func (h *RentalMasterMCP) Venue(c *gin.Context)    { h.change(c, "venue") }

var rentalMasterKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
var rentalMasterContextPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type rentalMasterSpec struct {
	table, pk, identity string
	columns             map[string]string
	limits              map[string]int
}

func masterSpec(kind string) rentalMasterSpec {
	if kind == "venue" {
		return rentalMasterSpec{"venues", "id", "name", map[string]string{"name": "name", "street": "street", "house_number": "house_number", "zip": "zip", "city": "city", "contact_name": "contact_name", "phone": "phone", "email": "email", "notes": "notes"}, map[string]int{"house_number": 50, "zip": 20, "phone": 100, "notes": 10000}}
	}
	return rentalMasterSpec{"customers", "customerid", "COALESCE(NULLIF(companyname,''),NULLIF(name,''),trim(concat_ws(' ',firstname,lastname)))", map[string]string{"name": "name", "company_name": "companyname", "first_name": "firstname", "last_name": "lastname", "street": "street", "house_number": "housenumber", "zip": "zip", "city": "city", "federal_state": "federalstate", "country": "country", "phone": "phonenumber", "email": "email", "customer_type": "customertype", "is_customer": "is_customer", "is_supplier": "is_supplier", "notes": "notes"}, map[string]int{"first_name": 100, "last_name": 100, "house_number": 20, "zip": 20, "city": 100, "federal_state": 100, "country": 100, "phone": 50, "customer_type": 50, "notes": 10000}}
}
func masterKeys(s rentalMasterSpec) []string {
	keys := make([]string, 0, len(s.columns))
	for k := range s.columns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func masterRecordSQL(s rentalMasterSpec) string {
	fields := []string{"'id'," + s.pk, "'is_archived',COALESCE(is_archived,false)", "'archived_at',archived_at", "'created_at',created_at", "'updated_at',to_char(updated_at,'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')"}
	for _, k := range masterKeys(s) {
		fields = append(fields, "'"+k+"',"+s.columns[k])
	}
	return "jsonb_build_object(" + strings.Join(fields, ",") + ")"
}
func masterIdentity(kind string, fields map[string]any) string {
	str := func(k string) string { s, _ := fields[k].(string); return strings.TrimSpace(s) }
	if kind == "venue" {
		return str("name")
	}
	for _, k := range []string{"company_name", "name"} {
		if s := str(k); s != "" {
			return s
		}
	}
	return strings.TrimSpace(str("first_name") + " " + str("last_name"))
}
func masterError(c *gin.Context, err error) {
	c.JSON(500, gin.H{"error": "Rental master transaction failed; no data was committed"})
}

func (h *RentalMasterMCP) change(c *gin.Context, kind string) {
	op := c.Param("operation")
	if !map[string]bool{"create": true, "update": true, "archive": true, "restore": true, "revert_update": true}[op] {
		c.JSON(404, gin.H{"error": "Unknown rental master operation"})
		return
	}
	user, ok := GetCurrentUser(c)
	if !ok || user.UserID == 0 || !user.IsAdmin || !user.IsActive || !strings.EqualFold(c.GetHeader("X-Cores-Origin"), "MCP/AI") {
		c.JSON(403, gin.H{"error": "An active signed-in rental administrator and MCP origin are required"})
		return
	}
	// An authenticated browser user cannot forge the delegated action with an
	// HTTP header. The MCP signs the selected action into the short-lived cookie.
	cookie, _ := c.Cookie("cores_token")
	claims := struct {
		UserID uint   `json:"uid"`
		Scope  string `json:"mcp_scope"`
		jwt.RegisteredClaims
	}{}
	tok, err := jwt.ParseWithClaims(cookie, &claims, func(t *jwt.Token) (any, error) { return []byte(os.Getenv("CORES_JWT_SECRET")), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	action := op
	if op == "revert_update" {
		action = "update"
	}
	if op == "restore" {
		action = "archive"
	}
	if err != nil || tok == nil || !tok.Valid || os.Getenv("CORES_JWT_SECRET") == "" || claims.UserID != user.UserID || claims.Scope != "cores:rental:"+action {
		c.JSON(403, gin.H{"error": "The matching signed rental action scope is required"})
		return
	}
	var in rentalMasterRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 32*1024))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil {
		c.JSON(400, gin.H{"error": "Invalid bounded rental master fields"})
		return
	}
	if decoder.Decode(new(any)) != io.EOF || in.ID < 0 || in.ID > 2147483647 || op == "create" && in.ID != 0 || op != "create" && in.ID == 0 {
		c.JSON(400, gin.H{"error": "One object and the correct canonical record ID are required"})
		return
	}
	preview := in.Preview || !in.ConfirmChange
	if in.AuditID < 0 || (in.AuditID != 0 && op != "revert_update") {
		c.JSON(400, gin.H{"error": "audit_id is restricted to the named field revert"})
		return
	}
	if !preview && !rentalMasterContextPattern.MatchString(in.ExpectedContext) {
		c.JSON(428, gin.H{"error": "Exact expected_context from the final preview required"})
		return
	}
	spec := masterSpec(kind)
	rawFields, _ := json.Marshal(in.rentalMasterFields)
	fields := map[string]any{}
	_ = json.Unmarshal(rawFields, &fields)
	for key, v := range fields {
		if v == nil {
			delete(fields, key)
			continue
		}
		if _, allowed := spec.columns[key]; !allowed || op == "archive" || op == "restore" || op == "revert_update" {
			c.JSON(400, gin.H{"error": "Field is not allowed for this entity/operation", "field": key})
			return
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			limit := spec.limits[key]
			if limit == 0 {
				limit = 255
			}
			if len([]rune(s)) > limit {
				c.JSON(400, gin.H{"error": "Field exceeds its documented maximum length", "field": key})
				return
			}
			if key == "email" && s != "" {
				address, e := mail.ParseAddress(s)
				if e != nil || address.Address != s {
					c.JSON(400, gin.H{"error": "A plain valid business email address is required"})
					return
				}
			}
			if key == "customer_type" && s != "" && s != "Unternehmen" && s != "Privat" {
				c.JSON(400, gin.H{"error": "customer_type must be Unternehmen, Privat or empty"})
				return
			}
			if s == "" {
				fields[key] = nil
			} else {
				fields[key] = s
			}
		}
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		masterError(c, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='15s';SET LOCAL TIME ZONE 'UTC'`); err != nil {
		masterError(c, err)
		return
	}
	var actualAdmin bool
	if err = tx.QueryRow(`SELECT is_admin AND is_active FROM users WHERE userid=$1 FOR SHARE`, user.UserID).Scan(&actualAdmin); err != nil || !actualAdmin {
		c.JSON(403, gin.H{"error": "Current rental administrator rights required"})
		return
	}
	receiptID := int64(0)
	if !preview {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if !rentalMasterKeyPattern.MatchString(key) {
			c.JSON(428, gin.H{"error": "A valid Idempotency-Key is required"})
			return
		}
		encoded, _ := json.Marshal(in)
		keyDigest := sha256.Sum256([]byte(key))
		digest := sha256.Sum256(encoded)
		keyHash, requestHash := hex.EncodeToString(keyDigest[:]), hex.EncodeToString(digest[:])
		operation := kind + "." + op
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
				c.JSON(409, gin.H{"error": "Idempotency key already used with different fields"})
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
	if _, err = tx.Exec(`LOCK TABLE customers,venues IN SHARE ROW EXCLUSIVE MODE;LOCK TABLE jobs,status IN SHARE MODE`); err != nil {
		masterError(c, err)
		return
	}
	if op == "revert_update" {
		if _, err = tx.Exec(`LOCK TABLE audit_log IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			masterError(c, err)
			return
		}
	}
	current := map[string]any{}
	draft := map[string]any{}
	required := []string{}
	if op != "create" {
		var raw json.RawMessage
		if err = tx.QueryRow("SELECT "+masterRecordSQL(spec)+" FROM "+spec.table+" WHERE "+spec.pk+"=$1", in.ID).Scan(&raw); err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Rental master not found"})
			return
		} else if err != nil {
			masterError(c, err)
			return
		}
		if err = json.Unmarshal(raw, &current); err != nil {
			masterError(c, err)
			return
		}
		for k, v := range current {
			draft[k] = v
		}
		expectedArchived := op == "restore"
		if current["is_archived"] != expectedArchived {
			required = append(required, "lifecycle_status")
		}
		if !preview && in.ExpectedUpdatedAt == "" || in.ExpectedUpdatedAt != "" && in.ExpectedUpdatedAt != current["updated_at"] {
			required = append(required, "expected_updated_at")
		}
	} else {
		for k := range spec.columns {
			draft[k] = nil
		}
		draft["is_archived"] = false
		if kind == "customer" {
			draft["is_customer"] = true
			draft["is_supplier"] = false
		}
	}
	for k, v := range fields {
		draft[k] = v
	}
	if op == "archive" {
		draft["is_archived"] = true
	}
	if op == "restore" {
		draft["is_archived"] = false
	}
	sourceAudit := map[string]any{}
	if op == "revert_update" {
		var auditID int64
		var actor sql.NullInt64
		var auditAction string
		var before, after json.RawMessage
		var changedAt time.Time
		err = tx.QueryRow(`SELECT id,user_id,action,old_values,new_values,timestamp FROM audit_log WHERE entity_type=$1 AND entity_id=$2 ORDER BY id DESC LIMIT 1`, "rental_"+kind, fmt.Sprint(in.ID)).Scan(&auditID, &actor, &auditAction, &before, &after, &changedAt)
		if err == sql.ErrNoRows {
			required = append(required, "own_last_update")
		} else if err != nil {
			masterError(c, err)
			return
		} else {
			oldRecord, newValues := map[string]any{}, map[string]any{}
			if json.Unmarshal(before, &oldRecord) != nil || json.Unmarshal(after, &newValues) != nil {
				required = append(required, "valid_update_history")
			} else {
				afterRecord, _ := newValues["after"].(map[string]any)
				sourceAudit = map[string]any{"audit_id": auditID, "user_id": actor.Int64, "action": auditAction, "changed_at": changedAt, "result_version": newValues["updated_at"]}
				if !actor.Valid || actor.Int64 != int64(user.UserID) || auditAction != "rental."+kind+".update" || newValues["origin"] != "MCP/AI" {
					required = append(required, "own_last_update")
				}
				if afterRecord == nil || newValues["updated_at"] != current["updated_at"] {
					required = append(required, "unchanged_update_version")
				}
				if (!preview && in.AuditID == 0) || (in.AuditID != 0 && in.AuditID != auditID) {
					required = append(required, "audit_id")
				}
				if !rentalMasterRevertFieldsValid(spec, oldRecord) {
					required = append(required, "valid_update_history")
				}
				for _, key := range masterKeys(spec) {
					value, present := oldRecord[key]
					if !present {
						required = append(required, "complete_update_history")
						break
					}
					draft[key] = value
					fields[key] = value
				}
			}
		}
	}
	identity := masterIdentity(kind, draft)
	if identity == "" {
		required = append(required, "name")
	}
	if kind == "customer" && draft["is_customer"] != true && draft["is_supplier"] != true {
		required = append(required, "customer_or_supplier_role")
	}
	if op == "update" && len(fields) == 0 {
		required = append(required, "changed_fields")
	}
	email, _ := draft["email"].(string)
	candidates := []map[string]any{}
	rows, e := tx.Query("SELECT "+spec.pk+","+spec.identity+",COALESCE(is_archived,false),to_char(updated_at,'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),(lower(trim("+spec.identity+"))=lower($2) OR ($3<>'' AND lower(COALESCE(email,''))=lower($3))) AS exact FROM "+spec.table+" WHERE "+spec.pk+"<>$1 AND (lower("+spec.identity+") LIKE $4 OR ($3<>'' AND lower(COALESCE(email,''))=lower($3))) ORDER BY exact DESC,"+spec.pk+" LIMIT 51", in.ID, identity, email, "%"+strings.ToLower(escapeRentalMasterLike(identity))+"%")
	if e != nil {
		masterError(c, e)
		return
	}
	for rows.Next() {
		var id int64
		var name, version string
		var archived, exact bool
		if e = rows.Scan(&id, &name, &archived, &version, &exact); e != nil {
			rows.Close()
			masterError(c, e)
			return
		}
		candidates = append(candidates, map[string]any{"id": id, "name": name, "is_archived": archived, "updated_at": version, "exact": exact})
		if exact && (op == "create" || op == "update" || op == "restore" || op == "revert_update") {
			if archived && op == "create" {
				required = append(required, "restoration_required")
			} else if !in.AllowDuplicate {
				required = append(required, "review_duplicate_and_allow_explicitly")
			}
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		masterError(c, e)
		return
	}
	if len(candidates) > 50 {
		required = append(required, "narrow_duplicate_identity")
	}
	jobs := json.RawMessage(`[]`)
	if op != "create" {
		if err = tx.QueryRow(`SELECT rental_master_active_jobs($1,$2::int)`, kind, in.ID).Scan(&jobs); err != nil {
			masterError(c, err)
			return
		}
	}
	var jobRecords []any
	if err = json.Unmarshal(jobs, &jobRecords); err != nil {
		masterError(c, err)
		return
	}
	if op == "archive" && len(jobRecords) > 0 {
		required = append(required, "active_jobs")
	}
	diff := map[string]any{}
	for k, v := range draft {
		if !reflect.DeepEqual(current[k], v) {
			diff[k] = map[string]any{"before": current[k], "after": v}
		}
	}
	contextBytes, _ := json.Marshal(map[string]any{"kind": kind, "operation": op, "current": current, "draft": draft, "active_jobs": jobRecords, "candidates": candidates, "allow_duplicate": in.AllowDuplicate, "source_audit": sourceAudit})
	contextDigest := sha256.Sum256(contextBytes)
	fingerprint := hex.EncodeToString(contextDigest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := fmt.Sprintf("%s RENTAL %s %d %s", strings.ToUpper(op), strings.ToUpper(kind), in.ID, fingerprint[:16])
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact record/draft-bound confirmation phrase required"})
		return
	}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "current": current, "draft": draft, "diff": diff, "active_jobs": jobRecords, "similar_records": candidates, "expected_updated_at": current["updated_at"], "expected_context": fingerprint, "required_confirmation_text": phrase, "effects": map[string]any{"history_retained": true, "historical_jobs_changed": false, "external_messages_sent": false}}
	if op == "revert_update" {
		result["source_audit"] = sourceAudit
		result["expected_audit_id"] = sourceAudit["audit_id"]
	}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	recordID := in.ID
	if op == "create" {
		columns, placeholders := []string{}, []string{}
		args := []any{}
		for _, k := range masterKeys(spec) {
			columns = append(columns, spec.columns[k])
			args = append(args, draft[k])
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		err = tx.QueryRow("INSERT INTO "+spec.table+"("+strings.Join(columns, ",")+") VALUES("+strings.Join(placeholders, ",")+") RETURNING "+spec.pk, args...).Scan(&recordID)
	} else if op == "update" || op == "revert_update" {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		assignments := []string{}
		args := []any{recordID}
		for _, k := range keys {
			args = append(args, fields[k])
			assignments = append(assignments, spec.columns[k]+fmt.Sprintf("=$%d", len(args)))
		}
		_, err = tx.Exec("UPDATE "+spec.table+" SET "+strings.Join(assignments, ",")+" WHERE "+spec.pk+"=$1", args...)
	} else {
		_, err = tx.Exec("UPDATE "+spec.table+" SET is_archived=$2 WHERE "+spec.pk+"=$1", recordID, op == "archive")
	}
	if err != nil {
		masterError(c, err)
		return
	}
	var after json.RawMessage
	if err = tx.QueryRow("SELECT "+masterRecordSQL(spec)+" FROM "+spec.table+" WHERE "+spec.pk+"=$1", recordID).Scan(&after); err != nil {
		masterError(c, err)
		return
	}
	var record map[string]any
	if err = json.Unmarshal(after, &record); err != nil {
		masterError(c, err)
		return
	}
	before, _ := json.Marshal(current)
	auditValues := map[string]any{"origin": "MCP/AI", "after": record, "updated_at": record["updated_at"]}
	if op == "revert_update" {
		auditValues["reverted_audit_id"] = sourceAudit["audit_id"]
	}
	newAudit, _ := json.Marshal(auditValues)
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental."+kind+"."+op, "rental_"+kind, fmt.Sprint(recordID), string(before), string(newAudit)); err != nil {
		masterError(c, err)
		return
	}
	result = map[string]any{"operation_status": op + "d", "record": record, "diff": diff, "effects": result["effects"]}
	if op == "revert_update" {
		result["operation_status"] = "reverted"
		result["reverted_audit_id"] = sourceAudit["audit_id"]
	}
	response, _ := json.Marshal(result)
	status := 200
	if op == "create" {
		status = 201
	}
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
func escapeRentalMasterLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Legacy before-values must satisfy current business validation before undo.
func rentalMasterRevertFieldsValid(spec rentalMasterSpec, values map[string]any) bool {
	for _, key := range masterKeys(spec) {
		value, present := values[key]
		if !present {
			return false
		}
		if key == "is_customer" || key == "is_supplier" {
			if _, ok := value.(bool); !ok {
				return false
			}
			continue
		}
		if value == nil {
			continue
		}
		str, ok := value.(string)
		if !ok {
			return false
		}
		limit := spec.limits[key]
		if limit == 0 {
			limit = 255
		}
		if len([]rune(str)) > limit {
			return false
		}
		if key == "email" && str != "" {
			address, err := mail.ParseAddress(str)
			if err != nil || address.Address != str {
				return false
			}
		}
		if key == "customer_type" && str != "" && str != "Unternehmen" && str != "Privat" {
			return false
		}
	}
	return true
}
