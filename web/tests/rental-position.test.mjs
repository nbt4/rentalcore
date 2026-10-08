import test from 'node:test';
import assert from 'node:assert/strict';
import { rentalPositionInput } from '../src/lib/rental-position.ts';

const item = { equipmentID: 17, productName: 'Synthetic subwoofer', supplierName: 'Synthetic supplier', rentalPrice: 37.50, customerPrice: 50, category: 'Audio' };

test('+ Mietprodukt builds a real rental line at the customer price', () => {
  const position = rentalPositionInput(item, 3);
  assert.equal(position.position_type, 'rental');
  assert.equal(position.rental_equipment_id, 17);
  assert.equal(position.quantity, 3);
  assert.equal(position.description, item.productName);
  assert.equal(position.unit_price, 50);
  assert.notEqual(position.unit_price, item.rentalPrice);
  assert.equal(position.follow_day_factor, 0);
});

test('missing prices block instead of inventing a price; explicit zero remains valid', () => {
  for (const field of ['rentalPrice', 'customerPrice']) {
    for (const value of [null, undefined, -1, NaN, Infinity]) {
      assert.throws(() => rentalPositionInput({ ...item, [field]: value }));
    }
  }
  assert.equal(rentalPositionInput({ ...item, customerPrice: 0 }).unit_price, 0);
});
