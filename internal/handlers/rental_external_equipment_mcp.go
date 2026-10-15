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
	"strings"

	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type rentalExternalEquipmentRequest struct {
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Supplier         string   `json:"supplier"`
	SupplierID       int64    `json:"supplier_id"`
	RentalPrice      *float64 `json:"rental_price"`
	CustomerPrice    *float64 `json:"customer_price"`
	Description      string   `json:"description"`
	Notes            string   `json:"notes"`
	AllowSimilar     bool     `json:"allow_similar"`
	ExpectedContext  string   `json:"expected_context"`
	ConfirmationText string   `json:"confirmation_text"`
	ConfirmChange    bool     `json:"confirm_change"`
	Preview          bool     `json:"preview"`
}

type RentalExternalEquipmentMCP struct{ db *sql.DB }

func NewRentalExternalEquipmentMCP(db *repository.Database) *RentalExternalEquipmentMCP {
	sqlDB, _ := db.DB.DB()
	return &RentalExternalEquipmentMCP{db: sqlDB}
}

const rentalExternalEquipmentRecordSQL = `jsonb_build_object('equipment_id',id,'name',name,'supplier',supplier,'supplier_id',supplier_id,'category',category,'description',description,'notes',notes,'rental_price',rental_price,'customer_price',customer_price,'is_active',is_active,'created_at',created_at,'updated_at',to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))`

func (h *RentalExternalEquipmentMCP) Create(c *gin.Context) {
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
	var in rentalExternalEquipmentRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 256*1024+1))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF || decoder.InputOffset() > 256*1024 {
		c.JSON(400, gin.H{"error": "One bounded typed external rental catalog object required"})
		return
	}
	if in.SupplierID < 0 || in.SupplierID > math.MaxInt32 {
		c.JSON(400, gin.H{"error": "Canonical supplier_id required"})
		return
	}
	for _, field := range []struct {
		value *string
		limit int
	}{{&in.Name, 255}, {&in.Supplier, 255}, {&in.Category, 100}, {&in.Description, 10000}, {&in.Notes, 10000}} {
		*field.value = strings.TrimSpace(*field.value)
		if len([]rune(*field.value)) > field.limit {
			c.JSON(400, gin.H{"error": "Catalog field exceeds documented length"})
			return
		}
	}
	for _, price := range []*float64{in.RentalPrice, in.CustomerPrice} {
		if price != nil && (!validRentalPrice(price) || *price > 99999999.99) {
			c.JSON(400, gin.H{"error": "Prices must be nonnegative, at most 99999999.99, with at most two decimal places"})
			return
		}
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
		const operation = "external_equipment.create"
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
	// Lock the referenced supplier before the catalog, matching owner job workflows.
	// Serializing catalog writers also closes the duplicate check/insert race.
	if _, err = tx.Exec(`LOCK TABLE customers IN SHARE MODE;LOCK TABLE rental_equipment IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		masterError(c, err)
		return
	}
	required := []string{}
	supplier := map[string]any{}
	if in.SupplierID != 0 {
		supplier, err = rentalJobJSON(tx, `SELECT jsonb_build_object('supplier_id',customerid,'name',COALESCE(NULLIF(companyname,''),NULLIF(name,''),trim(concat_ws(' ',firstname,lastname)))) FROM customers WHERE customerid=$1 AND is_supplier AND NOT COALESCE(is_archived,false)`, in.SupplierID)
		if err == sql.ErrNoRows {
			supplier = map[string]any{}
			required = append(required, "active_supplier_id")
		} else if err != nil {
			masterError(c, err)
			return
		} else if in.Supplier != "" && !strings.EqualFold(in.Supplier, fmt.Sprint(supplier["name"])) {
			required = append(required, "supplier_name_matches_id")
		} else {
			in.Supplier = fmt.Sprint(supplier["name"])
		}
	}
	for _, f := range []struct{ name, value string }{{"name", in.Name}, {"category", in.Category}, {"supplier", in.Supplier}} {
		if f.value == "" {
			required = append(required, f.name)
		}
	}
	if in.RentalPrice == nil {
		required = append(required, "rental_price")
	}
	if in.CustomerPrice == nil {
		required = append(required, "customer_price")
	}
	draft := map[string]any{"name": in.Name, "category": in.Category, "supplier": in.Supplier, "supplier_id": nil, "rental_price": in.RentalPrice, "customer_price": in.CustomerPrice, "description": in.Description, "notes": in.Notes, "is_active": true}
	if in.SupplierID != 0 {
		draft["supplier_id"] = in.SupplierID
	}
	candidates := []map[string]any{}
	rows, err := tx.Query(`SELECT `+rentalExternalEquipmentRecordSQL+` FROM rental_equipment WHERE $1<>'' AND name ILIKE $2 ESCAPE '\' ORDER BY id LIMIT 51`, in.Name, "%"+escapeRentalMasterLike(in.Name)+"%")
	if err != nil {
		masterError(c, err)
		return
	}
	for rows.Next() {
		var raw json.RawMessage
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			masterError(c, err)
			return
		}
		candidate := map[string]any{}
		if err = json.Unmarshal(raw, &candidate); err != nil {
			rows.Close()
			masterError(c, err)
			return
		}
		exact := strings.EqualFold(strings.TrimSpace(fmt.Sprint(candidate["name"])), in.Name) && strings.EqualFold(strings.TrimSpace(fmt.Sprint(candidate["supplier"])), in.Supplier)
		candidate["exact"] = exact
		candidates = append(candidates, candidate)
		if exact {
			required = append(required, "reuse_existing_equipment")
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		masterError(c, err)
		return
	}
	if len(candidates) > 50 {
		required = append(required, "narrow_duplicate_identity")
	}
	if len(candidates) > 0 && !in.AllowSimilar {
		required = append(required, "review_similar_and_allow_explicitly")
	}
	encoded, _ := json.Marshal(map[string]any{"operation": "external_equipment.create", "draft": draft, "supplier": supplier, "candidates": candidates, "allow_similar": in.AllowSimilar})
	digest := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(digest[:])
	if in.ExpectedContext != "" && in.ExpectedContext != fingerprint {
		required = append(required, "expected_context")
	}
	phrase := "CREATE RENTAL EXTERNAL EQUIPMENT " + fingerprint[:16]
	if !preview && len(required) == 0 && in.ConfirmationText != phrase {
		c.JSON(428, gin.H{"error": "Exact draft-bound confirmation phrase required"})
		return
	}
	effects := map[string]any{"catalog_item_created": true, "stock_movements": false, "warehouse_products_created": false, "job_assignments_created": false, "external_messages_sent": false}
	result := map[string]any{"operation_status": "confirmation_required", "preview": true, "ready_to_execute": len(required) == 0, "required_fields": required, "draft": draft, "supplier": supplier, "possible_duplicates": candidates, "expected_context": fingerprint, "required_confirmation_text": phrase, "effects": effects}
	if preview || len(required) > 0 {
		if len(required) > 0 {
			result["operation_status"] = "needs_input"
		}
		c.JSON(200, result)
		return
	}
	var id int64
	err = tx.QueryRow(`INSERT INTO rental_equipment(name,supplier,supplier_id,category,rental_price,customer_price,description,notes,is_active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,true) RETURNING id`, in.Name, in.Supplier, draft["supplier_id"], in.Category, *in.RentalPrice, *in.CustomerPrice, in.Description, in.Notes).Scan(&id)
	if err != nil {
		masterError(c, err)
		return
	}
	record, err := rentalJobJSON(tx, `SELECT `+rentalExternalEquipmentRecordSQL+` FROM rental_equipment WHERE id=$1`, id)
	if err != nil {
		masterError(c, err)
		return
	}
	after, _ := json.Marshal(map[string]any{"origin": "MCP/AI", "after": record, "effects": effects})
	if _, err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES($1,'rental.external_equipment.create','rental_external_equipment',$2,'{}'::jsonb,$3::jsonb,CURRENT_TIMESTAMP)`, user.UserID, fmt.Sprint(id), string(after)); err != nil {
		masterError(c, err)
		return
	}
	response, _ := json.Marshal(map[string]any{"operation_status": "created", "equipment": record, "effects": effects})
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
