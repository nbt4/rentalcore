package postalcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://openplzapi.org/de"

var ErrInvalidPostalCode = errors.New("postal code must contain exactly five digits")

var germanPostalCodePattern = regexp.MustCompile(`^\d{5}$`)

type cacheEntry struct {
	cities    []string
	expiresAt time.Time
}

// Client resolves German postal codes through OpenPLZ and caches results so
// repeated customer form lookups do not create repeated external requests.
type Client struct {
	baseURL  string
	http     *http.Client
	cacheTTL time.Duration
	cacheMu  sync.RWMutex
	cache    map[string]cacheEntry
}

func NewClient() *Client {
	baseURL := strings.TrimSpace(os.Getenv("POSTAL_LOOKUP_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return newClient(baseURL, &http.Client{Timeout: 4 * time.Second})
}

func newClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		http:     httpClient,
		cacheTTL: 24 * time.Hour,
		cache:    make(map[string]cacheEntry),
	}
}

func (c *Client) Lookup(ctx context.Context, postalCode string) ([]string, error) {
	postalCode = strings.TrimSpace(postalCode)
	if !germanPostalCodePattern.MatchString(postalCode) {
		return nil, ErrInvalidPostalCode
	}

	if cities, ok := c.cached(postalCode); ok {
		return cities, nil
	}

	endpoint, err := url.Parse(c.baseURL + "/Localities")
	if err != nil {
		return nil, fmt.Errorf("build postal lookup URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("postalCode", postalCode)
	query.Set("page", "1")
	query.Set("pageSize", "50")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build postal lookup request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("postal lookup request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("postal lookup returned status %d", response.StatusCode)
	}

	var localities []struct {
		Name       string `json:"name"`
		PostalCode string `json:"postalCode"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&localities); err != nil {
		return nil, fmt.Errorf("decode postal lookup: %w", err)
	}

	unique := make(map[string]struct{})
	for _, locality := range localities {
		name := strings.TrimSpace(locality.Name)
		if locality.PostalCode == postalCode && name != "" {
			unique[name] = struct{}{}
		}
	}
	cities := make([]string, 0, len(unique))
	for name := range unique {
		cities = append(cities, name)
	}
	sort.Strings(cities)
	c.store(postalCode, cities)
	return append([]string(nil), cities...), nil
}

func (c *Client) cached(postalCode string) ([]string, bool) {
	c.cacheMu.RLock()
	entry, ok := c.cache[postalCode]
	c.cacheMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return append([]string(nil), entry.cities...), true
}

func (c *Client) store(postalCode string, cities []string) {
	c.cacheMu.Lock()
	c.cache[postalCode] = cacheEntry{
		cities:    append([]string(nil), cities...),
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.cacheMu.Unlock()
}
