package repository

import (
	"strings"

	"gorm.io/gorm"
)

// ProductSearchTerms normalizes a free-text query into terms which must all be
// present somewhere in the product context. This lets a query combine fields,
// for example "LD Systems Stinger" across brand and product name.
func ProductSearchTerms(value string) []string {
	return strings.Fields(strings.ToLower(strings.TrimSpace(value)))
}

// ApplyProductSearch searches the complete product context instead of only the
// title: master data, classification, brand/manufacturer, technical values and
// identifiers of the owned devices.
func ApplyProductSearch(query *gorm.DB, value string) *gorm.DB {
	terms := ProductSearchTerms(value)
	if len(terms) == 0 {
		return query
	}

	query = query.
		Joins("LEFT JOIN categories search_category ON search_category.categoryid = products.categoryid").
		Joins("LEFT JOIN subcategories search_subcategory ON search_subcategory.subcategoryid = products.subcategoryid").
		Joins("LEFT JOIN subbiercategories search_subbiercategory ON search_subbiercategory.subbiercategoryid = products.subbiercategoryid").
		Joins("LEFT JOIN brands search_brand ON search_brand.brandid = products.brandid").
		Joins("LEFT JOIN manufacturer search_manufacturer ON search_manufacturer.manufacturerid = products.manufacturerid")

	for _, term := range terms {
		pattern := "%" + term + "%"
		query = query.Where(`(
			LOWER(CONCAT_WS(' ',products.productid::text,products.name,products.description,
			 search_category.name,search_subcategory.name,search_subbiercategory.name,
			 search_brand.name,search_manufacturer.name,products.generic_barcode,
			 products.product_type,products.tracking_mode,products.weight::text,
			 products.height::text,products.width::text,products.depth::text,
			 products.powerconsumption::text)) LIKE ?
			OR EXISTS (SELECT 1 FROM devices search_device WHERE search_device.productid=products.productid
			 AND LOWER(CONCAT_WS(' ',search_device.deviceid,search_device.serialnumber,
			 search_device.barcode,search_device.qr_code,search_device.notes)) LIKE ?)
		)`, pattern, pattern)
	}
	return query
}

// ApplyDeviceSearch extends product context with the individual device fields.
// The caller must join products as `products` before applying the filter.
func ApplyDeviceSearch(query *gorm.DB, value string) *gorm.DB {
	terms := ProductSearchTerms(value)
	if len(terms) == 0 {
		return query
	}
	query = query.
		Joins("LEFT JOIN categories search_category ON search_category.categoryid = products.categoryid").
		Joins("LEFT JOIN subcategories search_subcategory ON search_subcategory.subcategoryid = products.subcategoryid").
		Joins("LEFT JOIN subbiercategories search_subbiercategory ON search_subbiercategory.subbiercategoryid = products.subbiercategoryid").
		Joins("LEFT JOIN brands search_brand ON search_brand.brandid = products.brandid").
		Joins("LEFT JOIN manufacturer search_manufacturer ON search_manufacturer.manufacturerid = products.manufacturerid")
	for _, term := range terms {
		query = query.Where(`LOWER(CONCAT_WS(' ',devices.deviceid,devices.serialnumber,devices.barcode,
			devices.qr_code,devices.notes,products.productid::text,products.name,products.description,
			products.generic_barcode,search_category.name,search_subcategory.name,
			search_subbiercategory.name,search_brand.name,search_manufacturer.name)) LIKE ?`, "%"+term+"%")
	}
	return query
}
