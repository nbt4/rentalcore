import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Briefcase, Users, TrendingUp, Calendar, ArrowRight } from 'lucide-react';
import { api } from '../lib/api';
import type { Job } from '../lib/api';
import { toast } from '../lib/toast';

interface Stats {
  total_jobs?: number;
  active_jobs?: number;
  total_customers?: number;
  revenue_month?: number;
}

export function Dashboard() {
  const [stats, setStats] = useState<Stats>({});
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const [jobsRes, customersRes] = await Promise.all([
          api.get<{ jobs: Job[] }>('/jobs'),
          api.get<{ customers: unknown[] }>('/customers'),
        ]);
        const jobs = jobsRes.data.jobs || [];
        const customers = customersRes.data.customers || [];
        const activeJobs = jobs.filter((j) =>
          !['completed', 'archived', 'cancelled'].includes((j.status?.status || '').toLowerCase())
        );
        setStats({
          total_jobs: jobs.length,
          active_jobs: activeJobs.length,
          total_customers: customers.length,
          revenue_month: jobs
            .filter((j) => {
              const d = j.created_at ? new Date(j.created_at) : null;
              if (!d) return false;
              const now = new Date();
              return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
            })
            .reduce((sum, j) => sum + (j.final_revenue ?? j.revenue ?? 0), 0),
        });
        setRecentJobs(jobs.slice(0, 5));
      } catch (e) {
        toast.error(e);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const statCards = [
    { label: 'Aktive Jobs', value: stats.active_jobs ?? 0, icon: Briefcase, color: 'text-accent-red', link: '/jobs' },
    { label: 'Gesamt Jobs', value: stats.total_jobs ?? 0, icon: Calendar, color: 'text-blue-400', link: '/jobs' },
    { label: 'Kontakte', value: stats.total_customers ?? 0, icon: Users, color: 'text-green-400' },
    {
      label: 'Umsatz (Monat)',
      value: `€${(stats.revenue_month ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
      icon: TrendingUp,
      color: 'text-yellow-400',
      link: '/analytics',
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="w-10 h-10 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="text-gray-400 text-sm mt-1">Übersicht über deine Aufträge</p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map(({ label, value, icon: Icon, color, link }) => {
          const cls = "glass-dark rounded-xl p-5 border border-white/10 hover:border-white/20 transition-all hover:bg-white/5";
          const inner = (
            <>
              <div className="flex items-center justify-between mb-3">
                <span className="text-gray-400 text-sm">{label}</span>
                <Icon className={`w-5 h-5 ${color}`} />
              </div>
              <div className="text-2xl font-bold text-white">{value}</div>
            </>
          );
          return link
            ? <Link key={label} to={link} className={cls}>{inner}</Link>
            : <div key={label} className={cls}>{inner}</div>;
        })}
      </div>

      {/* Recent jobs */}
      <div className="glass-dark rounded-xl border border-white/10">
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
          <h2 className="font-semibold text-white">Letzte Jobs</h2>
          <Link to="/jobs" className="text-accent-red hover:text-accent-red/80 text-sm flex items-center gap-1">
            Alle <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
        <div className="divide-y divide-white/5">
          {recentJobs.length === 0 ? (
            <div className="px-6 py-8 text-center text-gray-500 text-sm">Keine Jobs vorhanden</div>
          ) : (
            recentJobs.map((job) => (
              <Link
                key={job.jobID}
                to={`/jobs/${job.jobID}`}
                className="flex items-center justify-between px-6 py-4 hover:bg-white/5 transition-colors"
              >
                <div>
                  <div className="font-medium text-white">{job.job_code}</div>
                  <div className="text-sm text-gray-400">
                    {job.customer
                      ? (job.customer.companyname || `${job.customer.firstname || ''} ${job.customer.lastname || ''}`.trim() || 'Kein Kontakt')
                      : (job.customer_name?.trim() || 'Kein Kontakt')}
                    {job.description && ` · ${job.description}`}
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {job.status && (
                    <span className="px-2 py-1 rounded-full text-xs font-medium bg-accent-red/10 text-accent-red border border-accent-red/20">
                      {job.status.status}
                    </span>
                  )}
                  <ArrowRight className="w-4 h-4 text-gray-500" />
                </div>
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
