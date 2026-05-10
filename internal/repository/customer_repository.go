package repository

import (
	"fmt"
	"time"

	"go-barcode-webapp/internal/models"
)

type CustomerRepository struct {
	db *Database
}

func NewCustomerRepository(db *Database) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	fmt.Printf("🔧 DEBUG CustomerRepo.Create: Before DB operation, customer ID: %d\n", customer.CustomerID)
	result := r.db.Create(customer)
	fmt.Printf("🔧 DEBUG CustomerRepo.Create: After DB operation, customer ID: %d, Error: %v\n", customer.CustomerID, result.Error)
	fmt.Printf("🔧 DEBUG CustomerRepo.Create: Rows affected: %d\n", result.RowsAffected)
	return result.Error
}

func (r *CustomerRepository) GetByID(id uint) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.First(&customer, id).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *CustomerRepository) Update(customer *models.Customer) error {
	return r.db.Save(customer).Error
}

func (r *CustomerRepository) Delete(id uint) error {
	return r.db.Delete(&models.Customer{}, id).Error
}

func (r *CustomerRepository) List(params *models.FilterParams) ([]models.Customer, error) {
	var customers []models.Customer

	query := r.db.Model(&models.Customer{})
	query = query.Where("is_archived = false OR is_archived IS NULL")

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("companyname LIKE ? OR firstname LIKE ? OR lastname LIKE ? OR email LIKE ?", searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	query = query.Order("companyname ASC")

	err := query.Find(&customers).Error
	return customers, err
}

// GetByM365ID sucht einen Kunden anhand seiner M365-Kontakt-ID.
func (r *CustomerRepository) GetByM365ID(m365ID string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("m365_id = ?", m365ID).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// SetM365ID speichert die M365-Kontakt-ID für einen Kunden.
func (r *CustomerRepository) SetM365ID(customerID uint, m365ID string) error {
	return r.db.Model(&models.Customer{}).
		Where("customerid = ?", customerID).
		Update("m365_id", m365ID).Error
}

// GetUnsynced gibt alle nicht-archivierten Kunden ohne m365_id zurück.
func (r *CustomerRepository) GetUnsynced() ([]models.Customer, error) {
	var customers []models.Customer
	err := r.db.Where("(is_archived = false OR is_archived IS NULL) AND (m365_id IS NULL OR m365_id = '')").
		Find(&customers).Error
	return customers, err
}

// Archive markiert einen Kunden als archiviert (nicht löschend).
func (r *CustomerRepository) Archive(customerID uint) error {
	now := time.Now()
	return r.db.Model(&models.Customer{}).
		Where("customerid = ?", customerID).
		Updates(map[string]interface{}{
			"is_archived": true,
			"archived_at": now,
			"m365_id":     nil,
		}).Error
}

// ListByRole returns customers filtered by role ("customer", "supplier", or "" for all).
func (r *CustomerRepository) ListByRole(params *models.FilterParams, role string) ([]models.Customer, error) {
	var customers []models.Customer

	query := r.db.Model(&models.Customer{})
	query = query.Where("is_archived = false OR is_archived IS NULL")

	switch role {
	case "customer":
		query = query.Where("is_customer = true")
	case "supplier":
		query = query.Where("is_supplier = true")
	// empty string: no filter — return all
	}

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("companyname LIKE ? OR firstname LIKE ? OR lastname LIKE ? OR email LIKE ?", searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	query = query.Order("companyname ASC")

	err := query.Find(&customers).Error
	return customers, err
}