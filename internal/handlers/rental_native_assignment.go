package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Native editors have no guided confirmation/receipt contract for delegated grants.
func rejectDelegatedRentalPosition(c *gin.Context) bool {
	delegated := strings.EqualFold(c.GetHeader("X-Cores-Origin"), "MCP/AI")
	if cookie, err := c.Cookie("cores_token"); err == nil && os.Getenv("CORES_JWT_SECRET") != "" {
		token, e := jwt.Parse(cookie, func(*jwt.Token) (any, error) { return []byte(os.Getenv("CORES_JWT_SECRET")), nil }, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}), jwt.WithoutClaimsValidation())
		if e == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				_, present := claims["mcp_scope"]
				delegated = delegated || present
			}
		}
	}
	if delegated {
		c.AbortWithStatusJSON(403, gin.H{"error": "Use the named confirmed rental owner workflow for delegated position/assignment writes"})
		return true
	}
	return false
}

func nativeRentalAssignmentSnapshot(tx *gorm.DB, positionID uint) (json.RawMessage, error) {
	var raw []byte
	err := tx.Raw(`SELECT jsonb_build_object('position',to_jsonb(p),'assignment',(SELECT to_jsonb(c) FROM job_rental_equipment c WHERE c.position_id=p.position_id)) FROM job_positions p WHERE p.position_id=?`, positionID).Row().Scan(&raw)
	return raw, err
}
func auditNativeRentalAssignment(tx *gorm.DB, c *gin.Context, positionID, jobID uint, op string, before json.RawMessage) error {
	user, ok := GetCurrentUser(c)
	if !ok || user == nil || user.UserID == 0 {
		return fmt.Errorf("interactive user required for native rental audit")
	}
	after, err := nativeRentalAssignmentSnapshot(tx, positionID)
	if err != nil {
		return err
	}
	var equipmentID int64
	if err = tx.Raw(`SELECT rental_equipment_id FROM job_positions WHERE position_id=?`, positionID).Row().Scan(&equipmentID); err != nil {
		return err
	}
	var payload map[string]json.RawMessage
	if err = json.Unmarshal(after, &payload); err != nil {
		return err
	}
	raw, err := json.Marshal(map[string]any{"origin": "UI", "after": payload["assignment"], "position": payload["position"]})
	if err != nil {
		return err
	}
	entityID := fmt.Sprintf("%d/%d", jobID, equipmentID)
	if err = tx.Exec(`INSERT INTO audit_log(user_id,action,entity_type,entity_id,old_values,new_values,timestamp) VALUES(?,?,'rental_job_external_equipment',?,?::jsonb,?::jsonb,CURRENT_TIMESTAMP)`, user.UserID, "rental.job_external_equipment."+op, entityID, string(before), string(raw)).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO job_history(job_id,user_id,change_type,field_name,old_value,new_value,description) VALUES(?,?,'updated',?,?,?,?)`, jobID, user.UserID, "ui.job_external_equipment."+op, string(before), string(raw), "Native retained rental assignment "+op).Error
}
