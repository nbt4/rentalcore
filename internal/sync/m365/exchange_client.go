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

// ExchangeAdminClient verwaltet GAL-Kontakte (MailContact) via Exchange Online admin API.
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
	ExternalEmailAddress string `json:"ExternalEmailAddress"` // "smtp:email@domain.com"
	FirstName            string `json:"FirstName,omitempty"`
	LastName             string `json:"LastName,omitempty"`
	Company              string `json:"Company,omitempty"`
	Phone                string `json:"Phone,omitempty"`
}

type exchangeContactList struct {
	Value []struct {
		ExternalEmailAddress string `json:"ExternalEmailAddress"`
		Name                 string `json:"Name"`
	} `json:"value"`
	NextLink string `json:"@odata.nextLink"`
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
		return "", fmt.Errorf("exchange token: empty access_token in response")
	}

	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return c.token, nil
}

func (c *ExchangeAdminClient) baseURL() string {
	return fmt.Sprintf("https://outlook.office365.com/adminapi/beta/%s", c.tenantID)
}

func (c *ExchangeAdminClient) doRequest(method, reqURL string, body interface{}) (*http.Response, error) {
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
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// normEmail stellt sicher dass die Adresse ohne "smtp:"-Prefix vorliegt.
func normEmail(email string) string {
	return strings.TrimPrefix(email, "smtp:")
}

// CreateMailContact legt einen neuen GAL-Kontakt an.
func (c *ExchangeAdminClient) CreateMailContact(contact GALContact) error {
	if contact.ExternalEmailAddress == "" {
		return fmt.Errorf("ExternalEmailAddress required")
	}
	contact.ExternalEmailAddress = "smtp:" + normEmail(contact.ExternalEmailAddress)

	resp, err := c.doRequest("POST", c.baseURL()+"/MailContact", contact)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("create GAL contact: HTTP %d", resp.StatusCode)
	}
	return nil
}

// UpdateMailContact aktualisiert einen bestehenden GAL-Kontakt anhand seiner E-Mail.
func (c *ExchangeAdminClient) UpdateMailContact(email string, contact GALContact) error {
	reqURL := fmt.Sprintf("%s/MailContact('%s')", c.baseURL(), url.PathEscape(normEmail(email)))
	if contact.ExternalEmailAddress != "" {
		contact.ExternalEmailAddress = "smtp:" + normEmail(contact.ExternalEmailAddress)
	}

	resp, err := c.doRequest("PATCH", reqURL, contact)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("update GAL contact: HTTP %d", resp.StatusCode)
	}
	return nil
}

// DeleteMailContact löscht einen GAL-Kontakt anhand seiner E-Mail.
func (c *ExchangeAdminClient) DeleteMailContact(email string) error {
	reqURL := fmt.Sprintf("%s/MailContact('%s')", c.baseURL(), url.PathEscape(normEmail(email)))
	resp, err := c.doRequest("DELETE", reqURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete GAL contact: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ListMailContactEmails gibt alle E-Mail-Adressen bestehender GAL MailContacts zurück.
func (c *ExchangeAdminClient) ListMailContactEmails() (map[string]bool, error) {
	emails := make(map[string]bool)
	reqURL := c.baseURL() + "/MailContact?$select=ExternalEmailAddress,Name"

	for reqURL != "" {
		resp, err := c.doRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		var list exchangeContactList
		decErr := json.NewDecoder(resp.Body).Decode(&list)
		resp.Body.Close()
		if decErr != nil {
			return nil, decErr
		}

		for _, contact := range list.Value {
			emails[normEmail(contact.ExternalEmailAddress)] = true
		}
		reqURL = list.NextLink
	}
	return emails, nil
}
