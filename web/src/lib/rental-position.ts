import type { RentalCatalogItem } from './api';

export function rentalPositionInput(item: RentalCatalogItem, quantity = 1) {
  if (item.customerPrice == null || item.rentalPrice == null ||
      !Number.isFinite(item.customerPrice) || item.customerPrice < 0 ||
      !Number.isFinite(item.rentalPrice) || item.rentalPrice < 0) {
    throw new Error('Rental catalog purchase and customer prices are required.');
  }
  return {
    position_type: 'rental' as const,
    rental_equipment_id: item.equipmentID,
    description: item.productName,
    quantity,
    unit: 'Stück',
    unit_price: item.customerPrice,
    follow_day_factor: 0,
  };
}
