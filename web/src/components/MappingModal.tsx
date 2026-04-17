import { useState, useEffect, useRef, useCallback } from 'react';
import { X, CheckCircle, AlertCircle, Search, ChevronRight, ArrowLeft, Plus } from 'lucide-react';

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
  type: 'product' | 'package';
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
    && (getNullInt(item.mapped_product_id) > 0 || getNullInt(item.mapped_package_id) > 0);
}

// ── CreateNewForm ───────────────────────────────────────────────────────────

interface CreateNewFormProps {
  prefill: string;
  createType: 'product' | 'package';
  onCreated: (result: SearchResult) => void;
  onCancel: () => void;
}

function CreateNewForm({ prefill, createType, onCreated, onCancel }: CreateNewFormProps) {
  const [name, setName] = useState(prefill);
  const [deviceCount, setDeviceCount] = useState('');
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');

  const submit = async () => {
    if (!name.trim()) { setErr('Name ist erforderlich.'); return; }
    setSaving(true); setErr('');
    try {
      const url = createType === 'product' ? '/api/pdf/product-quick-create' : '/api/pdf/package-quick-create';
      const body: Record<string, unknown> = { name: name.trim() };
      if (createType === 'product' && deviceCount) body.device_count = parseInt(deviceCount, 10) || 0;
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      const d = await res.json();
      if (!res.ok) throw new Error(d.error || 'Fehler');
      onCreated({
        id: createType === 'product' ? d.product_id : d.package_id,
        name: d.name,
        type: createType,
        sub: createType === 'product'
          ? `Produkt${d.devices_created ? ` · ${d.devices_created} Geräte angelegt` : ''}`
          : 'Paket',
      });
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Fehler');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mt-2 p-3 rounded-lg border" style={{ background: 'var(--rc-bg-card)', borderColor: 'var(--rc-border)' }}>
      <p className="text-xs font-semibold mb-2" style={{ color: 'var(--rc-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        Neues {createType === 'product' ? 'Produkt' : 'Paket'} anlegen
      </p>
      <input
        autoFocus
        value={name}
        onChange={e => setName(e.target.value)}
        onKeyDown={e => e.key === 'Enter' && submit()}
        placeholder="Name…"
        className="rc-input rc-input-sm w-full mb-2"
      />
      {createType === 'product' && (
        <input
          type="number"
          min="0"
          max="500"
          value={deviceCount}
          onChange={e => setDeviceCount(e.target.value)}
          placeholder="Anzahl Geräte (optional)"
          className="rc-input rc-input-sm w-full mb-2"
        />
      )}
      {err && <p className="text-xs mb-2" style={{ color: 'var(--rc-danger)' }}>{err}</p>}
      <div className="flex gap-2">
        <button type="button" onClick={submit} disabled={saving} className="rc-btn rc-btn-primary rc-btn-sm" style={{ flex: 1 }}>
          {saving ? '…' : 'Anlegen'}
        </button>
        <button type="button" onClick={onCancel} className="rc-btn rc-btn-secondary rc-btn-sm">Abbrechen</button>
      </div>
    </div>
  );
}

// ── InlineSearch ────────────────────────────────────────────────────────────

interface InlineSearchProps {
  initialQuery: string;
  onSelect: (r: SearchResult) => void;
  onCreateNew: (type: 'product' | 'package', query: string) => void;
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
        const [pRes, pkgRes] = await Promise.all([
          fetch(`/api/pdf/products/search?q=${encodeURIComponent(q)}&limit=5`, { credentials: 'include' }),
          fetch(`/api/pdf/packages/search?q=${encodeURIComponent(q)}&limit=3`, { credentials: 'include' }),
        ]);
        const pd = pRes.ok ? await pRes.json() : {};
        const pkd = pkgRes.ok ? await pkgRes.json() : {};
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
        setResults([...products, ...packages]);
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
          placeholder="Produkt oder Paket suchen…"
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
              <div className="flex gap-1 px-2 pb-2">
                <button type="button" onClick={() => onCreateNew('product', query)}
                  className="rc-btn rc-btn-sm rc-btn-outline flex items-center gap-1" style={{ flex: 1, fontSize: '11px' }}>
                  <Plus className="w-3 h-3" /> Produkt anlegen
                </button>
                <button type="button" onClick={() => onCreateNew('package', query)}
                  className="rc-btn rc-btn-sm rc-btn-outline flex items-center gap-1" style={{ flex: 1, fontSize: '11px' }}>
                  <Plus className="w-3 h-3" /> Paket anlegen
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
  const [activeCreate, setActiveCreate] = useState<{ itemId: number; type: 'product' | 'package'; prefill: string } | null>(null);
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

  const handleSelect = async (item: ExtractionItem, result: SearchResult) => {
    setSavingItem(item.item_id);
    setActiveSearch(null);
    setActiveCreate(null);
    try {
      const body = result.type === 'package'
        ? { package_id: result.id, status: 'user_confirmed' }
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
        mapping_status: 'user_confirmed',
        mapping_confidence: 100,
        mapped_name: result.name,
      }));
    } finally { setSavingItem(null); }
  };

  const handleCreateNew = (itemId: number, type: 'product' | 'package', query: string) => {
    setActiveSearch(null);
    setActiveCreate({ itemId, type, prefill: query });
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
              {/* Meta row: customer + dates */}
              {(meta.customer_name || meta.start_date) && (
                <div className="rc-card mb-4 px-3 py-2 flex flex-wrap gap-4 text-xs" style={{ background: 'var(--rc-bg-secondary)' }}>
                  {meta.customer_name && (
                    <span style={{ color: 'var(--rc-text-secondary)' }}>
                      Kunde: <span style={{ color: 'var(--rc-text-primary)', fontWeight: 500 }}>{meta.customer_name}</span>
                    </span>
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
              )}

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
                  const isCreating = activeCreate?.itemId === item.item_id;
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
                          ) : isCreating ? (
                            <CreateNewForm
                              prefill={activeCreate.prefill}
                              createType={activeCreate.type}
                              onCreated={(r) => handleSelect(item, r)}
                              onCancel={() => setActiveCreate(null)}
                            />
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
                {!allMapped && `${totalCount - mappedCount} Item(s) noch offen`}
              </span>
              <button type="button" onClick={handleProceedToPreview} disabled={!allMapped}
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
    </div>
  );
}
