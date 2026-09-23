import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { BarChart2, TrendingUp, Euro, Briefcase, ChevronRight, Home, ReceiptEuro } from 'lucide-react';
import { analyticsApi, api } from '../lib/api';
import type { Job, RevenueDrilldown, RevenueDrilldownNode } from '../lib/api';
import { toast } from '../lib/toast';

interface RevenueData {
  month: string;
  netRevenue: number;
  grossRevenue: number;
  jobs: number;
}

function buildMonthlyData(months: RevenueDrilldown['monthly_revenue'], scope: 'realized' | 'pipeline'): RevenueData[] {
  const map = new Map<string, RevenueData>();
  const now = new Date();
  for (let i = 0; i < 6; i++) {
    const offset = scope === 'pipeline' ? i : i - 5;
    const d = new Date(now.getFullYear(), now.getMonth() + offset, 1);
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    map.set(key, {
      month: d.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' }),
      netRevenue: 0,
      grossRevenue: 0,
      jobs: 0,
    });
  }
  for (const month of months) {
    const entry = map.get(month.month);
    if (entry) {
      entry.netRevenue = month.net_revenue;
      entry.grossRevenue = month.gross_revenue;
      entry.jobs = month.job_count;
    }
  }
  return Array.from(map.values());
}

const formatCurrency = (value: number) => value.toLocaleString('de-DE', {
  style: 'currency',
  currency: 'EUR',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

function periodLabels(scope: 'realized' | 'pipeline'): Record<string, string> {
  return scope === 'pipeline'
    ? { all: 'Gesamter Zeitraum', '30days': 'Nächste 30 Tage', '90days': 'Nächste 90 Tage', '1year': 'Nächstes Jahr' }
    : { all: 'Gesamter Zeitraum', '30days': 'Letzte 30 Tage', '90days': 'Letzte 90 Tage', '1year': 'Letztes Jahr' };
}

function resolveDrilldownPath(categories: RevenueDrilldownNode[], path: string[]) {
  const nodes: RevenueDrilldownNode[] = [];
  let candidates = categories;
  for (const id of path) {
    const node = candidates.find((candidate) => candidate.id === id);
    if (!node) break;
    nodes.push(node);
    candidates = node.children || [];
  }
  return nodes;
}

export function AnalyticsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [drilldown, setDrilldown] = useState<RevenueDrilldown | null>(null);
  const [drilldownLoading, setDrilldownLoading] = useState(true);
  const [period, setPeriod] = useState('all');
  const [scope, setScope] = useState<'realized' | 'pipeline'>('realized');
  const [drilldownPath, setDrilldownPath] = useState<string[]>([]);

  useEffect(() => {
    api.get<{ jobs: Job[] }>('/jobs')
      .then((r) => setJobs(r.data.jobs || []))
      .catch((e: any) => toast.error(e))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    let active = true;
    setDrilldownLoading(true);
    setDrilldown(null);
    setDrilldownPath([]);
    analyticsApi.getRevenueDrilldown(period, scope)
      .then((response) => { if (active) setDrilldown(response.data); })
      .catch((error: any) => { if (active) toast.error(error); })
      .finally(() => { if (active) setDrilldownLoading(false); });
    return () => { active = false; };
  }, [period, scope]);

  const monthly = buildMonthlyData(drilldown?.monthly_revenue || [], scope);
  const totalGrossRevenue = drilldown?.total_gross_revenue ?? 0;
  const totalNetRevenue = drilldown?.total_net_revenue ?? totalGrossRevenue;
  const totalTaxAmount = drilldown?.total_tax_amount ?? 0;
  const jobCount = drilldown?.job_count ?? 0;
  const maxRevenue = Math.max(...monthly.map((m) => m.grossRevenue), 1);

  const selectedNodes = useMemo(
    () => resolveDrilldownPath(drilldown?.categories || [], drilldownPath),
    [drilldown, drilldownPath],
  );
  const selectedNode = selectedNodes[selectedNodes.length - 1];
  const visibleDrilldownNodes = selectedNode?.children || drilldown?.categories || [];
  const parentRevenue = selectedNode?.gross_revenue ?? drilldown?.total_gross_revenue ?? 0;
  const ownProductRevenue = drilldown?.categories.find((node) => node.id === 'own-products')?.gross_revenue || 0;
  const serviceRevenue = drilldown?.categories.find((node) => node.id === 'services')?.gross_revenue || 0;

  const statusGroups = jobs.reduce<Record<string, number>>((acc, j) => {
    const s = j.status?.status || 'Unbekannt';
    acc[s] = (acc[s] || 0) + 1;
    return acc;
  }, {});

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><BarChart2 className="w-6 h-6 text-accent-red" /> Analyse</h1>
          <p className="text-gray-400 text-sm mt-1">{scope === 'realized' ? 'Realisierter Umsatz aus abgeschlossenen Jobs' : 'Voraussichtlicher Umsatz aus Jobs in Planung und Bestätigung'} · bis zur Position und zum Job nachvollziehbar</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <label className="flex flex-col gap-1 text-xs text-gray-500">
            Umsatzansicht
            <select
              value={scope}
              onChange={(event) => setScope(event.target.value as 'realized' | 'pipeline')}
              className="bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white min-w-48"
            >
              <option value="realized">Realisierter Umsatz</option>
              <option value="pipeline">Pipeline · geplant und bestätigt</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-xs text-gray-500">
            Analysezeitraum
            <select
              value={period}
              onChange={(event) => setPeriod(event.target.value)}
              className="bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white min-w-48"
            >
              {Object.entries(periodLabels(scope)).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
          </label>
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 xl:grid-cols-5 gap-4">
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">{scope === 'realized' ? 'Realisierter Bruttoumsatz' : 'Pipeline brutto'}</span>
            <Euro className="w-5 h-5 text-yellow-400" />
          </div>
          <div className="text-2xl font-bold">{drilldownLoading ? '…' : drilldown ? formatCurrency(totalGrossRevenue) : '—'}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">{scope === 'realized' ? 'Realisierter Nettoumsatz' : 'Pipeline netto'}</span>
            <ReceiptEuro className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-2xl font-bold">{drilldownLoading ? '…' : drilldown ? formatCurrency(totalNetRevenue) : '—'}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">MwSt.</span>
            <Euro className="w-5 h-5 text-gray-300" />
          </div>
          <div className="text-2xl font-bold">{drilldownLoading ? '…' : drilldown ? formatCurrency(totalTaxAmount) : '—'}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">{scope === 'realized' ? 'Abgeschlossene Jobs' : 'Pipeline Jobs'}</span>
            <Briefcase className="w-5 h-5 text-accent-red" />
          </div>
          <div className="text-2xl font-bold">{drilldownLoading ? '…' : drilldown ? jobCount : '—'}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Ø Brutto / Job</span>
            <TrendingUp className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-2xl font-bold">
            {drilldownLoading ? '…' : drilldown ? formatCurrency(jobCount > 0 ? totalGrossRevenue / jobCount : 0) : '—'}
          </div>
        </div>
      </div>

      {/* Revenue drilldown */}
      <div className="glass-dark rounded-xl border border-white/10 overflow-hidden">
        <div className="p-5 border-b border-white/10">
          <div className="flex items-center gap-2">
            <ReceiptEuro className="w-5 h-5 text-accent-red" />
            <h2 className="font-semibold text-white">{scope === 'realized' ? 'Realisierter Umsatz' : 'Pipeline'} · Drilldown</h2>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            {scope === 'realized' ? 'Nur abgeschlossene Jobs zählen als realisierter Umsatz.' : 'Planung und bestätigte Jobs erscheinen ausschließlich als Pipeline.'} Brutto und Netto stammen aus den Auftragspositionen; die Mietmarge zieht Lieferantenkosten ab.
          </p>
        </div>

        {drilldownLoading ? (
          <div className="flex justify-center py-14"><div className="w-7 h-7 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>
        ) : drilldown ? (
          <>
            <div className="grid grid-cols-2 lg:grid-cols-5 gap-px bg-white/10 border-b border-white/10">
              {[
                ['Eigene Produkte (brutto)', ownProductRevenue],
                ['Mietumsatz (brutto)', drilldown.rental_gross_revenue],
                ['Mietausgaben', drilldown.rental_cost],
                ['Mietmarge', drilldown.rental_margin],
                ['Dienstleistungen', serviceRevenue],
              ].map(([label, value]) => (
                <div key={String(label)} className="bg-dark-200/90 px-4 py-3">
                  <div className="text-[0.7rem] uppercase tracking-wide text-gray-500">{label}</div>
                  <div className={`font-semibold mt-1 ${label === 'Mietmarge' && Number(value) < 0 ? 'text-red-400' : 'text-white'}`}>
                    {formatCurrency(Number(value))}
                  </div>
                </div>
              ))}
            </div>

            <div className="px-5 py-3 border-b border-white/10 flex items-center gap-1.5 text-sm overflow-x-auto">
              <button
                type="button"
                onClick={() => setDrilldownPath([])}
                className={`flex items-center gap-1 whitespace-nowrap rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent-red)] ${drilldownPath.length === 0 ? 'text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <Home className="w-3.5 h-3.5" /> Gesamtumsatz
              </button>
              {selectedNodes.map((node, index) => (
                <span key={node.id} className="flex items-center gap-1.5 whitespace-nowrap">
                  <ChevronRight className="w-3.5 h-3.5 text-gray-600" />
                  <button
                    type="button"
                    onClick={() => setDrilldownPath(drilldownPath.slice(0, index + 1))}
                    className={`rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent-red)] ${index === selectedNodes.length - 1 ? 'text-white' : 'text-gray-400 hover:text-white'}`}
                  >
                    {node.label}
                  </button>
                </span>
              ))}
            </div>

            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px] text-sm">
                <thead className="text-xs uppercase tracking-wide text-gray-500 bg-white/[0.02]">
                  <tr>
                    <th className="text-left px-5 py-3">Aufschlüsselung</th>
                    <th className="text-right px-4 py-3">Brutto</th>
                    <th className="text-right px-4 py-3">Netto</th>
                    <th className="text-left px-4 py-3 w-44">Anteil</th>
                    <th className="text-right px-4 py-3">Ausgaben</th>
                    <th className="text-right px-4 py-3">Marge</th>
                    <th className="text-right px-5 py-3">Aufträge</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {visibleDrilldownNodes.map((node) => {
                    const share = parentRevenue > 0 ? Math.max(0, Math.min(100, node.gross_revenue / parentRevenue * 100)) : 0;
                    const canOpen = (node.children?.length || 0) > 0 || (node.jobs?.length || 0) > 0;
                    return (
                      <tr
                        key={node.id}
                        className="hover:bg-white/[0.04] transition-colors"
                      >
                        <td className="px-5 py-3.5">
                          {canOpen ? (
                            <button
                              type="button"
                              onClick={() => setDrilldownPath([...drilldownPath, node.id])}
                              className="flex items-center gap-2 min-w-0 min-h-11 text-left rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent-red)]"
                              aria-label={`${node.label} öffnen, ${node.bookings} Aufträge`}
                            >
                              <span className="font-medium text-white truncate">{node.label}</span>
                              <ChevronRight className="w-4 h-4 text-gray-500 shrink-0" />
                              {node.type === 'device' && <span className="text-[0.65rem] text-gray-600 uppercase">Gerät</span>}
                            </button>
                          ) : <span className="font-medium text-white">{node.label}</span>}
                          {node.quantity > 0 && <div className="text-xs text-gray-600 mt-0.5">Menge {node.quantity.toLocaleString('de-DE')}</div>}
                        </td>
                        <td className="px-4 py-3.5 text-right font-medium text-white">{formatCurrency(node.gross_revenue)}</td>
                        <td className="px-4 py-3.5 text-right text-gray-300">{formatCurrency(node.net_revenue)}</td>
                        <td className="px-4 py-3.5">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 bg-white/5 rounded-full h-1.5 overflow-hidden">
                              <div className="h-full bg-accent-red/70 rounded-full" style={{ width: `${share}%` }} />
                            </div>
                            <span className="text-xs text-gray-500 w-12 text-right">{share.toLocaleString('de-DE', { maximumFractionDigits: 1 })}%</span>
                          </div>
                        </td>
                        <td className="px-4 py-3.5 text-right text-gray-300">{node.has_cost ? formatCurrency(node.cost) : '—'}</td>
                        <td className={`px-4 py-3.5 text-right font-medium ${node.has_cost && node.margin < 0 ? 'text-red-400' : node.has_cost ? 'text-green-400' : 'text-gray-600'}`}>
                          {node.has_cost ? <>{formatCurrency(node.margin)} <span className="text-xs opacity-70">({node.margin_percent.toLocaleString('de-DE', { maximumFractionDigits: 1 })}%)</span></> : '—'}
                        </td>
                        <td className="px-5 py-3.5 text-right text-gray-400">{node.bookings}</td>
                      </tr>
                    );
                  })}
                  {visibleDrilldownNodes.length === 0 && (
                    <tr><td colSpan={7} className="px-5 py-10 text-center text-gray-500">{selectedNode?.jobs?.length ? 'Die zugehörigen Aufträge stehen unten.' : 'Für diesen Zeitraum liegen keine Umsatzdaten vor.'}</td></tr>
                  )}
                </tbody>
              </table>
            </div>

            {selectedNode?.jobs && selectedNode.jobs.length > 0 && (
              <div className="border-t border-white/10">
                <div className="px-5 py-3">
                  <h3 className="font-semibold text-white">Aufträge zu {selectedNode.label}</h3>
                  <p className="text-xs text-gray-500">{selectedNode.bookings} Jobs · Jobnummer, Titel und zugehörige Position</p>
                </div>
                <div className="px-5 pb-5">
                  <div className="suite-table-wrap">
                    <table className="w-full min-w-[640px] text-sm">
                      <thead><tr><th className="text-left px-4 py-3">Job</th><th className="text-left px-4 py-3">Titel</th><th className="text-left px-4 py-3">Position</th><th className="text-right px-4 py-3">Brutto</th><th className="text-right px-4 py-3">Netto</th></tr></thead>
                      <tbody>
                        {selectedNode.jobs.map((booking) => (
                          <tr key={`${booking.job_id}:${booking.position_id}`}>
                            <td className="px-4 py-3"><Link className="font-medium text-[var(--color-accent-red)] underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent-red)]" to={`/jobs/${booking.job_id}`}>{booking.job_code || `Job ${booking.job_id}`}</Link></td>
                            <td className="px-4 py-3">{booking.job_title || 'Ohne Titel'}</td>
                            <td className="px-4 py-3">{booking.position_label}</td>
                            <td className="px-4 py-3 text-right tabular-nums">{formatCurrency(booking.gross_revenue)}</td>
                            <td className="px-4 py-3 text-right tabular-nums">{formatCurrency(booking.net_revenue)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {drilldown.unattributed_revenue !== 0 && (
              <div className="px-5 py-3 border-t border-white/10 text-xs text-yellow-400/80 bg-yellow-500/[0.03]">
                {formatCurrency(drilldown.unattributed_revenue)} konnten keiner Auftragsposition zugeordnet werden und bleiben im Gesamtumsatz enthalten.
              </div>
            )}
          </>
        ) : (
          <div className="py-12 text-center text-gray-500">Der Umsatz-Drilldown konnte nicht geladen werden.</div>
        )}
      </div>

      {/* Revenue chart */}
      <div className="glass-dark rounded-xl border border-white/10 p-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
          <h2 className="font-semibold text-white">{scope === 'realized' ? 'Monatlicher Umsatz (letzte 6 Monate)' : 'Monatliche Pipeline (nächste 6 Monate)'}</h2>
          <div className="flex items-center gap-4 text-xs text-gray-400">
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm bg-accent-red" /> Brutto</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm bg-green-500/70" /> Netto</span>
          </div>
        </div>
        <div className="flex items-end gap-3 h-40">
          {monthly.map((m) => (
            <div key={m.month} className="flex-1 flex flex-col items-center gap-2">
              <span className="text-xs text-gray-400">
                €{m.grossRevenue >= 1000 ? `${(m.grossRevenue / 1000).toFixed(1)}k` : m.grossRevenue.toFixed(0)}
              </span>
              <div className="w-full flex items-end gap-1 h-full">
                <div
                  className="w-1/2 rounded-t-sm bg-accent-red/70 hover:bg-accent-red transition-colors"
                  style={{ height: `${(m.grossRevenue / maxRevenue) * 100}%`, minHeight: m.grossRevenue > 0 ? '4px' : '0' }}
                  title={`Brutto ${formatCurrency(m.grossRevenue)}`}
                />
                <div
                  className="w-1/2 rounded-t-sm bg-green-500/50 hover:bg-green-500/70 transition-colors"
                  style={{ height: `${(m.netRevenue / maxRevenue) * 100}%`, minHeight: m.netRevenue > 0 ? '4px' : '0' }}
                  title={`Netto ${formatCurrency(m.netRevenue)}`}
                />
              </div>
              <span className="text-xs text-gray-400">{m.month}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Status distribution */}
      {Object.keys(statusGroups).length > 0 && (
        <div className="glass-dark rounded-xl border border-white/10 p-6">
          <h2 className="font-semibold text-white mb-4">Jobs nach Status</h2>
          <div className="space-y-3">
            {Object.entries(statusGroups).sort(([, a], [, b]) => b - a).map(([status, count]) => (
              <div key={status} className="flex items-center gap-3">
                <div className="w-28 text-sm text-gray-400 truncate">{status}</div>
                <div className="flex-1 bg-white/5 rounded-full h-2 overflow-hidden">
                  <div
                    className="h-full bg-accent-red/70 rounded-full"
                    style={{ width: `${(count / jobs.length) * 100}%` }}
                  />
                </div>
                <div className="text-sm text-gray-300 w-8 text-right">{count}</div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
