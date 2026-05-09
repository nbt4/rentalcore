package m365

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

// SyncService koordiniert den bidirektionalen Sync zwischen RentalCore und M365.
type SyncService struct {
	client       *GraphClient
	customerRepo *repository.CustomerRepository
	db           *sql.DB
	syncInterval time.Duration
}

// NewSyncService erstellt einen SyncService. db wird für sync_state-Zugriff benötigt.
func NewSyncService(client *GraphClient, customerRepo *repository.CustomerRepository, db *sql.DB, syncInterval time.Duration) *SyncService {
	return &SyncService{
		client:       client,
		customerRepo: customerRepo,
		db:           db,
		syncInterval: syncInterval,
	}
}

// Start startet den Delta-Poll-Loop als Goroutine. Blockiert nicht.
func (s *SyncService) Start(ctx context.Context) {
	if err := s.ensureSyncStateTable(); err != nil {
		log.Printf("M365 sync: could not ensure sync_state table: %v", err)
		return
	}
	go s.runDeltaLoop(ctx)
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

// PushCreate sendet einen neuen Kunden an M365 und speichert die erhaltene M365-ID.
func (s *SyncService) PushCreate(customer *models.Customer) error {
	contact := CustomerToContact(customer)
	m365ID, err := s.client.CreateContact(contact)
	if err != nil {
		return fmt.Errorf("PushCreate: %w", err)
	}
	return s.customerRepo.SetM365ID(customer.CustomerID, m365ID)
}

// PushUpdate aktualisiert einen bestehenden Kontakt in M365.
func (s *SyncService) PushUpdate(customer *models.Customer) error {
	if customer.M365ID == nil || *customer.M365ID == "" {
		return s.PushCreate(customer)
	}
	contact := CustomerToContact(customer)
	return s.client.UpdateContact(*customer.M365ID, contact)
}

// PushDelete löscht den Kontakt in M365.
func (s *SyncService) PushDelete(customer *models.Customer) {
	if customer.M365ID == nil || *customer.M365ID == "" {
		return
	}
	if err := s.client.DeleteContact(*customer.M365ID); err != nil {
		log.Printf("M365 sync: PushDelete for %s failed: %v", *customer.M365ID, err)
	}
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
