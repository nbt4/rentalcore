package warehousecore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// RentalFieldDef describes a custom field definition from WarehouseCore
type RentalFieldDef struct {
	Name      string  `json:"name"`
	FieldType string  `json:"field_type"`
	Unit      *string `json:"unit"`
}

// RentalFieldValue is a custom field value attached to a rental equipment item
type RentalFieldValue struct {
	FieldDefinitionID int             `json:"field_definition_id"`
	Value             string          `json:"value"`
	SortOrder         int             `json:"sort_order"`
	Definition        *RentalFieldDef `json:"definition"`
}

// RentalEquipmentItem represents a rental equipment item from WarehouseCore
type RentalEquipmentItem struct {
	EquipmentID   uint               `json:"equipment_id"`
	ProductName   string             `json:"product_name"`
	SupplierName  string             `json:"supplier_name"`
	RentalPrice   float64            `json:"rental_price"`
	CustomerPrice float64            `json:"customer_price"`
	Category      string             `json:"category"`
	Description   string             `json:"description"`
	IsActive      bool               `json:"is_active"`
	FieldValues   []RentalFieldValue `json:"field_values"`
}

// Client is a client for communicating with WarehouseCore API
type Client struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	cache      []RentalEquipmentItem
	cacheTime  time.Time
	cacheTTL   time.Duration
}

// NewClient creates a new WarehouseCore client
func NewClient() *Client {
	// Get the WarehouseCore domain from environment variable
	domain := os.Getenv("WAREHOUSECORE_DOMAIN")
	if domain == "" {
		// Fallback for development
		domain = "localhost:8082"
	}

	// Determine protocol based on domain
	protocol := "https"
	if strings.Contains(domain, "localhost") || strings.Contains(domain, "127.0.0.1") {
		protocol = "http"
	}

	baseURL := fmt.Sprintf("%s://%s", protocol, strings.TrimSuffix(domain, "/"))

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: 5 * time.Minute,
	}
}

// NewClientWithURL creates a client with a specific base URL
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: 5 * time.Minute,
	}
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetRentalEquipment fetches rental equipment from WarehouseCore
func (c *Client) GetRentalEquipment() ([]RentalEquipmentItem, error) {
	// Check cache first
	c.mu.RLock()
	if len(c.cache) > 0 && time.Since(c.cacheTime) < c.cacheTTL {
		result := make([]RentalEquipmentItem, len(c.cache))
		copy(result, c.cache)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Fetch from WarehouseCore
	url := fmt.Sprintf("%s/api/v1/rental-equipment", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rental equipment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rental equipment API returned status %d", resp.StatusCode)
	}

	var items []RentalEquipmentItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode rental equipment: %w", err)
	}

	// Update cache
	c.mu.Lock()
	c.cache = items
	c.cacheTime = time.Now()
	c.mu.Unlock()

	return items, nil
}

// GetActiveRentalEquipment fetches only active rental equipment
func (c *Client) GetActiveRentalEquipment() ([]RentalEquipmentItem, error) {
	items, err := c.GetRentalEquipment()
	if err != nil {
		return nil, err
	}

	// Filter active items
	active := make([]RentalEquipmentItem, 0, len(items))
	for _, item := range items {
		if item.IsActive {
			active = append(active, item)
		}
	}

	return active, nil
}

// GetRentalEquipmentBySupplier returns rental equipment grouped by supplier
func (c *Client) GetRentalEquipmentBySupplier() (map[string][]RentalEquipmentItem, error) {
	items, err := c.GetActiveRentalEquipment()
	if err != nil {
		return nil, err
	}

	// Group by supplier
	bySupplier := make(map[string][]RentalEquipmentItem)
	for _, item := range items {
		supplier := item.SupplierName
		if supplier == "" {
			supplier = "Unknown Supplier"
		}
		bySupplier[supplier] = append(bySupplier[supplier], item)
	}

	return bySupplier, nil
}

// ClearCache clears the cached rental equipment data
func (c *Client) ClearCache() {
	c.mu.Lock()
	c.cache = nil
	c.cacheTime = time.Time{}
	c.mu.Unlock()
}
