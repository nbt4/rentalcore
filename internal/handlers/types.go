package handlers

import "go-barcode-webapp/internal/models"

// ProductGroup represents a group of devices organized by product
type ProductGroup struct {
	Product    *models.Product    `json:"product"`
	Devices    []models.JobDevice `json:"devices"`
	Count      int                `json:"count"`
	TotalValue float64            `json:"total_value"`
}