import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  AlertTriangle,
  ArrowRight,
  BarChart3,
  Briefcase,
  CalendarDays,
  CheckCircle2,
  FileText,
  Plus,
  RefreshCw,
  Search,
  TrendingUp,
  Users,
} from 'lucide-react';
import { api } from '../lib/api';
import type { Customer, Job } from '../lib/api';
import { useAuth } from '../contexts/AuthContext';
import { toast } from '../lib/toast';
import { suiteDateLabel, suiteGreeting } from '../lib/cores-design';

const DAY_IN_MS = 86_400_000;
const FINISHED_STATUSES = [
  'completed', 'finished', 'cancelled', 'canceled', 'archived', 'paid', 'on hold',
  'abgeschlossen', 'beendet', 'storniert', 'abgebrochen', 'archiviert', 'bezahlt', 'pausiert',
];

function startOfDay(date = new Date()) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function parseDate(value?: string | null) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function isFinished(job: Job) {
  const status = (job.status?.status || '').trim().toLowerCase();
  return FINISHED_STATUSES.some((finished) => status === finished || status.includes(finished));
}

function customerName(job: Job) {
  const customer = job.customer;
  if (!customer) return 'Kein Kontakt';
  return customer.companyname
    || `${customer.firstname || ''} ${customer.lastname || ''}`.trim()
    || 'Kein Kontakt';
}

function formatMoney(value: number) {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0,
  }).format(value);
}

function formatDate(value?: string | null, options?: Intl.DateTimeFormatOptions) {
  const date = parseDate(value);
  if (!date) return 'Termin offen';
  return new Intl.DateTimeFormat('de-DE', options || { day: '2-digit', month: 'short', year: 'numeric' }).format(date);
}

function statusClasses(status?: string) {
  const value = (status || '').toLowerCase();
  if (value.includes('aktiv') || value.includes('active') || value.includes('confirm')) {
    return 'border-green-500/20 bg-green-500/10 text-green-300';
  }
  if (value.includes('abgeschlossen') || value.includes('completed') || value.includes('paid')) {
    return 'border-blue-500/20 bg-blue-500/10 text-blue-300';
  }
  if (value.includes('cancel') || value.includes('storn') || value.includes('abgebrochen')) {
    return 'border-red-500/20 bg-red-500/10 text-red-300';
  }
  return 'border-white/10 bg-white/5 text-gray-300';
}

function scheduleLabel(job: Job, today: Date) {
  const start = parseDate(job.startDate);
  const end = parseDate(job.endDate);
  if (end && startOfDay(end).getTime() < today.getTime()) {
    const days = Math.max(1, Math.round((today.getTime() - startOfDay(end).getTime()) / DAY_IN_MS));
    return { text: `${days} T. überfällig`, classes: 'text-red-300 bg-red-500/10' };
  }
  if (start && startOfDay(start).getTime() <= today.getTime() && (!end || startOfDay(end).getTime() >= today.getTime())) {
    return { text: 'Läuft heute', classes: 'text-green-300 bg-green-500/10' };
  }
  if (start) {
    const days = Math.max(1, Math.ceil((startOfDay(start).getTime() - today.getTime()) / DAY_IN_MS));
    return { text: days === 1 ? 'Morgen' : `In ${days} Tagen`, classes: 'text-blue-300 bg-blue-500/10' };
  }
  return { text: 'Termin offen', classes: 'text-gray-400 bg-white/5' };
}

export function Dashboard() {
  const { user } = useAuth();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);

  const loadDashboard = useCallback(async () => {
    setLoading(true);
    setLoadFailed(false);
    try {
      const [jobsRes, customersRes] = await Promise.all([
        api.get<{ jobs: Job[] }>('/jobs'),
        api.get<{ customers: Customer[] }>('/customers'),
      ]);
      setJobs(jobsRes.data.jobs || []);
      setCustomers(customersRes.data.customers || []);
    } catch (error) {
      setLoadFailed(true);
      toast.error(error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadDashboard(); }, [loadDashboard]);

  const dashboard = useMemo(() => {
    const today = startOfDay();
    const activeJobs = jobs.filter((job) => !isFinished(job));
    const happeningNow = activeJobs.filter((job) => {
      const start = parseDate(job.startDate);
      const end = parseDate(job.endDate);
      return start && startOfDay(start) <= today && (!end || startOfDay(end) >= today);
    });
    const overdue = activeJobs.filter((job) => {
      const end = parseDate(job.endDate);
      return end && startOfDay(end) < today;
    });
    const upcoming = activeJobs.filter((job) => {
      const start = parseDate(job.startDate);
      if (!start) return false;
      const difference = startOfDay(start).getTime() - today.getTime();
      return difference > 0 && difference <= 14 * DAY_IN_MS;
    });
    const monthValue = jobs
      .filter((job) => {
        const date = parseDate(job.startDate) || parseDate(job.created_at);
        return date && date.getMonth() === today.getMonth() && date.getFullYear() === today.getFullYear();
      })
      .reduce((sum, job) => sum + Number(job.final_revenue ?? job.revenue ?? 0), 0);
    const recentJobs = [...jobs]
      .sort((a, b) => {
        const aCreated = parseDate(a.created_at)?.getTime() ?? a.jobID;
        const bCreated = parseDate(b.created_at)?.getTime() ?? b.jobID;
        return bCreated - aCreated;
      })
      .slice(0, 6);
    const workQueue = [...overdue, ...happeningNow, ...recentJobs]
      .filter((job, index, allJobs) => allJobs.findIndex((candidate) => candidate.jobID === job.jobID) === index)
      .slice(0, 6);
    const schedule = [...activeJobs]
      .filter((job) => job.startDate || job.endDate)
      .sort((a, b) => {
        const aEnd = parseDate(a.endDate);
        const bEnd = parseDate(b.endDate);
        const aOverdue = Boolean(aEnd && startOfDay(aEnd).getTime() < today.getTime());
        const bOverdue = Boolean(bEnd && startOfDay(bEnd).getTime() < today.getTime());
        if (aOverdue !== bOverdue) return aOverdue ? -1 : 1;
        if (aOverdue && bOverdue) return bEnd!.getTime() - aEnd!.getTime();
        return (parseDate(a.startDate)?.getTime() ?? Number.MAX_SAFE_INTEGER)
          - (parseDate(b.startDate)?.getTime() ?? Number.MAX_SAFE_INTEGER);
      })
      .slice(0, 5);

    return { today, activeJobs, happeningNow, overdue, upcoming, monthValue, workQueue, schedule };
  }, [jobs]);

  const todayLabel = suiteDateLabel();

  const statCards = [
    {
      label: 'Aktive Jobs', value: dashboard.activeJobs.length, detail: `${jobs.length} insgesamt`,
      icon: Briefcase, iconClasses: 'bg-red-500/10 text-red-300', link: '/jobs',
    },
    {
      label: 'Heute im Einsatz', value: dashboard.happeningNow.length,
      detail: dashboard.overdue.length ? `${dashboard.overdue.length} überfällig` : 'Alles im Zeitplan',
      icon: CheckCircle2,
      iconClasses: dashboard.overdue.length ? 'bg-amber-500/10 text-amber-300' : 'bg-green-500/10 text-green-300',
      link: '/jobs',
    },
    {
      label: 'Nächste 14 Tage', value: dashboard.upcoming.length, detail: `${customers.length} Kontakte`,
      icon: CalendarDays, iconClasses: 'bg-blue-500/10 text-blue-300', link: '/jobs',
    },
    {
      label: 'Auftragswert im Monat', value: formatMoney(dashboard.monthValue), detail: 'nach Job-Start',
      icon: TrendingUp, iconClasses: 'bg-violet-500/10 text-violet-300', link: '/analytics',
    },
  ];

  return (
    <div className="suite-dashboard">
      <header className="suite-dashboard-header">
        <div className="suite-dashboard-heading">
            <p className="suite-dashboard-eyebrow">{todayLabel}</p>
            <h1 className="suite-dashboard-title">{suiteGreeting(user)}</h1>
            <p className="suite-dashboard-subtitle">
              {dashboard.happeningNow.length > 0
                ? `${dashboard.happeningNow.length} ${dashboard.happeningNow.length === 1 ? 'Job läuft' : 'Jobs laufen'} heute. Behalte Termine und neue Aufträge direkt im Blick.`
                : 'Heute läuft kein terminierter Job. Nutze die Übersicht für die nächsten Aufträge.'}
            </p>
        </div>
          <div className="suite-dashboard-actions">
            <Link to="/jobs/new" className="suite-button suite-button--primary">
              <Plus className="h-4 w-4" /> Neuer Job
            </Link>
            <Link to="/jobs" className="suite-button">
              <Search className="h-4 w-4" /> Jobs durchsuchen
            </Link>
          </div>
      </header>

      {loadFailed && !loading && (
        <div className="flex flex-col gap-3 rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-100 sm:flex-row sm:items-center sm:justify-between" role="alert">
          <div className="flex items-center gap-2"><AlertTriangle className="h-4 w-4" /> Dashboard-Daten konnten nicht geladen werden.</div>
          <button type="button" onClick={loadDashboard} className="inline-flex min-h-10 items-center justify-center gap-2 rounded-lg bg-white/10 px-3 font-medium hover:bg-white/15">
            <RefreshCw className="h-4 w-4" /> Erneut versuchen
          </button>
        </div>
      )}

      <section className="suite-kpi-grid">
        {statCards.map(({ label, value, detail, icon: Icon, iconClasses, link }) => (
          <Link key={label} to={link} className="group min-w-0 rounded-xl border border-white/10 bg-white/[0.035] p-4 transition hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.06] sm:p-5">
            {loading ? (
              <div className="animate-pulse space-y-4"><div className="h-9 w-9 rounded-lg bg-white/10" /><div className="h-7 w-2/3 rounded bg-white/10" /><div className="h-3 w-1/2 rounded bg-white/5" /></div>
            ) : (
              <>
                <div className="mb-5 flex items-start justify-between gap-2">
                  <span className="text-xs font-medium leading-5 text-gray-400 sm:text-sm">{label}</span>
                  <span className={`rounded-lg p-2 ${iconClasses}`}><Icon className="h-4 w-4" /></span>
                </div>
                <div className="truncate text-xl font-bold tabular-nums text-white sm:text-2xl">{value}</div>
                <div className="mt-1 flex items-center justify-between gap-2 text-xs text-gray-500">
                  <span className="truncate">{detail}</span><ArrowRight className="h-3.5 w-3.5 shrink-0 opacity-0 transition group-hover:translate-x-0.5 group-hover:opacity-100" />
                </div>
              </>
            )}
          </Link>
        ))}
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.65fr)_minmax(320px,0.85fr)]">
        <section className="overflow-hidden rounded-xl border border-white/10 bg-white/[0.025]">
          <div className="flex items-center justify-between border-b border-white/10 px-4 py-4 sm:px-5">
            <div>
              <h2 className="font-semibold text-white">Jetzt bearbeiten</h2>
              <p className="mt-0.5 text-xs text-gray-500">Überfällige, laufende und zuletzt angelegte Aufträge</p>
            </div>
            <Link to="/jobs" className="inline-flex min-h-10 items-center gap-1.5 rounded-lg px-2 text-sm font-medium text-gray-300 hover:bg-white/5 hover:text-white">
              Alle Jobs <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
          {loading ? (
            <div className="divide-y divide-white/5">{Array.from({ length: 5 }).map((_, index) => <div key={index} className="animate-pulse px-5 py-4"><div className="h-4 w-1/3 rounded bg-white/10" /><div className="mt-2 h-3 w-2/3 rounded bg-white/5" /></div>)}</div>
          ) : dashboard.workQueue.length === 0 ? (
            <div className="flex flex-col items-center px-6 py-14 text-center">
              <span className="mb-3 rounded-xl bg-white/5 p-3 text-gray-500"><Briefcase className="h-6 w-6" /></span>
              <p className="font-medium text-gray-300">Noch keine Jobs vorhanden</p>
              <p className="mt-1 text-sm text-gray-500">Lege den ersten Auftrag an, um hier zu starten.</p>
              <Link to="/jobs/new" className="mt-4 text-sm font-medium text-accent-red hover:text-red-300">Job anlegen</Link>
            </div>
          ) : (
            <div className="divide-y divide-white/5">
              {dashboard.workQueue.map((job) => (
                <Link key={job.jobID} to={`/jobs/${job.jobID}`} className="group flex min-w-0 items-center gap-3 px-4 py-3.5 transition hover:bg-white/[0.045] sm:px-5">
                  <span className="hidden h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white/5 text-gray-400 sm:flex"><Briefcase className="h-4 w-4" /></span>
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate font-medium text-white">{job.job_code}</span>
                      {job.status?.status && <span className={`hidden shrink-0 rounded-full border px-2 py-0.5 text-[11px] font-medium sm:inline ${statusClasses(job.status.status)}`}>{job.status.status}</span>}
                    </div>
                    <p className="mt-0.5 truncate text-sm text-gray-400">{customerName(job)}{job.description ? ` · ${job.description}` : ''}</p>
                  </div>
                  <div className="hidden shrink-0 text-right md:block">
                    <div className="text-sm text-gray-300">{formatDate(job.startDate)}</div>
                    <div className="mt-0.5 text-xs text-gray-500">{formatMoney(Number(job.final_revenue ?? job.revenue ?? 0))}</div>
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-gray-600 transition group-hover:translate-x-0.5 group-hover:text-gray-300" />
                </Link>
              ))}
            </div>
          )}
        </section>

        <div className="space-y-6">
          <section className="overflow-hidden rounded-xl border border-white/10 bg-white/[0.025]">
            <div className="border-b border-white/10 px-5 py-4">
              <h2 className="font-semibold text-white">Terminradar</h2>
              <p className="mt-0.5 text-xs text-gray-500">Laufende und nächste aktive Jobs</p>
            </div>
            {loading ? (
              <div className="space-y-3 p-5">{Array.from({ length: 4 }).map((_, index) => <div key={index} className="h-14 animate-pulse rounded-lg bg-white/5" />)}</div>
            ) : dashboard.schedule.length === 0 ? (
              <div className="px-5 py-10 text-center text-sm text-gray-500"><CalendarDays className="mx-auto mb-2 h-6 w-6 opacity-50" />Keine aktiven Termine</div>
            ) : (
              <div className="divide-y divide-white/5">
                {dashboard.schedule.map((job) => {
                  const timing = scheduleLabel(job, dashboard.today);
                  const scheduleDate = job.startDate || job.endDate;
                  return (
                    <Link key={job.jobID} to={`/jobs/${job.jobID}`} className="group flex items-center gap-3 px-5 py-3.5 transition hover:bg-white/[0.045]">
                      <span className="flex h-10 w-10 shrink-0 flex-col items-center justify-center rounded-lg bg-white/5 text-center">
                        <span className="text-[10px] uppercase leading-none text-gray-500">{formatDate(scheduleDate, { month: 'short' }).replace('.', '')}</span>
                        <span className="mt-0.5 text-sm font-semibold leading-none text-gray-200">{formatDate(scheduleDate, { day: '2-digit' })}</span>
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-gray-200 group-hover:text-white">{job.job_code}</p>
                        <p className="mt-0.5 truncate text-xs text-gray-500">{customerName(job)}</p>
                      </div>
                      <span className={`shrink-0 rounded-md px-2 py-1 text-[11px] font-medium ${timing.classes}`}>{timing.text}</span>
                    </Link>
                  );
                })}
              </div>
            )}
          </section>

          <section className="rounded-xl border border-white/10 bg-white/[0.025] p-4">
            <h2 className="px-1 pb-3 text-sm font-semibold text-white">Schnellstart</h2>
            <div className="grid grid-cols-3 gap-2">
              {[
                { to: '/analytics', label: 'Analyse', icon: BarChart3 },
                { to: '/documents', label: 'Dokumente', icon: FileText },
                { to: '/employees', label: 'Team', icon: Users },
              ].map(({ to, label, icon: Icon }) => (
                <Link key={to} to={to} className="flex min-h-20 flex-col items-center justify-center gap-2 rounded-lg border border-transparent bg-white/[0.035] text-xs font-medium text-gray-400 transition hover:border-white/10 hover:bg-white/[0.07] hover:text-white">
                  <Icon className="h-4 w-4 text-gray-300" />{label}
                </Link>
              ))}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
