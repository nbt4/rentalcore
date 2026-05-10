package m365

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ExchangeAdminClient verwaltet GAL-Kontakte via Exchange Online cmdlet REST API.
// Benötigt Exchange.ManageAsApp + "Mail Recipients"-Rolle auf dem Service Principal.
type ExchangeAdminClient struct {
	tenantID     string
	clientID     string
	clientSecret string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time

	httpClient *http.Client
}

// GALContact entspricht einem Exchange MailContact-Objekt.
type GALContact struct {
	Name                 string `json:"Name"`
	ExternalEmailAddress string `json:"ExternalEmailAddress"`
	FirstName            string `json:"FirstName,omitempty"`
	LastName             string `json:"LastName,omitempty"`
	Company              string `json:"Company,omitempty"`
	Phone                string `json:"Phone,omitempty"`
}

// cmdletRequest entspricht dem Exchange Online InvokeCommand-Format.
type cmdletRequest struct {
	CmdletInput cmdletInput `json:"CmdletInput"`
}

type cmdletInput struct {
	CmdletName string         `json:"CmdletName"`
	Parameters map[string]any `json:"Parameters"`
}

func NewExchangeAdminClient(tenantID, clientID, clientSecret string) *ExchangeAdminClient {
	return &ExchangeAdminClient{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ExchangeAdminClient) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"scope":         {"https://outlook.office365.com/.default"},
	}

	resp, err := c.httpClient.PostForm(
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.tenantID),
		data,
	)
	if err != nil {
		return "", fmt.Errorf("exchange token request: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("exchange token decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("exchange token: empty access_token")
	}

	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return c.token, nil
}

func (c *ExchangeAdminClient) invokeCommand(cmdletName string, params map[string]any) error {
	token, err := c.getToken()
	if err != nil {
		return err
	}

	reqBody := cmdletRequest{
		CmdletInput: cmdletInput{
			CmdletName: cmdletName,
			Parameters: params,
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("https://outlook.office365.com/adminapi/beta/%s/InvokeCommand", c.tenantID)
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%s: HTTP %d — %s", cmdletName, resp.StatusCode, string(body))
}

func (c *ExchangeAdminClient) invokeQuery(cmdletName string, params map[string]any) ([]map[string]any, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	reqBody := cmdletRequest{
		CmdletInput: cmdletInput{
			CmdletName: cmdletName,
			Parameters: params,
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("https://outlook.office365.com/adminapi/beta/%s/InvokeCommand", c.tenantID)
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s: HTTP %d — %s", cmdletName, resp.StatusCode, string(body))
	}

	var result struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result.Value, nil
}

// normEmail entfernt den "smtp:"-Prefix falls vorhanden.
func normEmail(email string) string {
	return strings.TrimPrefix(strings.TrimPrefix(email, "smtp:"), "SMTP:")
}

// CreateMailContact legt einen neuen GAL MailContact via New-MailContact an.
func (c *ExchangeAdminClient) CreateMailContact(contact GALContact) error {
	if contact.ExternalEmailAddress == "" {
		return fmt.Errorf("ExternalEmailAddress required")
	}
	params := map[string]any{
		"Name":                 contact.Name,
		"ExternalEmailAddress": normEmail(contact.ExternalEmailAddress),
	}
	if contact.FirstName != "" {
		params["FirstName"] = contact.FirstName
	}
	if contact.LastName != "" {
		params["LastName"] = contact.LastName
	}
	return c.invokeCommand("New-MailContact", params)
}

// UpdateMailContact aktualisiert einen GAL MailContact via Set-MailContact.
func (c *ExchangeAdminClient) UpdateMailContact(email string, contact GALContact) error {
	params := map[string]any{
		"Identity": normEmail(email),
	}
	if contact.Name != "" {
		params["DisplayName"] = contact.Name
	}
	if contact.FirstName != "" {
		params["FirstName"] = contact.FirstName
	}
	if contact.LastName != "" {
		params["LastName"] = contact.LastName
	}
	return c.invokeCommand("Set-MailContact", params)
}

// DeleteMailContact löscht einen GAL MailContact via Remove-MailContact.
func (c *ExchangeAdminClient) DeleteMailContact(email string) error {
	return c.invokeCommand("Remove-MailContact", map[string]any{
		"Identity": normEmail(email),
		"Confirm":  false,
	})
}

// ListMailContactEmails gibt alle E-Mail-Adressen bestehender GAL MailContacts zurück.
func (c *ExchangeAdminClient) ListMailContactEmails() (map[string]bool, error) {
	items, err := c.invokeQuery("Get-MailContact", map[string]any{
		"ResultSize": "Unlimited",
	})
	if err != nil {
		return nil, err
	}

	emails := make(map[string]bool)
	for _, item := range items {
		if raw, ok := item["ExternalEmailAddress"]; ok {
			if email, ok := raw.(string); ok {
				emails[normEmail(email)] = true
			}
		}
	}
	return emails, nil
}
