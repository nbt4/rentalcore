import { useEffect, useState } from 'react';
import { BarChart2, TrendingUp, Euro, Briefcase } from 'lucide-react';
import { api } from '../lib/api';
import type { Job } from '../lib/api';

interface RevenueData {
  month: string;
  revenue: number;
  jobs: number;
}

function buildMonthlyData(jobs: Job[]): RevenueData[] {
  const map = new Map<string, RevenueData>();
  const now = new Date();
  for (let i = 5; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    map.set(key, {
      month: d.toLocaleDateString('de-DE', { month: 'short', year: '2-digit' }),
      revenue: 0,
      jobs: 0,
    });
  }
  for (const job of jobs) {
    const d = job.created_at ? new Date(job.created_at) : null;
    if (!d) continue;
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    const entry = map.get(key);
    if (entry) {
      entry.revenue += job.final_revenue ?? job.revenue ?? 0;
      entry.jobs += 1;
    }
  }
  return Array.from(map.values());
}

export function AnalyticsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<{ jobs: Job[] }>('/jobs')
      .then((r) => setJobs(r.data.jobs || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const monthly = buildMonthlyData(jobs);
  const totalRevenue = jobs.reduce((s, j) => s + (j.final_revenue ?? j.revenue ?? 0), 0);
  const maxRevenue = Math.max(...monthly.map((m) => m.revenue), 1);

  const statusGroups = jobs.reduce<Record<string, number>>((acc, j) => {
    const s = j.status?.status || 'Unbekannt';
    acc[s] = (acc[s] || 0) + 1;
    return acc;
  }, {});

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><BarChart2 className="w-6 h-6 text-accent-red" /> Analyse</h1>
        <p className="text-gray-400 text-sm mt-1">Übersicht über Umsatz und Job-Statistiken</p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Gesamtumsatz</span>
            <Euro className="w-5 h-5 text-yellow-400" />
          </div>
          <div className="text-2xl font-bold">€{totalRevenue.toLocaleString('de-DE', { minimumFractionDigits: 2 })}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Gesamt Jobs</span>
            <Briefcase className="w-5 h-5 text-accent-red" />
          </div>
          <div className="text-2xl font-bold">{jobs.length}</div>
        </div>
        <div className="glass-dark rounded-xl border border-white/10 p-5">
          <div className="flex items-center justify-between mb-2">
            <span className="text-gray-400 text-sm">Ø Umsatz / Job</span>
            <TrendingUp className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-2xl font-bold">
            €{jobs.length > 0 ? (totalRevenue / jobs.length).toLocaleString('de-DE', { minimumFractionDigits: 2 }) : '0,00'}
          </div>
        </div>
      </div>

      {/* Revenue chart */}
      <div className="glass-dark rounded-xl border border-white/10 p-6">
        <h2 className="font-semibold text-white mb-6">Monatlicher Umsatz (letzte 6 Monate)</h2>
        <div className="flex items-end gap-3 h-40">
          {monthly.map((m) => (
            <div key={m.month} className="flex-1 flex flex-col items-center gap-2">
              <span className="text-xs text-gray-400">
                €{m.revenue >= 1000 ? `${(m.revenue / 1000).toFixed(1)}k` : m.revenue.toFixed(0)}
              </span>
              <div
                className="w-full rounded-t-sm bg-accent-red/60 hover:bg-accent-red transition-colors"
                style={{ height: `${(m.revenue / maxRevenue) * 100}%`, minHeight: m.revenue > 0 ? '4px' : '0' }}
              />
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
