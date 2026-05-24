package m365

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// GraphClient hält OAuth2-Token und führt Graph-API-Calls durch.
type GraphClient struct {
	tenantID     string
	clientID     string
	clientSecret string
	mailboxID    string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time

	httpClient *http.Client
}

// M365Contact entspricht dem Microsoft Graph contact-Objekt.
type M365Contact struct {
	ID                   string       `json:"id,omitempty"`
	GivenName            string       `json:"givenName,omitempty"`
	Surname              string       `json:"surname,omitempty"`
	CompanyName          string       `json:"companyName,omitempty"`
	EmailAddresses       []EmailAddr  `json:"emailAddresses,omitempty"`
	BusinessPhones       []string     `json:"businessPhones,omitempty"`
	BusinessAddress      Address      `json:"businessAddress,omitempty"`
	PersonalNotes        string       `json:"personalNotes,omitempty"`
	LastModifiedDateTime string       `json:"lastModifiedDateTime,omitempty"`
	Removed              *RemovedInfo `json:"@removed,omitempty"`
}

type EmailAddr struct {
	Address string `json:"address"`
}

type Address struct {
	Street          string `json:"street,omitempty"`
	PostalCode      string `json:"postalCode,omitempty"`
	City            string `json:"city,omitempty"`
	CountryOrRegion string `json:"countryOrRegion,omitempty"`
}

// RemovedInfo erscheint in Delta-Responses für gelöschte Kontakte.
type RemovedInfo struct {
	Reason string `json:"reason"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewGraphClient(tenantID, clientID, clientSecret, mailboxID string) *GraphClient {
	return &GraphClient{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		mailboxID:    mailboxID,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GraphClient) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	}

	resp, err := c.httpClient.PostForm(
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.tenantID),
		data,
	)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("token decode failed: %w", err)
	}

	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return c.token, nil
}

func (c *GraphClient) doRequest(method, reqURL string, body interface{}) (*http.Response, error) {
	return c.doRequestWithHeaders(method, reqURL, body, nil)
}

func (c *GraphClient) doRequestWithHeaders(method, reqURL string, body interface{}, extraHeaders map[string]string) (*http.Response, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// CreateContact legt einen neuen Kontakt im Shared Mailbox an und gibt die M365-ID zurück.
func (c *GraphClient) CreateContact(contact M365Contact) (string, error) {
	reqURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/contacts", c.mailboxID)
	resp, err := c.doRequest("POST", reqURL, contact)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create contact: HTTP %d", resp.StatusCode)
	}

	var created M365Contact
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// UpdateContact aktualisiert einen bestehenden Kontakt (PATCH).
func (c *GraphClient) UpdateContact(contactID string, contact M365Contact) error {
	reqURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/contacts/%s", c.mailboxID, contactID)
	resp, err := c.doRequest("PATCH", reqURL, contact)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update contact: HTTP %d", resp.StatusCode)
	}
	return nil
}

// DeleteContact löscht einen Kontakt in M365.
func (c *GraphClient) DeleteContact(contactID string) error {
	reqURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/contacts/%s", c.mailboxID, contactID)
	resp, err := c.doRequest("DELETE", reqURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete contact: HTTP %d", resp.StatusCode)
	}
	return nil
}

type deltaResponse struct {
	Value     []M365Contact `json:"value"`
	NextLink  string        `json:"@odata.nextLink"`
	DeltaLink string        `json:"@odata.deltaLink"`
}

// GetDelta holt Änderungen seit dem letzten Delta-Token.
// Leerer deltaToken = Erst-Sync (alle Kontakte).
func (c *GraphClient) GetDelta(deltaToken string) (contacts []M365Contact, newToken string, err error) {
	var reqURL string
	if deltaToken == "" {
		reqURL = fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/contacts/delta", c.mailboxID)
	} else {
		reqURL = fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/contacts/delta?$deltaToken=%s",
			c.mailboxID, deltaToken)
	}

	for reqURL != "" {
		resp, err := c.doRequest("GET", reqURL, nil)
		if err != nil {
			return nil, "", err
		}

		var dr deltaResponse
		decErr := json.NewDecoder(resp.Body).Decode(&dr)
		resp.Body.Close()
		if decErr != nil {
			return nil, "", decErr
		}

		contacts = append(contacts, dr.Value...)

		if dr.DeltaLink != "" {
			newToken = ExtractDeltaToken(dr.DeltaLink)
			reqURL = ""
		} else {
			reqURL = dr.NextLink
		}
	}

	return contacts, newToken, nil
}

// ExtractDeltaToken extrahiert den Token-Wert aus einem Delta-Link.
// Exportiert für Tests.
func ExtractDeltaToken(deltaLink string) string {
	const prefix = "$deltaToken="
	if idx := strings.Index(deltaLink, prefix); idx != -1 {
		return deltaLink[idx+len(prefix):]
	}
	return ""
}
