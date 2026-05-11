package m365

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

// SyncService koordiniert den bidirektionalen Sync zwischen RentalCore und M365.
type SyncService struct {
	client         *GraphClient
	exchangeClient *ExchangeAdminClient
	customerRepo   *repository.CustomerRepository
	db             *sql.DB
	syncInterval   time.Duration
}

// NewSyncService erstellt einen SyncService. db wird für sync_state-Zugriff benötigt.
func NewSyncService(client *GraphClient, exchangeClient *ExchangeAdminClient, customerRepo *repository.CustomerRepository, db *sql.DB, syncInterval time.Duration) *SyncService {
	return &SyncService{
		client:         client,
		exchangeClient: exchangeClient,
		customerRepo:   customerRepo,
		db:             db,
		syncInterval:   syncInterval,
	}
}

// Start startet den Delta-Poll-Loop als Goroutine. Blockiert nicht.
// Beim ersten Start wird außerdem ein einmaliger Bulk-Push aller Kunden
// ohne m365_id angestoßen, bevor der Delta-Loop beginnt.
func (s *SyncService) Start(ctx context.Context) {
	if err := s.ensureSyncStateTable(); err != nil {
		log.Printf("M365 sync: could not ensure sync_state table: %v", err)
		return
	}
	go func() {
		s.BulkPushUnsynced()
		s.BulkPushToGAL()
		s.runDeltaLoop(ctx)
	}()
	log.Printf("M365 sync: started (interval: %s)", s.syncInterval)
}

func (s *SyncService) runDeltaLoop(ctx context.Context) {
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	s.runOnce()

	for {
		select {
		case <-ctx.Done():
			log.Println("M365 sync: stopping delta loop")
			return
		case <-ticker.C:
			s.runOnce()
		}
	}
}

func (s *SyncService) runOnce() {
	deltaToken, _ := s.loadDeltaToken()
	contacts, newToken, err := s.client.GetDelta(deltaToken)
	if err != nil {
		log.Printf("M365 sync: delta fetch failed: %v", err)
		return
	}

	for _, contact := range contacts {
		if contact.Removed != nil {
			s.handleM365Deletion(contact.ID)
		} else {
			s.handleM365Change(contact)
		}
	}

	if newToken != "" {
		if err := s.saveDeltaToken(newToken); err != nil {
			log.Printf("M365 sync: failed to save delta token: %v", err)
		}
	}
}

func (s *SyncService) handleM365Change(contact M365Contact) {
	existing, err := s.customerRepo.GetByM365ID(contact.ID)
	if err != nil {
		newCustomer := ContactToCustomer(contact)
		m365ID := contact.ID
		newCustomer.M365ID = &m365ID
		if createErr := s.customerRepo.Create(&newCustomer); createErr != nil {
			log.Printf("M365 sync: create customer failed for contact %s: %v", contact.ID, createErr)
		}
		return
	}

	var m365Time time.Time
	if contact.LastModifiedDateTime != "" {
		m365Time, _ = time.Parse(time.RFC3339, contact.LastModifiedDateTime)
	}

	if !ShouldApplyM365Change(existing.UpdatedAt, m365Time) {
		return
	}

	updated := ContactToCustomer(contact)
	updated.CustomerID = existing.CustomerID
	updated.M365ID = &contact.ID
	updated.IsCustomer = existing.IsCustomer
	updated.IsSupplier = existing.IsSupplier
	updated.CustomerType = existing.CustomerType

	if err := s.customerRepo.Update(&updated); err != nil {
		log.Printf("M365 sync: update customer %d failed: %v", existing.CustomerID, err)
	}
}

func (s *SyncService) handleM365Deletion(contactID string) {
	existing, err := s.customerRepo.GetByM365ID(contactID)
	if err != nil {
		return
	}
	if err := s.customerRepo.Archive(existing.CustomerID); err != nil {
		log.Printf("M365 sync: archive customer %d failed: %v", existing.CustomerID, err)
	}
}

// BulkPushUnsynced schiebt alle Kunden ohne m365_id einmalig nach M365.
// Läuft beim Server-Start einmal durch; danach übernimmt der Delta-Loop + Push-Hooks.
func (s *SyncService) BulkPushUnsynced() {
	customers, err := s.customerRepo.GetUnsynced()
	if err != nil {
		log.Printf("M365 bulk-push: could not fetch unsynced customers: %v", err)
		return
	}
	if len(customers) == 0 {
		log.Println("M365 bulk-push: all customers already synced")
		return
	}
	log.Printf("M365 bulk-push: pushing %d unsynced customers to M365", len(customers))

	ok, failed := 0, 0
	for i := range customers {
		if err := s.PushCreate(&customers[i]); err != nil {
			log.Printf("M365 bulk-push: customer %d failed: %v", customers[i].CustomerID, err)
			failed++
		} else {
			ok++
		}
		if i > 0 && i%10 == 0 {
			log.Printf("M365 bulk-push: progress %d/%d", i+1, len(customers))
		}
		// ~4 req/s Graph-API-Limit einhalten
		time.Sleep(250 * time.Millisecond)
	}
	log.Printf("M365 bulk-push: done — %d pushed, %d failed", ok, failed)
}

// PushCreate sendet einen neuen Kunden an M365 (Shared Mailbox + GAL).
func (s *SyncService) PushCreate(customer *models.Customer) error {
	contact := CustomerToContact(customer)
	m365ID, err := s.client.CreateContact(contact)
	if err != nil {
		return fmt.Errorf("PushCreate: %w", err)
	}
	if err := s.customerRepo.SetM365ID(customer.CustomerID, m365ID); err != nil {
		return err
	}
	if galErr := s.GALPushCreate(customer); galErr != nil {
		log.Printf("M365 GAL: PushCreate for customer %d failed: %v", customer.CustomerID, galErr)
	}
	return nil
}

// PushUpdate aktualisiert einen bestehenden Kontakt in M365 (Shared Mailbox + GAL).
func (s *SyncService) PushUpdate(customer *models.Customer) error {
	if customer.M365ID == nil || *customer.M365ID == "" {
		return s.PushCreate(customer)
	}
	contact := CustomerToContact(customer)
	if err := s.client.UpdateContact(*customer.M365ID, contact); err != nil {
		return err
	}
	if galErr := s.GALPushUpdate(customer); galErr != nil {
		log.Printf("M365 GAL: PushUpdate for customer %d failed: %v", customer.CustomerID, galErr)
	}
	return nil
}

// PushDelete löscht den Kontakt in M365 und der GAL.
func (s *SyncService) PushDelete(customer *models.Customer) {
	if customer.M365ID != nil && *customer.M365ID != "" {
		if err := s.client.DeleteContact(*customer.M365ID); err != nil {
			log.Printf("M365 sync: PushDelete mailbox contact %s failed: %v", *customer.M365ID, err)
		}
	}
	s.GALPushDelete(customer)
}

// GALPushCreate legt einen neuen Kunden in der GAL an.
func (s *SyncService) GALPushCreate(customer *models.Customer) error {
	if s.exchangeClient == nil {
		return nil
	}
	if customer.Email == nil || *customer.Email == "" {
		return nil
	}
	contact := CustomerToGALContact(customer)
	if err := s.exchangeClient.CreateMailContact(contact); err != nil {
		return fmt.Errorf("GALPushCreate: %w", err)
	}
	return s.customerRepo.SetGALContactID(customer.CustomerID, *customer.Email)
}

// GALPushUpdate aktualisiert einen bestehenden GAL-Kontakt.
// Bei E-Mail-Änderung wird der alte gelöscht und ein neuer angelegt.
func (s *SyncService) GALPushUpdate(customer *models.Customer) error {
	if s.exchangeClient == nil {
		return nil
	}
	if customer.Email == nil || *customer.Email == "" {
		return nil
	}
	contact := CustomerToGALContact(customer)

	if customer.GALContactID == nil || *customer.GALContactID == "" {
		return s.GALPushCreate(customer)
	}

	oldEmail := *customer.GALContactID
	if oldEmail != *customer.Email {
		// E-Mail geändert — alten Kontakt löschen, neuen anlegen
		if err := s.exchangeClient.DeleteMailContact(oldEmail); err != nil {
			log.Printf("M365 GAL: delete old contact %s failed: %v", oldEmail, err)
		}
		if err := s.exchangeClient.CreateMailContact(contact); err != nil {
			return fmt.Errorf("GALPushUpdate (re-create): %w", err)
		}
		return s.customerRepo.SetGALContactID(customer.CustomerID, *customer.Email)
	}

	return s.exchangeClient.UpdateMailContact(*customer.GALContactID, contact)
}

// GALPushDelete löscht den GAL-Kontakt eines Kunden.
func (s *SyncService) GALPushDelete(customer *models.Customer) {
	if s.exchangeClient == nil {
		return
	}
	if customer.GALContactID == nil || *customer.GALContactID == "" {
		return
	}
	if err := s.exchangeClient.DeleteMailContact(*customer.GALContactID); err != nil {
		log.Printf("M365 GAL: PushDelete %s failed: %v", *customer.GALContactID, err)
	}
}

// BulkPushToGAL schiebt alle Kunden mit E-Mail aber ohne gal_contact_id einmalig in die GAL.
func (s *SyncService) BulkPushToGAL() {
	if s.exchangeClient == nil {
		return
	}
	customers, err := s.customerRepo.GetGALUnsynced()
	if err != nil {
		log.Printf("M365 GAL bulk-push: could not fetch unsynced customers: %v", err)
		return
	}

	// Bereits in der GAL vorhandene Kontakte ermitteln (Erst-Sync-Schutz)
	existing, err := s.exchangeClient.ListMailContactEmails()
	if err != nil {
		log.Printf("M365 GAL bulk-push: could not list existing GAL contacts: %v", err)
		existing = map[string]bool{}
	}

	var toCreate []models.Customer
	for _, c := range customers {
		if c.Email == nil || *c.Email == "" {
			continue
		}
		if existing[*c.Email] {
			// Bereits in GAL — nur ID in DB setzen
			if dbErr := s.customerRepo.SetGALContactID(c.CustomerID, *c.Email); dbErr != nil {
				log.Printf("M365 GAL bulk-push: set existing GAL ID for %d failed: %v", c.CustomerID, dbErr)
			}
			continue
		}
		toCreate = append(toCreate, c)
	}

	if len(toCreate) == 0 {
		log.Println("M365 GAL bulk-push: all customers already in GAL")
		return
	}

	log.Printf("M365 GAL bulk-push: creating %d contacts in GAL", len(toCreate))
	ok, failed := 0, 0
	for i := range toCreate {
		if err := s.GALPushCreate(&toCreate[i]); err != nil {
			// 409 = E-Mail gehört bereits einem Exchange-Postfach — als gesynct markieren
			if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "ProxyAddressExists") {
				_ = s.customerRepo.SetGALContactID(toCreate[i].CustomerID, *toCreate[i].Email)
				log.Printf("M365 GAL bulk-push: customer %d email already in Exchange (mailbox) — marked synced", toCreate[i].CustomerID)
				ok++
			} else {
				log.Printf("M365 GAL bulk-push: customer %d failed: %v", toCreate[i].CustomerID, err)
				failed++
			}
		} else {
			ok++
		}
		if i > 0 && i%10 == 0 {
			log.Printf("M365 GAL bulk-push: progress %d/%d", i+1, len(toCreate))
		}
		time.Sleep(250 * time.Millisecond)
	}
	log.Printf("M365 GAL bulk-push: done — %d created, %d failed", ok, failed)
}

// ShouldApplyM365Change gibt true zurück wenn m365Time neuer als rcTime ist.
// Exportiert für Tests.
func ShouldApplyM365Change(rcTime time.Time, m365Time time.Time) bool {
	return m365Time.After(rcTime)
}

func (s *SyncService) loadDeltaToken() (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM sync_state WHERE key = 'm365_delta_token'`).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SyncService) saveDeltaToken(token string) error {
	_, err := s.db.Exec(`
		INSERT INTO sync_state (key, value, updated_at)
		VALUES ('m365_delta_token', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()
	`, token)
	return err
}

func (s *SyncService) ensureSyncStateTable() error {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'sync_state'
	`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = s.db.Exec(`
		CREATE TABLE sync_state (
			key        VARCHAR(100) PRIMARY KEY,
			value      TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}
