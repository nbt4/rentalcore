import { useState, useEffect, useRef, useCallback } from 'react';
import { X, CheckCircle, AlertCircle, Search, ChevronRight, ArrowLeft } from 'lucide-react';

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

export interface MappedItem {
  product_id: number;
  name: string;
  quantity: number;
}

export interface MappingModalProps {
  uploadId: number;
  onComplete: (items: MappedItem[]) => void;
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

// ── InlineSearch ────────────────────────────────────────────────────────────

function InlineSearch({ initialQuery, onSelect }: { initialQuery: string; onSelect: (r: SearchResult) => void }) {
  const [query, setQuery] = useState(initialQuery);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { inputRef.current?.focus(); }, []);

  const search = useCallback((q: string) => {
    if (q.trim().length < 2) { setResults([]); return; }
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
      } finally { setLoading(false); }
    }, 300);
  }, []);

  return (
    <div className="relative">
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-500" />
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => { setQuery(e.target.value); search(e.target.value); }}
          placeholder="Produkt suchen…"
          className="w-full pl-8 pr-3 py-1.5 bg-black/30 border border-blue-500/60 rounded-lg text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-400"
        />
      </div>
      {(results.length > 0 || loading) && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-gray-900 border border-white/10 rounded-lg shadow-xl z-50 overflow-hidden">
          {loading && <div className="px-3 py-2 text-xs text-gray-500">Suche…</div>}
          {results.map((r) => (
            <button key={`${r.type}-${r.id}`} type="button" onClick={() => onSelect(r)}
              className="w-full flex items-center justify-between px-3 py-2 text-sm text-left hover:bg-white/5 transition-colors">
              <span className="text-white">{r.name}</span>
              <span className="text-xs text-gray-500 ml-2">{r.sub}</span>
            </button>
          ))}
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
  const [activeSearch, setActiveSearch] = useState<number | null>(null);
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
        await fetch(`/api/pdf/auto-map/${extraction.extraction_id}`, { method: 'POST', credentials: 'include' });
        const res2 = await fetch(`/api/pdf/extraction/${uploadId}`, { credentials: 'include' });
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
    try {
      const body = result.type === 'package'
        ? { package_id: result.id, status: 'user_confirmed' }
        : { product_id: result.id, status: 'user_confirmed' };
      await fetch(`/api/pdf/items/${item.item_id}/mapping`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      setItems((prev) => prev.map((it) => it.item_id !== item.item_id ? it : {
        ...it,
        mapped_product_id: result.type === 'product' ? result.id : null,
        mapped_package_id: result.type === 'package' ? result.id : null,
        mapping_status: 'user_confirmed',
        mapping_confidence: 100,
      }));
    } finally { setSavingItem(null); }
  };

  const handleProceedToPreview = async () => {
    if (!extractionId) return;
    const res = await fetch(`/api/pdf/extractions/${extractionId}/preview`, { credentials: 'include' });
    const data = await res.json();
    setPreviewItems(data.items || []);
    setTotalAmount(data.total_amount || 0);
    setPhase('preview');
  };

  const handleConfirm = async () => {
    if (!extractionId) return;
    await fetch(`/api/pdf/extractions/${extractionId}/finalize`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({}),
    });
    const mapped: MappedItem[] = previewItems
      .filter((pi) => pi.target_type === 'product')
      .map((pi) => ({ product_id: pi.target_id, name: pi.name, quantity: pi.quantity }));
    onComplete(mapped);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="bg-gray-900 border border-white/10 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
          <div className="flex items-center gap-3">
            {phase === 'preview' && (
              <button type="button" onClick={() => setPhase('mapping')} className="p-1 hover:bg-white/10 rounded-lg transition-colors">
                <ArrowLeft className="w-4 h-4" />
              </button>
            )}
            <h2 className="text-base font-semibold text-white">
              {phase === 'preview' ? 'Vorschau — Items zum Job hinzufügen' : 'PDF Mapping'}
            </h2>
          </div>
          <button type="button" onClick={onClose} className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {phase === 'loading' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
              <p className="text-sm text-gray-400">OCR läuft & Auto-Mapping…</p>
            </div>
          )}
          {phase === 'error' && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-red-400">
              <AlertCircle className="w-8 h-8" />
              <p className="text-sm">{errorMsg}</p>
            </div>
          )}
          {phase === 'mapping' && (
            <>
              <div className="flex items-center gap-3 mb-5">
                <div className="flex-1 h-1.5 bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 rounded-full transition-all duration-300"
                    style={{ width: totalCount > 0 ? `${(mappedCount / totalCount) * 100}%` : '0%' }}
                  />
                </div>
                <span className="text-xs text-gray-400 whitespace-nowrap">{mappedCount}/{totalCount} gemappt</span>
              </div>
              <div className="space-y-2">
                {items.map((item) => {
                  const conf = getNullFloat(item.mapping_confidence);
                  const mapped = isMapped(item);
                  const isSearchOpen = activeSearch === item.item_id;
                  const saving = savingItem === item.item_id;
                  let rowBg = 'bg-orange-500/5 border border-orange-500/20';
                  if (item.mapping_status === 'user_confirmed') rowBg = 'bg-green-500/5 border border-green-500/20';
                  else if (mapped && conf >= 80) rowBg = 'bg-green-500/5 border border-green-500/10';
                  else if (mapped && conf >= 60) rowBg = 'bg-yellow-500/5 border border-yellow-500/20';

                  return (
                    <div key={item.item_id} className={`rounded-xl p-3 ${rowBg}`}>
                      <div className="grid grid-cols-[1fr_16px_1fr] gap-3 items-start">
                        <div>
                          <p className="text-sm text-gray-300">{item.raw_product_text}</p>
                          <p className="text-xs text-gray-600 mt-0.5">
                            {getNullInt(item.quantity)}× · {getNullFloat(item.unit_price).toFixed(2)} €
                          </p>
                        </div>
                        <span className="text-gray-600 text-sm text-center pt-1">→</span>
                        <div>
                          {saving ? (
                            <div className="flex items-center gap-2 py-1">
                              <div className="w-3.5 h-3.5 border border-blue-500 border-t-transparent rounded-full animate-spin" />
                              <span className="text-xs text-gray-500">Speichert…</span>
                            </div>
                          ) : mapped && !isSearchOpen ? (
                            <button
                              type="button"
                              onClick={() => setActiveSearch(item.item_id)}
                              className="flex items-center gap-2 group w-full text-left"
                            >
                              <CheckCircle className="w-3.5 h-3.5 text-green-500 flex-shrink-0" />
                              <span className="text-sm text-green-400 group-hover:text-green-300 transition-colors">
                                {item.mapping_status === 'user_confirmed' ? 'Manuell gemappt' : `Auto (${conf.toFixed(0)}%)`}
                              </span>
                              {conf > 0 && conf < 80 && (
                                <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                                  {conf.toFixed(0)}%
                                </span>
                              )}
                            </button>
                          ) : (
                            <InlineSearch
                              initialQuery={isSearchOpen ? '' : item.raw_product_text}
                              onSelect={(r) => handleSelect(item, r)}
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
              <p className="text-sm text-gray-400">Diese Items werden zum Job hinzugefügt:</p>
              <div className="space-y-1">
                {previewItems.map((pi) => (
                  <div key={pi.item_id} className="flex items-center justify-between py-2 border-b border-white/5">
                    <div>
                      <span className="text-sm text-white">{pi.name}</span>
                      <span className="text-xs text-gray-500 ml-2">{pi.raw_text}</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-gray-400">
                      <span>{pi.quantity}×</span>
                      <span>{pi.unit_price.toFixed(2)} €</span>
                      <span className="font-medium text-gray-300">{pi.line_total.toFixed(2)} €</span>
                    </div>
                  </div>
                ))}
              </div>
              {totalAmount > 0 && (
                <div className="flex justify-end pt-2">
                  <span className="text-sm font-semibold text-white">Gesamt: {totalAmount.toFixed(2)} €</span>
                </div>
              )}
              <p className="text-xs text-gray-600 text-center pt-2">Mappings werden global gespeichert</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-white/10 flex items-center justify-between">
          {phase === 'mapping' && (
            <>
              <span className="text-xs text-gray-500">
                {!allMapped && `${totalCount - mappedCount} Item(s) noch nicht gemappt`}
              </span>
              <button
                type="button"
                onClick={handleProceedToPreview}
                disabled={!allMapped}
                className="flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-400 disabled:bg-gray-700 disabled:text-gray-500 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors"
              >
                Weiter <ChevronRight className="w-4 h-4" />
              </button>
            </>
          )}
          {phase === 'preview' && (
            <>
              <button type="button" onClick={() => setPhase('mapping')} className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors">
                Zurück
              </button>
              <button
                type="button"
                onClick={handleConfirm}
                className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-500 text-white rounded-lg text-sm font-medium transition-colors"
              >
                <CheckCircle className="w-4 h-4" /> Zum Job hinzufügen
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
