package m365

import (
	"fmt"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
)

// CustomerToContact wandelt einen RentalCore-Kunden in einen M365-Kontakt um.
func CustomerToContact(c *models.Customer) M365Contact {
	contact := M365Contact{}

	if c.FirstName != nil {
		contact.GivenName = *c.FirstName
	}
	if c.LastName != nil {
		contact.Surname = *c.LastName
	}
	if c.CompanyName != nil {
		contact.CompanyName = *c.CompanyName
	}
	if c.Email != nil && *c.Email != "" {
		contact.EmailAddresses = []EmailAddr{{Address: *c.Email}}
	}
	if c.PhoneNumber != nil && *c.PhoneNumber != "" {
		contact.BusinessPhones = []string{*c.PhoneNumber}
	}
	if c.Notes != nil {
		contact.PersonalNotes = *c.Notes
	}

	street := ""
	if c.Street != nil {
		street = *c.Street
	}
	if c.HouseNumber != nil && *c.HouseNumber != "" {
		street = strings.TrimSpace(fmt.Sprintf("%s %s", street, *c.HouseNumber))
	}

	contact.BusinessAddress = Address{Street: street}
	if c.ZIP != nil {
		contact.BusinessAddress.PostalCode = *c.ZIP
	}
	if c.City != nil {
		contact.BusinessAddress.City = *c.City
	}
	if c.Country != nil {
		contact.BusinessAddress.CountryOrRegion = *c.Country
	}

	return contact
}

// ContactToCustomer wandelt einen M365-Kontakt in einen RentalCore-Kunden um.
func ContactToCustomer(contact M365Contact) models.Customer {
	c := models.Customer{}

	if contact.GivenName != "" {
		c.FirstName = strPtr(contact.GivenName)
	}
	if contact.Surname != "" {
		c.LastName = strPtr(contact.Surname)
	}
	if contact.CompanyName != "" {
		c.CompanyName = strPtr(contact.CompanyName)
	}
	if len(contact.EmailAddresses) > 0 && contact.EmailAddresses[0].Address != "" {
		c.Email = strPtr(contact.EmailAddresses[0].Address)
	}
	if len(contact.BusinessPhones) > 0 && contact.BusinessPhones[0] != "" {
		c.PhoneNumber = strPtr(contact.BusinessPhones[0])
	}
	if contact.PersonalNotes != "" {
		c.Notes = strPtr(contact.PersonalNotes)
	}

	street, houseNumber := SplitStreetAndNumber(contact.BusinessAddress.Street)
	if street != "" {
		c.Street = strPtr(street)
	}
	if houseNumber != "" {
		c.HouseNumber = strPtr(houseNumber)
	}
	if contact.BusinessAddress.PostalCode != "" {
		c.ZIP = strPtr(contact.BusinessAddress.PostalCode)
	}
	if contact.BusinessAddress.City != "" {
		c.City = strPtr(contact.BusinessAddress.City)
	}
	if contact.BusinessAddress.CountryOrRegion != "" {
		c.Country = strPtr(contact.BusinessAddress.CountryOrRegion)
	}

	if contact.LastModifiedDateTime != "" {
		if t, err := time.Parse(time.RFC3339, contact.LastModifiedDateTime); err == nil {
			c.M365UpdatedAt = &t
		}
	}

	return c
}

// SplitStreetAndNumber trennt "Hauptstraße 42" in ("Hauptstraße", "42").
// Exportiert für Tests.
func SplitStreetAndNumber(s string) (street, houseNumber string) {
	if s == "" {
		return "", ""
	}
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return s, ""
	}
	last := parts[len(parts)-1]
	if len(last) > 0 && last[0] >= '0' && last[0] <= '9' {
		return strings.Join(parts[:len(parts)-1], " "), last
	}
	return s, ""
}

func strPtr(s string) *string { return &s }
