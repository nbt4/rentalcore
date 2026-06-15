package pdf

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go-barcode-webapp/internal/models"

	"go-barcode-webapp/internal/logger"
)

type packageAliasAPIEntry struct {
	Alias       string   `json:"alias"`
	PackageID   int      `json:"package_id"`
	PackageCode string   `json:"package_code"`
	PackageName string   `json:"package_name"`
	Price       *float64 `json:"price"`
}

type aliasRecord struct {
	aliasKey string
	codeKey  string
	data     packageAliasAPIEntry
}

// PackageAliasCache caches alias map responses from WarehouseCore for quick lookups
type PackageAliasCache struct {
	endpoint    string
	client      *http.Client
	mu          sync.RWMutex
	entries     []aliasRecord
	lastRefresh time.Time
	ttl         time.Duration
}

// NewPackageAliasCache initializes the cache (nil if endpoint is empty)
func NewPackageAliasCache(endpoint string) *PackageAliasCache {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}

	return &PackageAliasCache{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		ttl: 15 * time.Minute,
	}
}

// Enabled returns true when the cache is configured
func (c *PackageAliasCache) Enabled() bool {
	return c != nil && c.endpoint != ""
}

// Warm forces an initial fetch to populate the in-memory cache
func (c *PackageAliasCache) Warm() {
	if c == nil {
		return
	}
	if err := c.refresh(); err != nil {
		logger.LogInfo("warning: failed to warm WarehouseCore alias cache: %v", err)
	}
}

// FindMatches tries to match the provided text against the alias map
func (c *PackageAliasCache) FindMatches(productText string, limit int) []models.ProductMappingSuggestion {
	if c == nil || limit <= 0 {
		return nil
	}

	text := strings.TrimSpace(productText)
	if text == "" {
		return nil
	}

	if err := c.ensureFresh(); err != nil {
		logger.LogInfo("warning: unable to refresh WarehouseCore alias cache: %v", err)
	}

	c.mu.RLock()
	entries := make([]aliasRecord, len(c.entries))
	copy(entries, c.entries)
	c.mu.RUnlock()

	if len(entries) == 0 {
		return nil
	}

	normalized := normalizeProductText(text)
	if normalized == "" {
		return nil
	}

	type scoredSuggestion struct {
		score      float64
		suggestion models.ProductMappingSuggestion
	}

	candidates := make([]scoredSuggestion, 0, len(entries))
	for _, entry := range entries {
		score := entry.matchScore(normalized)
		if score <= 0 {
			continue
		}

		suggestion := models.ProductMappingSuggestion{
			RawProductText: text,
			Confidence:     score,
			MappingType:    "package",
			PackageID:      intPtr(entry.data.PackageID),
			PackageCode:    entry.data.PackageCode,
			PackageName:    entry.data.PackageName,
		}
		if entry.data.Price != nil {
			suggestion.PackagePrice = entry.data.Price
		}

		candidates = append(candidates, scoredSuggestion{
			score:      score,
			suggestion: suggestion,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]models.ProductMappingSuggestion, len(candidates))
	for idx, candidate := range candidates {
		results[idx] = candidate.suggestion
	}
	return results
}

// FindBestMatch returns the single best alias hit (if any)
func (c *PackageAliasCache) FindBestMatch(productText string) *models.ProductMappingSuggestion {
	matches := c.FindMatches(productText, 1)
	if len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

func (c *PackageAliasCache) ensureFresh() error {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	stale := len(c.entries) == 0 || time.Since(c.lastRefresh) > c.ttl
	c.mu.RUnlock()

	if !stale {
		return nil
	}
	return c.refresh()
}

func (c *PackageAliasCache) refresh() error {
	if c == nil {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("build alias map request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch alias map: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alias map returned status %d", resp.StatusCode)
	}

	var payload []packageAliasAPIEntry
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode alias map: %w", err)
	}

	records := make([]aliasRecord, 0, len(payload))
	for _, entry := range payload {
		aliasKey := normalizeProductText(entry.Alias)
		codeKey := normalizeProductText(entry.PackageCode)
		if aliasKey == "" && codeKey == "" {
			continue
		}
		records = append(records, aliasRecord{
			aliasKey: aliasKey,
			codeKey:  codeKey,
			data:     entry,
		})
	}

	c.mu.Lock()
	c.entries = records
	c.lastRefresh = time.Now()
	c.mu.Unlock()
	return nil
}

func (entry aliasRecord) matchScore(normalizedText string) float64 {
	if normalizedText == "" {
		return 0
	}

	best := scoreMatch(normalizedText, entry.aliasKey)
	if best == 0 && entry.codeKey != "" {
		best = math.Max(best, scoreMatch(normalizedText, entry.codeKey))
	}
	return best
}

func scoreMatch(text, key string) float64 {
	if key == "" || text == "" {
		return 0
	}
	if text == key {
		return 100.0
	}
	if strings.Contains(text, key) {
		ratio := float64(len(key)) / float64(len(text))
		return clampScore(70.0 + ratio*30.0)
	}
	if strings.Contains(key, text) {
		ratio := float64(len(text)) / float64(len(key))
		return clampScore(55.0 + ratio*35.0)
	}
	return 0
}

func clampScore(score float64) float64 {
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func intPtr(value int) *int {
	return &value
}
