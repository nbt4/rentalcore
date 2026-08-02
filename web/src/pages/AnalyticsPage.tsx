import { useEffect, useMemo, useState } from 'react';
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

function buildMonthlyData(months: RevenueDrilldown['monthly_revenue']): RevenueData[] {
  const map = new Map<string, RevenueData>();
  const now = new Date();
  for (let i = 5; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
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

const periodLabels: Record<string, string> = {
  all: 'Gesamter Zeitraum',
  '30days': 'Letzte 30 Tage',
  '90days': 'Letzte 90 Tage',
  '1year': 'Letztes Jahr',
};

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
  const [drilldownPath, setDrilldownPath] = useState<string[]>([]);

  useEffect(() => {
    api.get<{ jobs: Job[] }>('/jobs')
      .then((r) => setJobs(r.data.jobs || []))
      .catch((e: any) => toast.error(e))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    setDrilldownLoading(true);
    setDrilldownPath([]);
    analyticsApi.getRevenueDrilldown(period)
      .then((response) => setDrilldown(response.data))
      .catch((error: any) => toast.error(error))
      .finally(() => setDrilldownLoading(false));
  }, [period]);

  const monthly = buildMonthlyData(drilldown?.monthly_revenue || []);
  const totalGrossRevenue = drilldown?.total_gross_revenue ?? jobs.reduce((s, j) => s + (j.final_revenue ?? j.revenue ?? 0), 0);
  const totalNetRevenue = drilldown?.total_net_revenue ?? totalGrossRevenue;
  const totalTaxAmount = drilldown?.total_tax_amount ?? 0;
  const maxRevenue = Math.max(...monthly.map((m) => m.grossRevenue), 1);

  const selectedNodes = useMemo(
    () => resolveDrilldownPath(drilldown?.categories || [], drilldownPath),
    [drilldown, drilldownPath],
  );
  const selectedNode = selectedNodes[selectedNodes.length - 1];
  const visibleDrilldownNodes = selectedNode?.children || drilldown?.categories || [];
  const parentRevenue = selectedNode?.gross_revenue || drilldown?.total_gross_revenue || 0;
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
          <p className="text-gray-400 text-sm mt-1">Umsatz bis auf Produkt, Mietartikel, Dienstleistung und Einzelgerät nachvollziehen</p>
        </div>
        <label className="flex flex-col gap-1 text-xs text-gray-500">
          Analysezeitraum
          <select
            value={period}
            onChange={(event) => setPeriod(event.target.value)}
            className="bg-dark-200 border border-white/10 rounded-lg px-3 py-2 text-sm text-white min-w-48"
          >
            {Object.entries(periodLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 xl:grid-cols-5 gap-4">
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Bruttoumsatz</span>
            <Euro className="w-5 h-5 text-yellow-400" />
          </div>
          <div className="text-2xl font-bold">{formatCurrency(totalGrossRevenue)}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Nettoumsatz</span>
            <ReceiptEuro className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-2xl font-bold">{formatCurrency(totalNetRevenue)}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">MwSt.</span>
            <Euro className="w-5 h-5 text-gray-300" />
          </div>
          <div className="text-2xl font-bold">{formatCurrency(totalTaxAmount)}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Gesamt Jobs</span>
            <Briefcase className="w-5 h-5 text-accent-red" />
          </div>
          <div className="text-2xl font-bold">{drilldown?.job_count ?? jobs.length}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Ø Brutto / Job</span>
            <TrendingUp className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-2xl font-bold">
            {formatCurrency((drilldown?.job_count ?? jobs.length) > 0 ? totalGrossRevenue / (drilldown?.job_count ?? jobs.length) : 0)}
          </div>
        </div>
      </div>

      {/* Revenue drilldown */}
      <div className="glass-dark rounded-xl border border-white/10 overflow-hidden">
        <div className="p-5 border-b border-white/10">
          <div className="flex items-center gap-2">
            <ReceiptEuro className="w-5 h-5 text-accent-red" />
            <h2 className="font-semibold text-white">Umsatz-Drilldown</h2>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            Brutto und Netto werden direkt aus den aktuellen Auftragspositionen, der Brutto-/Netto-Einstellung und dem jeweiligen MwSt.-Satz berechnet. Die Mietmarge verwendet den Bruttoumsatz abzüglich Lieferantenkosten.
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
                className={`flex items-center gap-1 whitespace-nowrap ${drilldownPath.length === 0 ? 'text-white' : 'text-gray-400 hover:text-white'}`}
              >
                <Home className="w-3.5 h-3.5" /> Gesamtumsatz
              </button>
              {selectedNodes.map((node, index) => (
                <span key={node.id} className="flex items-center gap-1.5 whitespace-nowrap">
                  <ChevronRight className="w-3.5 h-3.5 text-gray-600" />
                  <button
                    type="button"
                    onClick={() => setDrilldownPath(drilldownPath.slice(0, index + 1))}
                    className={index === selectedNodes.length - 1 ? 'text-white' : 'text-gray-400 hover:text-white'}
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
                    const canOpen = node.children?.length > 0;
                    return (
                      <tr
                        key={node.id}
                        onClick={() => canOpen && setDrilldownPath([...drilldownPath, node.id])}
                        className={canOpen ? 'hover:bg-white/[0.04] cursor-pointer transition-colors' : 'hover:bg-white/[0.02]'}
                      >
                        <td className="px-5 py-3.5">
                          <div className="flex items-center gap-2 min-w-0">
                            <span className="font-medium text-white truncate">{node.label}</span>
                            {canOpen && <ChevronRight className="w-4 h-4 text-gray-500 shrink-0" />}
                            {node.type === 'device' && <span className="text-[0.65rem] text-gray-600 uppercase">Gerät</span>}
                          </div>
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
                    <tr><td colSpan={7} className="px-5 py-10 text-center text-gray-500">Für diesen Zeitraum liegen keine Umsatzdaten vor.</td></tr>
                  )}
                </tbody>
              </table>
            </div>

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
          <h2 className="font-semibold text-white">Monatlicher Umsatz (letzte 6 Monate)</h2>
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
