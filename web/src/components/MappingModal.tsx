import { useState, useEffect, useRef, useCallback } from 'react';
import { X, CheckCircle, AlertCircle, Search, ChevronRight, ArrowLeft, Package, Box, Building2, UserRound, Pencil } from 'lucide-react';

// ── Types ──────────────────────────────────────────────────────────────────

interface NullInt64 { Int64: number; Valid: boolean; }
interface NullFloat64 { Float64: number; Valid: boolean; }

interface ExtractionItem {
  item_id: number;
  raw_product_text: string;
  quantity: NullInt64 | number | null;
  unit_price: NullFloat64 | number | null;
  line_total: NullFloat64 | number | null;
  mapped_product_id: NullInt64 | number | null;
  mapped_package_id: NullInt64 | number | null;
  mapped_rental_equipment_id: NullInt64 | number | null;
  mapping_status: 'pending' | 'auto_mapped' | 'user_confirmed' | 'user_rejected' | 'needs_creation';
  mapping_confidence: NullFloat64 | number | null;
  mapped_name?: string;
}

interface PreviewItem {
  item_id: number;
  name: string;
  raw_text: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  target_type: 'product' | 'package';
  target_id: number;
}

interface SearchResult {
  id: number;
  name: string;
  type: 'product' | 'package' | 'rental';
  sub: string;
}

export interface ExtractionMeta {
  customer_id?: number;
  customer_name?: string;
  start_date?: string;
  end_date?: string;
  document_date?: string;
  job_id?: number;
}

export interface MappedItem {
  product_id: number;
  name: string;
  quantity: number;
}

export interface MappingModalProps {
  uploadId: number;
  onComplete: (items: MappedItem[], meta: ExtractionMeta) => void;
  onClose: () => void;
}

function getNullInt(v: NullInt64 | number | null | undefined): number {
  if (v == null) return 0;
  if (typeof v === 'number') return v;
  return v.Valid ? v.Int64 : 0;
}

function getNullFloat(v: NullFloat64 | number | null | undefined): number {
  if (v == null) return 0;
  if (typeof v === 'number') return v;
  return v.Valid ? v.Float64 : 0;
}

function isMapped(item: ExtractionItem): boolean {
  return (item.mapping_status === 'auto_mapped' || item.mapping_status === 'user_confirmed')
    && (getNullInt(item.mapped_product_id) > 0 || getNullInt(item.mapped_package_id) > 0 || getNullInt(item.mapped_rental_equipment_id) > 0);
}

// ── FullCreateModal ─────────────────────────────────────────────────────────

type CreateTab = 'product' | 'package' | 'rental';

interface FullCreateModalProps {
  prefill: string;
  defaultTab?: CreateTab;
  onCreated: (result: SearchResult) => void;
  onClose: () => void;
}

function FullCreateModal({ prefill, defaultTab = 'product', onCreated, onClose }: FullCreateModalProps) {
  const [tab, setTab] = useState<CreateTab>(defaultTab);
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');

  // Produkt fields
  const [prodName, setProdName] = useState(prefill);
  const [prodDesc, setProdDesc] = useState('');
  const [prodPrice, setProdPrice] = useState('');
  const [prodDevices, setProdDevices] = useState('');

  // Paket fields
  const [pkgName, setPkgName] = useState(prefill);
  const [pkgCode, setPkgCode] = useState('');
  const [pkgDesc, setPkgDesc] = useState('');

  // Mietprodukt fields
  const [rentName, setRentName] = useState(prefill);
  const [supplierQuery, setSupplierQuery] = useState('');
  const [supplierResults, setSupplierResults] = useState<{ id: number; name: string }[]>([]);
  const [selectedSupplier, setSelectedSupplier] = useState<{ id: number; name: string } | null>(null);
  const [rentPrice, setRentPrice] = useState('');
  const [rentCustomer, setRentCustomer] = useState('');
  const [rentCategory, setRentCategory] = useState('');
  const [rentDesc, setRentDesc] = useState('');
  const [rentNotes, setRentNotes] = useState('');

  const searchSuppliers = async (q: string) => {
    if (q.trim().length < 2) { setSupplierResults([]); return; }
    const res = await fetch(`/api/pdf/customers/search?q=${encodeURIComponent(q)}&role=supplier`, { credentials: 'include' });
    if (!res.ok) return;
    const d = await res.json();
    setSupplierResults((d.customers || []).map((c: Record<string, unknown>) => ({
      id: c.customerid as number,
      name: (c.displayName || c.companyname || `${c.firstname || ''} ${c.lastname || ''}`.trim()) as string,
    })));
  };

  const submit = async () => {
    setSaving(true); setErr('');
    try {
      if (tab === 'product') {
        if (!prodName.trim()) { setErr('Name ist erforderlich.'); return; }
        const res = await fetch('/api/pdf/product-quick-create', {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
          body: JSON.stringify({
            name: prodName.trim(),
            description: prodDesc.trim() || undefined,
            item_cost_per_day: parseFloat(prodPrice) || 0,
            device_count: parseInt(prodDevices, 10) || 0,
          }),
        });
        const d = await res.json();
        if (!res.ok) throw new Error(d.error || 'Fehler');
        onCreated({ id: d.product_id, name: d.name, type: 'product',
          sub: `Produkt${d.devices_created ? ` · ${d.devices_created} Geräte` : ''}` });

      } else if (tab === 'package') {
        if (!pkgName.trim()) { setErr('Name ist erforderlich.'); return; }
        const res = await fetch('/api/pdf/package-quick-create', {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
          body: JSON.stringify({ name: pkgName.trim(), description: pkgDesc.trim() || undefined, code: pkgCode.trim() || undefined }),
        });
        const d = await res.json();
        if (!res.ok) throw new Error(d.error || 'Fehler');
        onCreated({ id: d.package_id, name: d.name, type: 'package', sub: 'Paket' });

      } else {
        if (!rentName.trim()) { setErr('Name ist erforderlich.'); return; }
        if (!selectedSupplier) { setErr('Lieferant ist erforderlich.'); return; }
        const res = await fetch('/api/pdf/rental-equipment-quick-create', {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
          body: JSON.stringify({
            name: rentName.trim(),
            supplier: selectedSupplier.name,
            supplier_id: selectedSupplier.id,
            rental_price: parseFloat(rentPrice) || 0,
            customer_price: parseFloat(rentCustomer) || 0,
            category: rentCategory.trim() || undefined,
            description: rentDesc.trim() || undefined,
            notes: rentNotes.trim() || undefined,
          }),
        });
        const d = await res.json();
        if (!res.ok) throw new Error(d.error || 'Fehler');
        onCreated({ id: d.rental_equipment_id, name: d.name, type: 'rental', sub: 'Mietprodukt' });
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Fehler');
    } finally {
      setSaving(false);
    }
  };

  const tabs: { id: CreateTab; label: string; icon: React.ReactNode }[] = [
    { id: 'product', label: 'Produkt', icon: <Box className="w-3.5 h-3.5" /> },
    { id: 'package', label: 'Paket', icon: <Package className="w-3.5 h-3.5" /> },
    { id: 'rental', label: 'Mietprodukt', icon: <Building2 className="w-3.5 h-3.5" /> },
  ];

  const inputCls = "rc-input rc-input-sm w-full";
  const labelCls = "block text-xs font-medium mb-1";

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center p-4"
      style={{ background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)' }}>
      <div className="rc-card w-full max-w-lg flex flex-col"
        style={{ borderRadius: '16px', border: '1px solid var(--rc-border)', boxShadow: '0 24px 64px rgba(0,0,0,0.6)', maxHeight: '90vh' }}>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4" style={{ borderBottom: '1px solid var(--rc-border)' }}>
          <h3 style={{ fontSize: '15px', fontWeight: 600, margin: 0, color: 'var(--rc-text-primary)' }}>Neu anlegen</h3>
          <button type="button" onClick={onClose} className="rc-btn rc-btn-secondary rc-btn-sm" style={{ padding: '4px 8px' }}>
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex px-5 pt-4 gap-1">
          {tabs.map(t => (
            <button key={t.id} type="button" onClick={() => { setTab(t.id); setErr(''); }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
              style={{
                background: tab === t.id ? 'var(--rc-primary)' : 'var(--rc-bg-secondary)',
                color: tab === t.id ? '#fff' : 'var(--rc-text-secondary)',
                border: `1px solid ${tab === t.id ? 'var(--rc-primary)' : 'var(--rc-border)'}`,
              }}>
              {t.icon}{t.label}
            </button>
          ))}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-3">

          {tab === 'product' && (
            <>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Name <span style={{ color: 'var(--rc-danger)' }}>*</span></label>
                <input autoFocus value={prodName} onChange={e => setProdName(e.target.value)} placeholder="z.B. Shure SM58" className={inputCls} />
              </div>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Beschreibung</label>
                <textarea value={prodDesc} onChange={e => setProdDesc(e.target.value)} rows={2} placeholder="Optionale Beschreibung…" className={inputCls} style={{ resize: 'none' }} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Preis/Tag (€)</label>
                  <input type="number" min="0" step="0.01" value={prodPrice} onChange={e => setProdPrice(e.target.value)} placeholder="0.00" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Geräte anlegen</label>
                  <input type="number" min="0" max="500" value={prodDevices} onChange={e => setProdDevices(e.target.value)} placeholder="0" className={inputCls} />
                </div>
              </div>
            </>
          )}

          {tab === 'package' && (
            <>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Name <span style={{ color: 'var(--rc-danger)' }}>*</span></label>
                <input autoFocus value={pkgName} onChange={e => setPkgName(e.target.value)} placeholder="z.B. PA-Set Klein" className={inputCls} />
              </div>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Kürzel / Code</label>
                <input value={pkgCode} onChange={e => setPkgCode(e.target.value)} placeholder="z.B. PA-S" className={inputCls} />
              </div>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Beschreibung</label>
                <textarea value={pkgDesc} onChange={e => setPkgDesc(e.target.value)} rows={2} placeholder="Optionale Beschreibung…" className={inputCls} style={{ resize: 'none' }} />
              </div>
            </>
          )}

          {tab === 'rental' && (
            <>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Produktname <span style={{ color: 'var(--rc-danger)' }}>*</span></label>
                <input autoFocus value={rentName} onChange={e => setRentName(e.target.value)} placeholder="z.B. Bühnenpodest 2×1m" className={inputCls} />
              </div>
              <div className="relative">
                <label className="block text-xs mb-1" style={{ color: 'var(--rc-text-secondary)' }}>
                  Lieferant *
                </label>
                <input
                  value={selectedSupplier ? selectedSupplier.name : supplierQuery}
                  onChange={e => {
                    setSelectedSupplier(null);
                    setSupplierQuery(e.target.value);
                    searchSuppliers(e.target.value);
                  }}
                  placeholder="Lieferant suchen…"
                  className="rc-input rc-input-sm w-full"
                />
                {supplierResults.length > 0 && !selectedSupplier && (
                  <div className="absolute top-full left-0 right-0 mt-1 rounded-lg overflow-hidden z-50"
                    style={{ background: 'var(--rc-bg-card)', border: '1px solid var(--rc-border)', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}>
                    {supplierResults.map(s => (
                      <button key={s.id} type="button"
                        onClick={() => { setSelectedSupplier(s); setSupplierResults([]); setSupplierQuery(''); }}
                        className="w-full flex items-center px-3 py-2 text-sm text-left"
                        style={{ borderBottom: '1px solid var(--rc-border)', color: 'var(--rc-text-primary)' }}
                        onMouseEnter={e => (e.currentTarget.style.background = 'var(--rc-bg-secondary)')}
                        onMouseLeave={e => (e.currentTarget.style.background = '')}>
                        {s.name}
                      </button>
                    ))}
                  </div>
                )}
                {selectedSupplier && (
                  <button type="button" onClick={() => { setSelectedSupplier(null); setSupplierQuery(''); }}
                    className="absolute right-2 top-7 text-xs"
                    style={{ color: 'var(--rc-text-secondary)' }}>
                    ✕
                  </button>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Mietpreis intern (€)</label>
                  <input type="number" min="0" step="0.01" value={rentPrice} onChange={e => setRentPrice(e.target.value)} placeholder="0.00" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Kundenpreis (€)</label>
                  <input type="number" min="0" step="0.01" value={rentCustomer} onChange={e => setRentCustomer(e.target.value)} placeholder="0.00" className={inputCls} />
                </div>
              </div>
              {parseFloat(rentPrice) > 0 && parseFloat(rentCustomer) > 0 && (
                <div className="rounded-lg px-3 py-2 text-center text-xs" style={{ background: 'var(--rc-bg-secondary)', color: 'var(--rc-text-secondary)' }}>
                  Marge: <span style={{ color: parseFloat(rentCustomer) >= parseFloat(rentPrice) ? 'var(--rc-success)' : 'var(--rc-danger)', fontWeight: 600 }}>
                    {((parseFloat(rentCustomer) - parseFloat(rentPrice)) / parseFloat(rentPrice) * 100).toFixed(1)}%
                  </span>
                  {' '}· +{(parseFloat(rentCustomer) - parseFloat(rentPrice)).toFixed(2)} €
                </div>
              )}
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Kategorie</label>
                <input value={rentCategory} onChange={e => setRentCategory(e.target.value)} placeholder="z.B. Bühne, Audio, Licht" className={inputCls} />
              </div>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Beschreibung</label>
                <textarea value={rentDesc} onChange={e => setRentDesc(e.target.value)} rows={2} placeholder="Optionale Beschreibung…" className={inputCls} style={{ resize: 'none' }} />
              </div>
              <div>
                <label className={labelCls} style={{ color: 'var(--rc-text-secondary)' }}>Interne Notizen</label>
                <textarea value={rentNotes} onChange={e => setRentNotes(e.target.value)} rows={2} placeholder="Lieferzeiten, Kontaktperson…" className={inputCls} style={{ resize: 'none' }} />
              </div>
            </>
          )}

          {err && <p className="text-xs" style={{ color: 'var(--rc-danger)' }}>{err}</p>}
        </div>

        {/* Footer */}
        <div className="flex gap-2 px-5 py-4" style={{ borderTop: '1px solid var(--rc-border)' }}>
          <button type="button" onClick={submit} disabled={saving} className="rc-btn rc-btn-primary rc-btn-sm" style={{ flex: 1 }}>
            {saving ? '…' : 'Anlegen & zuweisen'}
          </button>
          <button type="button" onClick={onClose} className="rc-btn rc-btn-secondary rc-btn-sm">Abbrechen</button>
        </div>
      </div>
    </div>
  );
}

// ── CustomerPicker ───────────────────────────────────────────────────────────

interface CustomerResult {
  customerid: number;
  displayName: string;
}

interface CustomerPickerProps {
  extractionId: number;
  currentName?: string;
  currentId?: number;
  onChanged: (id: number, name: string) => void;
}

function CustomerPicker({ extractionId, currentName, currentId, onChanged }: CustomerPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<CustomerResult[]>([]);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => { if (open) { setQuery(''); setResults([]); setTimeout(() => inputRef.current?.focus(), 50); } }, [open]);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  const search = useCallback((q: string) => {
    if (q.trim().length < 2) { setResults([]); return; }
    setLoading(true);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(async () => {
      try {
        const res = await fetch(`/api/pdf/customers/search?q=${encodeURIComponent(q)}`, { credentials: 'include' });
        const d = res.ok ? await res.json() : {};
        setResults((d.customers || []).slice(0, 8));
      } finally { setLoading(false); }
    }, 250);
  }, []);

  const select = async (customer: CustomerResult) => {
    setOpen(false);
    await fetch(`/api/pdf/customer-map/${extractionId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ customer_id: customer.customerid, customer_text: customer.displayName }),
    });
    onChanged(customer.customerid, customer.displayName);
  };

  const hasCustomer = !!currentId;

  return (
    <div className="relative">
      <button type="button" onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1.5 text-xs transition-colors group"
        style={{ color: hasCustomer ? 'var(--rc-text-secondary)' : 'var(--rc-warning)' }}>
        <UserRound className="w-3.5 h-3.5 flex-shrink-0" style={{ color: hasCustomer ? 'var(--rc-text-secondary)' : 'var(--rc-warning)' }} />
        <span>Kunde:</span>
        {hasCustomer
          ? <span style={{ color: 'var(--rc-text-primary)', fontWeight: 500 }}>{currentName}</span>
          : <span style={{ color: 'var(--rc-warning)', fontWeight: 500 }}>Nicht ausgewählt — bitte wählen</span>
        }
        <Pencil className="w-3 h-3 opacity-0 group-hover:opacity-60 transition-opacity" />
      </button>

      {open && (
        <div className="absolute top-full left-0 mt-1 z-50 rounded-lg overflow-hidden w-72"
          style={{ background: 'var(--rc-bg-card)', border: '1px solid var(--rc-border)', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}>
          <div className="p-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5" style={{ color: 'var(--rc-text-secondary)' }} />
              <input
                ref={inputRef}
                value={query}
                onChange={e => { setQuery(e.target.value); search(e.target.value); }}
                placeholder="Kunde suchen…"
                className="rc-input rc-input-sm w-full"
                style={{ paddingLeft: '2rem' }}
              />
            </div>
          </div>
          {loading && <div className="px-3 pb-2 text-xs" style={{ color: 'var(--rc-text-secondary)' }}>Suche…</div>}
          {results.map(r => (
            <button key={r.customerid} type="button" onClick={() => select(r)}
              className="w-full text-left px-3 py-2 text-sm transition-colors"
              style={{ borderTop: '1px solid var(--rc-border)' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--rc-bg-secondary)')}
              onMouseLeave={e => (e.currentTarget.style.background = '')}>
              <span style={{ color: 'var(--rc-text-primary)' }}>{r.displayName}</span>
            </button>
          ))}
          <div className="p-2 flex justify-end" style={{ borderTop: results.length > 0 ? '1px solid var(--rc-border)' : undefined }}>
            <button type="button" onClick={() => setOpen(false)} className="rc-btn rc-btn-secondary rc-btn-sm text-xs">Schließen</button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── InlineSearch ────────────────────────────────────────────────────────────

interface InlineSearchProps {
  initialQuery: string;
  onSelect: (r: SearchResult) => void;
  onCreateNew: (type: CreateTab, query: string) => void;
}

function InlineSearch({ initialQuery, onSelect, onCreateNew }: InlineSearchProps) {
  const [query, setQuery] = useState(initialQuery);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { inputRef.current?.focus(); }, []);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  const search = useCallback((q: string) => {
    if (q.trim().length < 2) { setResults([]); setSearched(false); return; }
    setLoading(true);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(async () => {
      try {
        const [pRes, pkgRes, reRes] = await Promise.all([
          fetch(`/api/pdf/products/search?q=${encodeURIComponent(q)}&limit=5`, { credentials: 'include' }),
          fetch(`/api/pdf/packages/search?q=${encodeURIComponent(q)}&limit=3`, { credentials: 'include' }),
          fetch(`/api/pdf/rental-equipment/search?q=${encodeURIComponent(q)}`, { credentials: 'include' }),
        ]);
        const pd = pRes.ok ? await pRes.json() : {};
        const pkd = pkgRes.ok ? await pkgRes.json() : {};
        const red = reRes.ok ? await reRes.json() : {};
        const products: SearchResult[] = (pd.products || []).slice(0, 5).map((p: Record<string, unknown>) => ({
          id: (p.productID || p.ProductID) as number,
          name: (p.name || p.Name) as string,
          type: 'product' as const,
          sub: 'Produkt',
        }));
        const packages: SearchResult[] = (pkd.packages || []).slice(0, 3).map((p: Record<string, unknown>) => ({
          id: (p.package_id || p.PackageID) as number,
          name: (p.name || p.Name) as string,
          type: 'package' as const,
          sub: `Paket${p.package_code ? ' · ' + p.package_code : ''}`,
        }));
        const rentals: SearchResult[] = (red.rental_equipment || []).slice(0, 3).map((r: Record<string, unknown>) => ({
          id: r.id as number,
          name: r.name as string,
          type: 'rental' as const,
          sub: `Mietprodukt${r.supplier ? ' · ' + r.supplier : ''}`,
        }));
        setResults([...products, ...packages, ...rentals]);
      } finally { setLoading(false); setSearched(true); }
    }, 300);
  }, []);

  const noResults = searched && !loading && results.length === 0 && query.trim().length >= 2;

  return (
    <div className="relative">
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5" style={{ color: 'var(--rc-text-secondary)' }} />
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => { setQuery(e.target.value); search(e.target.value); }}
          placeholder="Produkt, Paket oder Mietprodukt suchen…"
          className="rc-input rc-input-sm w-full"
          style={{ paddingLeft: '2rem' }}
        />
      </div>
      {(results.length > 0 || loading || noResults) && (
        <div className="absolute top-full left-0 right-0 mt-1 rounded-lg overflow-hidden z-50"
          style={{ background: 'var(--rc-bg-card)', border: '1px solid var(--rc-border)', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}>
          {loading && <div className="px-3 py-2 text-xs" style={{ color: 'var(--rc-text-secondary)' }}>Suche…</div>}
          {results.map((r) => (
            <button key={`${r.type}-${r.id}`} type="button" onClick={() => onSelect(r)}
              className="w-full flex items-center justify-between px-3 py-2 text-sm text-left transition-colors"
              style={{ borderBottom: '1px solid var(--rc-border)' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--rc-bg-secondary)')}
              onMouseLeave={e => (e.currentTarget.style.background = '')}>
              <span style={{ color: 'var(--rc-text-primary)' }}>{r.name}</span>
              <span className="text-xs ml-2" style={{ color: 'var(--rc-text-secondary)' }}>{r.sub}</span>
            </button>
          ))}
          {noResults && (
            <div>
              <div className="px-3 py-2 text-xs" style={{ color: 'var(--rc-text-secondary)' }}>Keine Treffer für „{query}"</div>
              <div className="flex gap-1 px-2 pb-2 flex-wrap">
                <button type="button" onClick={() => onCreateNew('product', query)}
                  className="rc-btn rc-btn-sm rc-btn-outline flex items-center gap-1" style={{ flex: '1 1 auto', fontSize: '11px' }}>
                  <Box className="w-3 h-3" /> Produkt
                </button>
                <button type="button" onClick={() => onCreateNew('package', query)}
                  className="rc-btn rc-btn-sm rc-btn-outline flex items-center gap-1" style={{ flex: '1 1 auto', fontSize: '11px' }}>
                  <Package className="w-3 h-3" /> Paket
                </button>
                <button type="button" onClick={() => onCreateNew('rental', query)}
                  className="rc-btn rc-btn-sm rc-btn-outline flex items-center gap-1" style={{ flex: '1 1 auto', fontSize: '11px' }}>
                  <Building2 className="w-3 h-3" /> Mietprodukt
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── MappingModal ────────────────────────────────────────────────────────────

export default function MappingModal({ uploadId, onComplete, onClose }: MappingModalProps) {
  const [phase, setPhase] = useState<'loading' | 'mapping' | 'preview' | 'error'>('loading');
  const [extractionId, setExtractionId] = useState<number | null>(null);
  const [items, setItems] = useState<ExtractionItem[]>([]);
  const [meta, setMeta] = useState<ExtractionMeta>({});
  const [activeSearch, setActiveSearch] = useState<number | null>(null);
  const [activeCreate, setActiveCreate] = useState<{ itemId: number; type: CreateTab; prefill: string } | null>(null);
  const [previewItems, setPreviewItems] = useState<PreviewItem[]>([]);
  const [totalAmount, setTotalAmount] = useState(0);
  const [errorMsg, setErrorMsg] = useState('');
  const [savingItem, setSavingItem] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        let extraction = null;
        for (let i = 0; i < 30; i++) {
          await new Promise((r) => setTimeout(r, 500));
          const res = await fetch(`/api/pdf/extraction/${uploadId}`, { credentials: 'include' });
          if (res.ok) { const d = await res.json(); if (d.extraction_id) { extraction = d; break; } }
        }
        if (!extraction) throw new Error('OCR-Timeout — bitte erneut versuchen');
        if (cancelled) return;
        setExtractionId(extraction.extraction_id);
        setMeta({
          customer_id: extraction.customer_id ?? undefined,
          customer_name: extraction.customer_name || undefined,
          start_date: extraction.start_date || undefined,
          end_date: extraction.end_date || undefined,
          document_date: extraction.document_date || undefined,
        });
        await fetch(`/api/pdf/auto-map/${extraction.extraction_id}`, { method: 'POST', credentials: 'include' });
        const res2 = await fetch(`/api/pdf/extraction/${uploadId}`, { credentials: 'include' });
        if (!res2.ok) throw new Error('Fehler beim Laden der Items');
        const final = await res2.json();
        if (cancelled) return;
        setItems(final.items || []);
        setPhase('mapping');
      } catch (e) {
        if (!cancelled) { setErrorMsg(e instanceof Error ? e.message : 'Fehler'); setPhase('error'); }
      }
    };
    load();
    return () => { cancelled = true; };
  }, [uploadId]);

  const mappedCount = items.filter(isMapped).length;
  const totalCount = items.length;
  const allMapped = mappedCount === totalCount && totalCount > 0;
  const hasCustomer = !!meta.customer_id;

  const handleSelect = async (item: ExtractionItem, result: SearchResult) => {
    setSavingItem(item.item_id);
    setActiveSearch(null);
    setActiveCreate(null);
    try {
      const body = result.type === 'package'
        ? { package_id: result.id, status: 'user_confirmed' }
        : result.type === 'rental'
        ? { rental_equipment_id: result.id, status: 'user_confirmed' }
        : { product_id: result.id, status: 'user_confirmed' };
      const res = await fetch(`/api/pdf/items/${item.item_id}/mapping`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) return;
      setItems((prev) => prev.map((it) => it.item_id !== item.item_id ? it : {
        ...it,
        mapped_product_id: result.type === 'product' ? result.id : null,
        mapped_package_id: result.type === 'package' ? result.id : null,
        mapped_rental_equipment_id: result.type === 'rental' ? result.id : null,
        mapping_status: 'user_confirmed',
        mapping_confidence: 100,
        mapped_name: result.name,
      }));
    } finally { setSavingItem(null); }
  };

  const handleCreateNew = (itemId: number, type: CreateTab, prefill: string) => {
    setActiveSearch(null);
    setActiveCreate({ itemId, type, prefill });
  };

  const handleProceedToPreview = async () => {
    if (!extractionId) return;
    try {
      const res = await fetch(`/api/pdf/extractions/${extractionId}/preview`, { credentials: 'include' });
      if (!res.ok) throw new Error('Vorschau konnte nicht geladen werden');
      const data = await res.json();
      setPreviewItems(data.items || []);
      setTotalAmount(data.total_amount || 0);
      setPhase('preview');
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : 'Fehler beim Laden der Vorschau');
      setPhase('error');
    }
  };

  const handleConfirm = async () => {
    if (!extractionId) return;
    try {
      const res = await fetch(`/api/pdf/extractions/${extractionId}/finalize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({}),
      });
      if (!res.ok) throw new Error('Finalisierung fehlgeschlagen');
      const data = await res.json();
      const mapped: MappedItem[] = previewItems
        .filter((pi) => pi.target_type === 'product')
        .map((pi) => ({ product_id: pi.target_id, name: pi.name, quantity: pi.quantity }));
      onComplete(mapped, { ...meta, job_id: data.job_id });
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : 'Fehler beim Speichern');
      setPhase('error');
    }
  };

  // confidence badge color
  const confColor = (conf: number) =>
    conf >= 80 ? 'var(--rc-success)' : conf >= 60 ? 'var(--rc-warning)' : 'var(--rc-danger)';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4"
      style={{ background: 'rgba(0,0,0,0.75)', backdropFilter: 'blur(4px)' }}>
      <div className="rc-card w-full max-w-2xl max-h-[90vh] flex flex-col"
        style={{ borderRadius: '16px', border: '1px solid var(--rc-border)', boxShadow: '0 20px 60px rgba(0,0,0,0.6)' }}>

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4" style={{ borderBottom: '1px solid var(--rc-border)' }}>
          <div className="flex items-center gap-3">
            {phase === 'preview' && (
              <button type="button" onClick={() => setPhase('mapping')}
                className="rc-btn rc-btn-secondary rc-btn-sm" style={{ padding: '4px 8px' }}>
                <ArrowLeft className="w-4 h-4" />
              </button>
            )}
            <h2 style={{ fontSize: '16px', margin: 0, fontWeight: 600, color: 'var(--rc-text-primary)' }}>
              {phase === 'preview' ? 'Vorschau — Items zum Job' : 'PDF Mapping'}
            </h2>
          </div>
          <button type="button" onClick={onClose}
            className="rc-btn rc-btn-secondary rc-btn-sm" style={{ padding: '4px 8px' }}>
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4">

          {phase === 'loading' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <div className="w-6 h-6 rounded-full border-2 border-t-transparent animate-spin"
                style={{ borderColor: 'var(--rc-primary)', borderTopColor: 'transparent' }} />
              <p className="text-sm" style={{ color: 'var(--rc-text-secondary)' }}>OCR läuft & Auto-Mapping…</p>
            </div>
          )}

          {phase === 'error' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <AlertCircle className="w-8 h-8" style={{ color: 'var(--rc-danger)' }} />
              <p className="text-sm" style={{ color: 'var(--rc-danger)' }}>{errorMsg}</p>
              <button type="button" onClick={onClose} className="rc-btn rc-btn-secondary rc-btn-sm">Schließen</button>
            </div>
          )}

          {phase === 'mapping' && (
            <>
              {/* Meta row: customer picker + dates — always visible */}
              <div className="rc-card mb-4 px-3 py-2 flex flex-wrap gap-4 items-center text-xs" style={{ background: 'var(--rc-bg-secondary)' }}>
                {extractionId && (
                  <CustomerPicker
                    extractionId={extractionId}
                    currentId={meta.customer_id}
                    currentName={meta.customer_name}
                    onChanged={(id, name) => setMeta(m => ({ ...m, customer_id: id, customer_name: name }))}
                  />
                )}
                {meta.start_date && (
                  <span style={{ color: 'var(--rc-text-secondary)' }}>
                    Von: <span style={{ color: 'var(--rc-text-primary)', fontWeight: 500 }}>{meta.start_date}</span>
                  </span>
                )}
                {meta.end_date && (
                  <span style={{ color: 'var(--rc-text-secondary)' }}>
                    Bis: <span style={{ color: 'var(--rc-text-primary)', fontWeight: 500 }}>{meta.end_date}</span>
                  </span>
                )}
              </div>

              {/* Progress bar */}
              <div className="flex items-center gap-3 mb-4">
                <div className="flex-1 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--rc-bg-secondary)' }}>
                  <div className="h-full rounded-full transition-all duration-300"
                    style={{ width: totalCount > 0 ? `${(mappedCount / totalCount) * 100}%` : '0%', background: 'var(--rc-primary)' }} />
                </div>
                <span className="text-xs whitespace-nowrap" style={{ color: 'var(--rc-text-secondary)' }}>
                  {mappedCount}/{totalCount} gemappt
                </span>
              </div>

              {/* Item rows */}
              <div className="space-y-2">
                {items.map((item) => {
                  const conf = getNullFloat(item.mapping_confidence);
                  const mapped = isMapped(item);
                  const isSearchOpen = activeSearch === item.item_id;
                  const saving = savingItem === item.item_id;

                  let borderColor = 'var(--rc-border)';
                  let bgStyle = 'var(--rc-bg-card)';
                  if (item.mapping_status === 'user_confirmed') { borderColor = 'var(--rc-success)'; bgStyle = 'rgba(34,197,94,0.05)'; }
                  else if (mapped && conf >= 80) { borderColor = 'rgba(34,197,94,0.3)'; bgStyle = 'rgba(34,197,94,0.04)'; }
                  else if (mapped && conf >= 60) { borderColor = 'var(--rc-warning)'; bgStyle = 'rgba(255,193,7,0.05)'; }

                  return (
                    <div key={item.item_id} className="rounded-xl p-3"
                      style={{ background: bgStyle, border: `1px solid ${borderColor}` }}>
                      <div className="grid gap-3" style={{ gridTemplateColumns: '1fr 20px 1fr' }}>
                        {/* Left: raw text */}
                        <div>
                          <p className="text-sm" style={{ color: 'var(--rc-text-primary)', wordBreak: 'break-word' }}>
                            {item.raw_product_text}
                          </p>
                          <p className="text-xs mt-0.5" style={{ color: 'var(--rc-text-secondary)' }}>
                            {getNullInt(item.quantity)}× · {getNullFloat(item.unit_price).toFixed(2)} €
                          </p>
                        </div>

                        <span className="text-center pt-1" style={{ color: 'var(--rc-text-secondary)' }}>→</span>

                        {/* Right: mapping status / search / create */}
                        <div>
                          {saving ? (
                            <div className="flex items-center gap-2 py-1">
                              <div className="w-3.5 h-3.5 rounded-full border border-t-transparent animate-spin"
                                style={{ borderColor: 'var(--rc-primary)', borderTopColor: 'transparent' }} />
                              <span className="text-xs" style={{ color: 'var(--rc-text-secondary)' }}>Speichert…</span>
                            </div>
                          ) : mapped && !isSearchOpen ? (
                            <button type="button" onClick={() => setActiveSearch(item.item_id)}
                              className="flex items-center gap-2 group w-full text-left">
                              <CheckCircle className="w-3.5 h-3.5 flex-shrink-0" style={{ color: 'var(--rc-success)' }} />
                              <span className="text-sm" style={{ color: 'var(--rc-success)' }}>
                                {item.mapped_name || (item.mapping_status === 'user_confirmed' ? 'Manuell' : `Auto`)}
                              </span>
                              {conf > 0 && conf < 80 && (
                                <span className="text-xs px-1.5 py-0.5 rounded"
                                  style={{ background: `${confColor(conf)}22`, color: confColor(conf) }}>
                                  {conf.toFixed(0)}%
                                </span>
                              )}
                            </button>
                          ) : (
                            <InlineSearch
                              initialQuery={isSearchOpen ? '' : item.raw_product_text}
                              onSelect={(r) => handleSelect(item, r)}
                              onCreateNew={(type, q) => handleCreateNew(item.item_id, type, q)}
                            />
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </>
          )}

          {phase === 'preview' && (
            <div className="space-y-3">
              {(meta.customer_name || meta.start_date) && (
                <div className="rc-card px-3 py-2 flex flex-wrap gap-4 text-xs mb-2" style={{ background: 'var(--rc-bg-secondary)' }}>
                  {meta.customer_name && (
                    <span style={{ color: 'var(--rc-text-secondary)' }}>
                      Kunde: <span style={{ color: 'var(--rc-text-primary)', fontWeight: 500 }}>{meta.customer_name}</span>
                    </span>
                  )}
                  {meta.start_date && (
                    <span style={{ color: 'var(--rc-text-secondary)' }}>
                      {meta.start_date}{meta.end_date ? ` – ${meta.end_date}` : ''}
                    </span>
                  )}
                </div>
              )}
              <p className="text-sm" style={{ color: 'var(--rc-text-secondary)' }}>Diese Items werden zum Job hinzugefügt:</p>
              <div className="space-y-1">
                {previewItems.map((pi) => (
                  <div key={pi.item_id} className="flex items-center justify-between py-2"
                    style={{ borderBottom: '1px solid var(--rc-border)' }}>
                    <div>
                      <span className="text-sm" style={{ color: 'var(--rc-text-primary)' }}>{pi.name}</span>
                      <span className="text-xs ml-2" style={{ color: 'var(--rc-text-secondary)' }}>{pi.raw_text}</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs" style={{ color: 'var(--rc-text-secondary)' }}>
                      <span>{pi.quantity}×</span>
                      <span>{pi.unit_price.toFixed(2)} €</span>
                      <span className="font-medium" style={{ color: 'var(--rc-text-primary)' }}>{pi.line_total.toFixed(2)} €</span>
                    </div>
                  </div>
                ))}
              </div>
              {totalAmount > 0 && (
                <div className="flex justify-end pt-2">
                  <span className="text-sm font-semibold" style={{ color: 'var(--rc-text-primary)' }}>
                    Gesamt: {totalAmount.toFixed(2)} €
                  </span>
                </div>
              )}
              <p className="text-xs text-center pt-2" style={{ color: 'var(--rc-text-secondary)' }}>
                Mappings werden global gespeichert
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 flex items-center justify-between"
          style={{ borderTop: '1px solid var(--rc-border)' }}>
          {phase === 'mapping' && (
            <>
              <span className="text-xs" style={{ color: 'var(--rc-text-secondary)' }}>
                {!hasCustomer && <span style={{ color: 'var(--rc-warning)' }}>Kunden auswählen · </span>}
                {!allMapped && `${totalCount - mappedCount} Item(s) noch offen`}
              </span>
              <button type="button" onClick={handleProceedToPreview} disabled={!allMapped || !hasCustomer}
                className="rc-btn rc-btn-primary rc-btn-sm flex items-center gap-2">
                Weiter <ChevronRight className="w-4 h-4" />
              </button>
            </>
          )}
          {phase === 'preview' && (
            <>
              <button type="button" onClick={() => setPhase('mapping')} className="rc-btn rc-btn-secondary rc-btn-sm">
                Zurück
              </button>
              <button type="button" onClick={handleConfirm} className="rc-btn rc-btn-primary rc-btn-sm flex items-center gap-2">
                <CheckCircle className="w-4 h-4" /> Zum Job hinzufügen
              </button>
            </>
          )}
        </div>
      </div>

      {/* Full create modal — rendered as overlay on top of MappingModal */}
      {activeCreate && (
        <FullCreateModal
          prefill={activeCreate.prefill}
          defaultTab={activeCreate.type}
          onCreated={(result) => {
            const item = items.find(it => it.item_id === activeCreate.itemId);
            if (item) handleSelect(item, result);
            setActiveCreate(null);
          }}
          onClose={() => setActiveCreate(null)}
        />
      )}
    </div>
  );
}
