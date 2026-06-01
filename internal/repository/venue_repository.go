package repository

import "go-barcode-webapp/internal/models"

type VenueRepository struct {
	db *Database
}

func NewVenueRepository(db *Database) *VenueRepository {
	return &VenueRepository{db: db}
}

func (r *VenueRepository) List() ([]models.Venue, error) {
	var venues []models.Venue
	err := r.db.Order("name ASC").Find(&venues).Error
	return venues, err
}

func (r *VenueRepository) GetByID(id uint) (*models.Venue, error) {
	var venue models.Venue
	err := r.db.First(&venue, id).Error
	if err != nil {
		return nil, err
	}
	return &venue, nil
}

func (r *VenueRepository) Create(v *models.Venue) error {
	return r.db.Create(v).Error
}

func (r *VenueRepository) Update(v *models.Venue) error {
	return r.db.Model(v).Where("id = ?", v.ID).Updates(map[string]interface{}{
		"name":         v.Name,
		"street":       v.Street,
		"house_number": v.HouseNumber,
		"zip":          v.ZIP,
		"city":         v.City,
		"contact_name": v.ContactName,
		"phone":        v.Phone,
		"email":        v.Email,
		"notes":        v.Notes,
	}).Error
}

func (r *VenueRepository) Delete(id uint) error {
	return r.db.Delete(&models.Venue{}, id).Error
}
