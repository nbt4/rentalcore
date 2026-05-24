import { useState, useEffect, useCallback } from 'react';
import { Package, Wrench, Plus, Trash2, Check, Cpu, Building2 } from 'lucide-react';
import { positionsApi, api } from '../lib/api';
import type { JobPosition, JobTotals, RentalCatalogItem } from '../lib/api';

interface Props {
  jobId: number;
  onChanged?: () => void;
}

export default function JobPositionsPanel({ jobId, onChanged }: Props) {
  const [positions, setPositions] = useState<JobPosition[]>([]);
  const [totals, setTotals] = useState<JobTotals | null>(null);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState<'product' | 'service' | 'rental' | null>(null);
  const [products, setProducts] = useState<{ productID: number; name: string; itemcostperday?: number }[]>([]);
  const [services, setServices] = useState<{ id: number; name: string; default_price?: number; unit?: string }[]>([]);
  const [rentalItems, setRentalItems] = useState<RentalCatalogItem[]>([]);
  const [multiplyByDays, setMultiplyByDays] = useState(true);
  const [pricesIncludeTax, setPricesIncludeTax] = useState(false);

  const loadData = useCallback(async () => {
    try {
      const [posRes, totRes] = await Promise.all([
        positionsApi.getByJob(jobId),
        positionsApi.getTotals(jobId),
      ]);
      setPositions(posRes.data.positions || []);
      setTotals(totRes.data);
      setMultiplyByDays(totRes.data.multiply_by_days ?? true);
      setPricesIncludeTax(totRes.data.prices_include_tax ?? false);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }, [jobId]);

  useEffect(() => { loadData(); }, [loadData]);

  useEffect(() => {
    api.get('/products', { params: { limit: 5000 } }).then(r => {
      const list = (r.data as any).products || r.data;
      if (Array.isArray(list)) setProducts(list);
    }).catch(() => {});
    api.get('/service-items').then(r => {
      const list = (r.data as any).service_items || (r.data as any).items || r.data;
      if (Array.isArray(list)) setServices(list);
    }).catch(() => {});
    positionsApi.getRentalCatalog().then(r => {
      setRentalItems(r.data.items || []);
    }).catch(() => {});
  }, []);

  const productPositions = positions.filter(p => p.position_type === 'product');
  const servicePositions = positions.filter(p => p.position_type === 'service');
  const rentalPositions = positions.filter(p => p.position_type === 'rental' || p.position_type === 'package');

  const handleAdd = async (type: 'product' | 'service' | 'rental', itemId: number) => {
    if (type === 'product') {
      const prod = products.find(p => p.productID === itemId);
      if (!prod) return;
      await positionsApi.create(jobId, {
        position_type: 'product',
        product_id: prod.productID,
        description: prod.name,
        quantity: 1,
        unit: 'Stück',
        unit_price: prod.itemcostperday || 0,
        follow_day_factor: 0.5,
      });
    } else if (type === 'service') {
      const svc = services.find(s => s.id === itemId);
      if (!svc) return;
      await positionsApi.create(jobId, {
        position_type: 'service',
        service_item_id: svc.id,
        description: svc.name,
        quantity: 1,
        unit: svc.unit || 'Pauschale',
        unit_price: svc.default_price || 0,
        follow_day_factor: 0,
      });
    } else {
      const item = rentalItems.find(r => r.equipmentID === itemId);
      if (!item) return;
      await positionsApi.create(jobId, {
        position_type: 'rental',
        rental_equipment_id: item.equipmentID,
        description: item.productName,
        quantity: 1,
        unit: 'Stück',
        unit_price: item.rentalPrice,
        follow_day_factor: 0,
      });
    }
    setAdding(null);
    loadData();
    onChanged?.();
  };

  const handleUpdate = async (pos: JobPosition, field: string, value: number | string) => {
    await positionsApi.update(jobId, pos.position_id, { [field]: value });
    loadData();
    onChanged?.();
  };

  const handleToggle = async (field: 'multiply_by_days' | 'prices_include_tax', value: boolean) => {
    if (field === 'multiply_by_days') setMultiplyByDays(value);
    else setPricesIncludeTax(value);
    await positionsApi.updatePriceSettings(jobId, { [field]: value });
    loadData();
    onChanged?.();
  };

  const handleDelete = async (posId: number) => {
    if (!confirm('Position löschen?')) return;
    await positionsApi.delete(jobId, posId);
    loadData();
    onChanged?.();
  };

  const fmt = (n: number) => n.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  if (loading) return <div className="flex justify-center py-10"><div className="w-6 h-6 border-2 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;

  return (
    <div className="space-y-6">
      {/* Products Section */}
      <div className="glass-dark rounded-xl border border-white/10 p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Package className="w-4 h-4 text-accent-red" />
            <h3 className="font-semibold text-white">Produkte ({productPositions.length})</h3>
          </div>
          <button
            onClick={() => setAdding(adding === 'product' ? null : 'product')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-red/10 text-accent-red hover:bg-accent-red/20 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Produkt
          </button>
        </div>

        {adding === 'product' && (
          <div className="mb-4 p-3 rounded-lg bg-white/[0.03] border border-white/10">
            <select
              className="w-full bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
              value=""
              onChange={e => { if (e.target.value) handleAdd('product', parseInt(e.target.value)); }}
            >
              <option value="">Produkt auswählen...</option>
              {products.map(p => (
                <option key={p.productID} value={p.productID}>{p.name} ({fmt(p.itemcostperday || 0)} €/Tag)</option>
              ))}
            </select>
          </div>
        )}

        {productPositions.length === 0 && !adding && (
          <p className="text-gray-500 text-sm py-2">Keine Produkte hinzugefügt.</p>
        )}

        <div className="space-y-2">
          {productPositions.map(pos => (
            <PositionRow key={pos.position_id} pos={pos} onUpdate={handleUpdate} onDelete={handleDelete} showFactor />
          ))}
        </div>
      </div>

      {/* Services Section */}
      <div className="glass-dark rounded-xl border border-white/10 p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Wrench className="w-4 h-4 text-accent-red" />
            <h3 className="font-semibold text-white">Dienstleistungen ({servicePositions.length})</h3>
          </div>
          <button
            onClick={() => setAdding(adding === 'service' ? null : 'service')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-red/10 text-accent-red hover:bg-accent-red/20 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Dienstleistung
          </button>
        </div>

        {adding === 'service' && (
          <div className="mb-4 p-3 rounded-lg bg-white/[0.03] border border-white/10">
            <select
              className="w-full bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
              value=""
              onChange={e => { if (e.target.value) handleAdd('service', parseInt(e.target.value)); }}
            >
              <option value="">Dienstleistung auswählen...</option>
              {services.map(s => (
                <option key={s.id} value={s.id}>{s.name} ({fmt(s.default_price || 0)} €)</option>
              ))}
            </select>
          </div>
        )}

        {servicePositions.length === 0 && !adding && (
          <p className="text-gray-500 text-sm py-2">Keine Dienstleistungen hinzugefügt.</p>
        )}

        <div className="space-y-2">
          {servicePositions.map(pos => (
            <PositionRow key={pos.position_id} pos={pos} onUpdate={handleUpdate} onDelete={handleDelete} showFactor={false} />
          ))}
        </div>
      </div>

      {/* Mietprodukte Section */}
      <div className="glass-dark rounded-xl border border-white/10 p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Building2 className="w-4 h-4 text-accent-red" />
            <h3 className="font-semibold text-white">Mietprodukte ({rentalPositions.length})</h3>
          </div>
          <button
            onClick={() => setAdding(adding === 'rental' ? null : 'rental')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-accent-red/10 text-accent-red hover:bg-accent-red/20 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Mietprodukt
          </button>
        </div>

        {adding === 'rental' && (
          <div className="mb-4 p-3 rounded-lg bg-white/[0.03] border border-white/10">
            <select
              className="w-full bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
              value=""
              onChange={e => { if (e.target.value) handleAdd('rental', parseInt(e.target.value)); }}
            >
              <option value="">Mietprodukt auswählen...</option>
              {rentalItems.map(r => (
                <option key={r.equipmentID} value={r.equipmentID}>
                  {r.productName} — {r.supplierName} ({fmt(r.rentalPrice)} €/Tag)
                </option>
              ))}
            </select>
          </div>
        )}

        {rentalPositions.length === 0 && adding !== 'rental' && (
          <p className="text-gray-500 text-sm py-2">Keine Mietprodukte hinzugefügt.</p>
        )}

        <div className="space-y-2">
          {rentalPositions.map(pos => (
            <PositionRow key={pos.position_id} pos={pos} onUpdate={handleUpdate} onDelete={handleDelete} showFactor={false} />
          ))}
        </div>
      </div>

      {/* Price Settings Toggles */}
      <div className="glass-dark rounded-xl border border-white/10 p-4">
        <div className="flex items-center gap-6 flex-wrap">
          <span className="text-xs text-gray-500 font-medium uppercase tracking-wide">Preiseinstellungen</span>
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <div
              onClick={() => handleToggle('multiply_by_days', !multiplyByDays)}
              className={`relative w-9 h-5 rounded-full transition-colors ${multiplyByDays ? 'bg-accent-red' : 'bg-white/10'}`}
            >
              <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${multiplyByDays ? 'translate-x-4' : 'translate-x-0'}`} />
            </div>
            <span className="text-sm text-gray-300">Preis × Veranstaltungstage</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <div
              onClick={() => handleToggle('prices_include_tax', !pricesIncludeTax)}
              className={`relative w-9 h-5 rounded-full transition-colors ${pricesIncludeTax ? 'bg-accent-red' : 'bg-white/10'}`}
            >
              <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${pricesIncludeTax ? 'translate-x-4' : 'translate-x-0'}`} />
            </div>
            <span className="text-sm text-gray-300">
              {pricesIncludeTax ? 'Preise inkl. MwSt (Brutto)' : 'Preise exkl. MwSt (Netto)'}
            </span>
          </label>
        </div>
      </div>

      {/* Totals */}
      {totals && (
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="space-y-2 max-w-xs ml-auto text-sm">
            <div className="flex justify-between text-gray-400">
              <span>Veranstaltungstage</span>
              <span className="text-white font-medium">{totals.event_days}</span>
            </div>
            <div className="flex justify-between text-gray-400">
              <span>Zwischensumme</span>
              <span className="text-white">{fmt(totals.subtotal)} €</span>
            </div>
            {totals.global_discount > 0 && (
              <div className="flex justify-between text-yellow-400">
                <span>Rabatt</span>
                <span>-{fmt(totals.global_discount)} €</span>
              </div>
            )}
            <div className="flex justify-between text-gray-300 border-t border-white/10 pt-2">
              <span>Netto</span>
              <span className="font-medium">{fmt(totals.netto)} €</span>
            </div>
            <div className="flex justify-between text-gray-400">
              <span>MwSt ({totals.tax_rate}%){pricesIncludeTax ? ' (enthalten)' : ''}</span>
              <span>{fmt(totals.tax)} €</span>
            </div>
            <div className="flex justify-between text-white font-bold text-base border-t border-white/10 pt-2">
              <span>Brutto</span>
              <span>{fmt(totals.brutto)} €</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ── Position Row ── */

function PositionRow({ pos, onUpdate, onDelete, showFactor }: {
  pos: JobPosition;
  onUpdate: (pos: JobPosition, field: string, value: number | string) => void;
  onDelete: (id: number) => void;
  showFactor: boolean;
}) {
  const [editing, setEditing] = useState<string | null>(null);
  const [editVal, setEditVal] = useState('');

  const startEdit = (field: string, value: number | string) => {
    setEditing(field);
    setEditVal(String(value));
  };

  const commitEdit = (field: string) => {
    const numVal = parseFloat(editVal);
    if (!isNaN(numVal)) {
      onUpdate(pos, field, numVal);
    }
    setEditing(null);
  };

  const scannedCount = pos.devices?.length || 0;
  const neededCount = Math.floor(pos.quantity);

  return (
    <div className="rounded-lg border border-white/5 overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3 bg-white/[0.02] hover:bg-white/[0.04] transition-colors">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-white truncate">{pos.description || pos.product?.name || pos.service_item?.name || '—'}</span>
            {pos.position_type === 'product' && (
              <span className={`text-[0.65rem] px-1.5 py-0.5 rounded-full font-medium ${
                scannedCount >= neededCount ? 'bg-green-500/10 text-green-400 border border-green-500/20' : 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20'
              }`}>
                {scannedCount}/{neededCount}
              </span>
            )}
          </div>
        </div>

        {/* Quantity */}
        <EditableCell value={pos.quantity} field="quantity" editing={editing} editVal={editVal} startEdit={startEdit} commitEdit={commitEdit} setEditVal={setEditVal} width="w-14" suffix="" />

        {/* Unit */}
        <span className="text-xs text-gray-500 w-12 text-center">{pos.unit}</span>

        {/* Unit Price */}
        <EditableCell value={pos.unit_price} field="unit_price" editing={editing} editVal={editVal} startEdit={startEdit} commitEdit={commitEdit} setEditVal={setEditVal} width="w-20" suffix="€" />

        {/* Factor */}
        {showFactor && (
          <EditableCell value={pos.follow_day_factor} field="follow_day_factor" editing={editing} editVal={editVal} startEdit={startEdit} commitEdit={commitEdit} setEditVal={setEditVal} width="w-12" suffix="×" />
        )}

        {/* Discount */}
        <EditableCell value={pos.discount_percent} field="discount_percent" editing={editing} editVal={editVal} startEdit={startEdit} commitEdit={commitEdit} setEditVal={setEditVal} width="w-14" suffix="%" />

        {/* Tax Rate */}
        <EditableCell value={pos.tax_rate} field="tax_rate" editing={editing} editVal={editVal} startEdit={startEdit} commitEdit={commitEdit} setEditVal={setEditVal} width="w-14" suffix="%" />

        {/* Line Total */}
        <span className="w-20 text-xs text-right font-medium text-white">
          {(pos.quantity * pos.unit_price * (1 - pos.discount_percent / 100)).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
        </span>

        {/* Delete */}
        <button onClick={() => onDelete(pos.position_id)} className="p-1.5 rounded hover:bg-red-500/20 text-gray-500 hover:text-red-400 transition-colors">
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Devices under product */}
      {pos.position_type === 'product' && pos.devices && pos.devices.length > 0 && (
        <div className="border-t border-white/5 px-4 py-2 bg-white/[0.01]">
          {pos.devices.map(d => (
            <div key={d.id} className="flex items-center gap-2 py-1 text-xs text-gray-400">
              <Cpu className="w-3 h-3 text-green-400" />
              <span className="font-mono text-green-400">{d.device_id}</span>
              <Check className="w-3 h-3 text-green-500" />
              <span className="text-gray-600">{new Date(d.scanned_at).toLocaleDateString('de-DE')}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Editable Cell ── */

function EditableCell({ value, field, editing, editVal, startEdit, commitEdit, setEditVal, width, suffix }: {
  value: number;
  field: string;
  editing: string | null;
  editVal: string;
  startEdit: (f: string, v: number) => void;
  commitEdit: (f: string) => void;
  setEditVal: (v: string) => void;
  width: string;
  suffix: string;
}) {
  if (editing === field) {
    return (
      <input
        type="number"
        step="any"
        className={`${width} bg-dark-300 border border-accent-red/50 rounded px-1.5 py-0.5 text-xs text-white text-right focus:outline-none`}
        value={editVal}
        onChange={e => setEditVal(e.target.value)}
        onBlur={() => commitEdit(field)}
        onKeyDown={e => { if (e.key === 'Enter') commitEdit(field); if (e.key === 'Escape') { setEditVal(''); commitEdit(field); } }}
        autoFocus
      />
    );
  }
  return (
    <button
      onClick={() => startEdit(field, value)}
      className={`${width} text-xs text-gray-300 text-right hover:text-white hover:bg-white/5 rounded px-1.5 py-0.5 transition-colors`}
    >
      {value}{suffix && <span className="text-gray-600 ml-0.5">{suffix}</span>}
    </button>
  );
}
