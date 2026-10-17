package handlers

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// Project only the HTTP response. Complete business context, audit snapshots and
// durable receipts retain their original values, including during replay.
func rentalJobFinancialResponse(c *gin.Context, status int, response []byte, financial bool) {
	if financial {
		c.Data(status, "application/json", response)
		return
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(response))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		masterError(c, err)
		return
	}
	omitted := false
	var project func(any)
	project = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			for _, key := range []string{"revenue", "final_revenue", "discount", "discount_type", "multiply_by_days", "prices_include_tax"} {
				if _, exists := node[key]; exists {
					delete(node, key)
					omitted = true
				}
			}
			for _, child := range node {
				project(child)
			}
		case []any:
			for _, child := range node {
				project(child)
			}
		}
	}
	project(value)
	if result, ok := value.(map[string]any); ok && omitted {
		result["financial_fields_omitted"] = true
		result["financial_access_scope"] = "cores:rental:financial"
	}
	c.JSON(status, value)
}
