import { useEffect, useState, useCallback, useRef } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import {
  Plus, Search, RefreshCw, Briefcase, Calendar, User,
  ArrowLeft, Trash2, Edit3, X, Check, Monitor, Package,
  ChevronRight, ChevronDown, FileText, Upload,
} from 'lucide-react';
import { jobsApi, customersApi, statusApi, api, jobEmployeesApi, employeesApi, venuesApi } from '../lib/api';
import type { Job, Customer, JobStatus, JobDevice, JobEmployee, Employee, Venue, JobTotals } from '../lib/api';
import MappingModal from '../components/MappingModal';
import type { MappedItem, ExtractionMeta } from '../components/MappingModal';
import JobPositionsPanel from '../components/JobPositionsPanel';
import JobActivityPanel from '../components/JobActivityPanel';
import { toast } from '../lib/toast';
import { appPath } from '../lib/app-paths';
import './jobs-page.css';

function statusColor(status: string) {
  const s = status.toLowerCase();
  if (s.includes('bestätigt') || s.includes('bestaetigt') || s.includes('confirmed')) return 'jobs-status--confirmed';
  if (s.includes('abgeschlossen') || s.includes('completed')) return 'jobs-status--completed';
  if (s.includes('storniert') || s.includes('cancelled') || s.includes('canceled')) return 'jobs-status--cancelled';
  return 'jobs-status--planning';
}

function jobStatusId(job: Job): number {
  return job.status_id ?? (job as Job & { statusID?: number }).statusID ?? 1;
}

function customerName(c?: Customer | null) {
  if (!c) return '—';
  if (c.companyname) return c.companyname;
  return `${c.firstname || ''} ${c.lastname || ''}`.trim() || '—';
}

function formatDate(d?: string | null) {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('de-DE');
}

// ── Product Tree (availability) ───────────────────────────────

interface ProductNode {
  id: number;
  name: string;
  available_count?: number;
  device_count?: number;
  is_consumable?: boolean;
  stock_quantity?: number;
}
interface TreeNode {
  id: number;
  name: string;
  products?: ProductNode[];
  subcategories?: TreeNode[];
  subbiercategories?: TreeNode[];
}

function ProductPicker({
  startDate, endDate, jobId, onSelect,
}: {
  startDate: string; endDate: string; jobId?: number;
  onSelect: (productId: number, name: string, qty: number) => void;
}) {
  const [tree, setTree] = useState<TreeNode[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [qty, setQty] = useState<Record<number, number>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!startDate || !endDate) return;
    setLoading(true);
    let url = `/devices/tree/availability?start_date=${startDate}&end_date=${endDate}`;
    if (jobId) url += `&job_id=${jobId}`;
    api.get(url).then((r) => setTree(r.data.treeData || [])).catch((e: any) => toast.error(e)).finally(() => setLoading(false));
  }, [startDate, endDate, jobId]);

  const toggle = (id: string) =>
    setExpanded((p) => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });

  const renderProducts = (products: ProductNode[]) =>
    products.map((p) => {
      const avail = p.available_count ?? 0;
      const total = p.device_count ?? 0;
      const cur = qty[p.id] ?? 0;
      return (
        <div key={p.id} className="flex items-center justify-between px-4 py-2 hover:bg-white/5 rounded-lg">
          <div className="flex items-center gap-2 min-w-0">
            <Package className="w-3.5 h-3.5 text-gray-500 flex-shrink-0" />
            <span className="text-sm text-white truncate">{p.name}</span>
            <span className={`text-xs px-1.5 py-0.5 rounded-full ${avail > 0 ? 'bg-green-500/10 text-green-400' : total === 0 ? 'bg-gray-500/10 text-gray-400' : 'bg-red-500/10 text-red-400'}`}>
              {avail}/{total} frei
            </span>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            <input
              type="number" min={0} value={cur}
              onChange={(e) => setQty((q) => ({ ...q, [p.id]: Math.max(0, Number(e.target.value)) }))}
              className="w-16 px-2 py-1 text-center text-sm bg-white/5 border border-white/10 rounded-lg focus:outline-none focus:border-accent-red"
            />
            <button
              onClick={() => { if (cur > 0) onSelect(p.id, p.name, cur); }}
              disabled={cur === 0 || total === 0}
              className="px-3 py-1 text-xs bg-accent-red/80 hover:bg-accent-red disabled:opacity-30 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
            >
              +
            </button>
          </div>
        </div>
      );
    });

  const renderNode = (node: TreeNode, prefix: string): React.ReactNode => {
    const key = `${prefix}-${node.id}`;
    const open = expanded.has(key);
    const children = [
      ...(node.subcategories || []).map((s) => renderNode(s, `sub-${key}`)),
      ...(node.subbiercategories || []).map((s) => renderNode(s, `subbier-${key}`)),
      ...(open && node.products ? renderProducts(node.products) : []),
    ];
    return (
      <div key={key}>
        <button
          onClick={() => toggle(key)}
          className="w-full flex items-center gap-2 px-3 py-2 hover:bg-white/5 rounded-lg text-left transition-colors"
        >
          {open ? <ChevronDown className="w-4 h-4 text-gray-400 flex-shrink-0" /> : <ChevronRight className="w-4 h-4 text-gray-400 flex-shrink-0" />}
          <span className="text-sm font-medium text-gray-200">{node.name}</span>
        </button>
        {open && <div className="ml-4 border-l border-white/5 pl-2">{children}</div>}
      </div>
    );
  };

  if (!startDate || !endDate) return (
    <p className="text-gray-500 text-sm text-center py-4">Bitte zuerst Start- und Enddatum setzen.</p>
  );
  if (loading) return <div className="flex justify-center py-6"><div className="w-6 h-6 border-2 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;
  if (!tree.length) return <p className="text-gray-500 text-sm text-center py-4">Keine Produkte verfügbar.</p>;

  return <div className="space-y-1 max-h-72 overflow-y-auto">{tree.map((n) => renderNode(n, 'cat'))}</div>;
}

// ── Device List (grouped by product) ─────────────────────────

export function DeviceList({ devices, jobId, onChanged }: { devices: JobDevice[]; jobId?: number; onChanged?: () => void }) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [removing, setRemoving] = useState<string | null>(null);

  const groups = devices.reduce<Record<string, { name: string; devices: JobDevice[] }>>((acc, d) => {
    const name = d.device?.product?.name || 'Unbekanntes Produkt';
    if (!acc[name]) acc[name] = { name, devices: [] };
    acc[name].devices.push(d);
    return acc;
  }, {});

  const toggle = (name: string) =>
    setExpanded((prev) => { const next = new Set(prev); next.has(name) ? next.delete(name) : next.add(name); return next; });

  const removeDevice = async (deviceId: string) => {
    if (!jobId) return;
    setRemoving(deviceId);
    try {
      await api.delete(`/jobs/${jobId}/devices/${deviceId}`);
      onChanged?.();
    } catch (e) {
      toast.error(e);
    } finally {
      setRemoving(null);
    }
  };

  return (
    <div className="glass-dark rounded-xl border border-white/10 p-5">
      <div className="flex items-center gap-2 mb-4">
        <Monitor className="w-4 h-4 text-accent-red" />
        <h3 className="font-semibold text-white">Geräte ({devices.length})</h3>
      </div>
      {devices.length === 0 ? (
        <p className="text-gray-500 text-sm">Keine Geräte zugeordnet</p>
      ) : (
        <div className="space-y-1">
          {Object.values(groups).map(({ name, devices: grpDevices }) => (
            <div key={name} className="rounded-lg overflow-hidden border border-white/5">
              <button
                onClick={() => toggle(name)}
                className="w-full flex items-center justify-between px-4 py-3 hover:bg-white/5 transition-colors text-left"
              >
                <div className="flex items-center gap-3">
                  <Monitor className="w-4 h-4 text-gray-400" />
                  <span className="text-white text-sm font-medium">{name}</span>
                  <span className="text-xs text-gray-400 bg-white/10 px-2 py-0.5 rounded-full">{grpDevices.length}×</span>
                </div>
                <span className="text-gray-500 text-xs">{expanded.has(name) ? '▲' : '▼'}</span>
              </button>
              {expanded.has(name) && (
                <div className="border-t border-white/5 px-4 py-2 bg-white/2">
                  {grpDevices.map((d) => (
                    <div key={d.deviceID} className="flex items-center justify-between py-1.5 text-xs text-gray-400">
                      <span className="font-mono">{d.device?.serialnumber || d.deviceID}</span>
                      <div className="flex items-center gap-3">
                        <span>
                          {d.custom_price != null
                            ? `€${Number(d.custom_price).toLocaleString('de-DE', { minimumFractionDigits: 2 })}/Tag`
                            : d.device?.product?.itemcostperday != null
                              ? `€${Number(d.device.product.itemcostperday).toLocaleString('de-DE', { minimumFractionDigits: 2 })}/Tag`
                              : ''}
                        </span>
                        {jobId && onChanged && (
                          <button
                            onClick={() => removeDevice(d.deviceID)}
                            disabled={removing === d.deviceID}
                            className="p-1 hover:bg-red-500/20 text-red-400 rounded transition-colors disabled:opacity-40"
                            title="Gerät entfernen"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── Requirements Panel (Stage 2 device assignment) ────────────

export function RequirementsPanel({ jobId, devices, onDeviceAssigned }: { jobId: number; devices: JobDevice[]; onDeviceAssigned: () => void }) {
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [assigning, setAssigning] = useState<number | null>(null);
  const [availableDevices, setAvailableDevices] = useState<AvailableDevice[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [assignError, setAssignError] = useState('');
  const [removingDevice, setRemovingDevice] = useState<string | null>(null);

  const load = useCallback(() => {
    api.get(`/jobs/${jobId}/requirements`)
      .then((r) => setRequirements(r.data.requirements || []))
      .catch((e: any) => toast.error(e));
  }, [jobId]);

  useEffect(() => { load(); }, [load]);

  const openAssignModal = async (productId: number) => {
    setAssigning(productId);
    setAssignError('');
    setLoadingDevices(true);
    try {
      const r = await api.get(`/jobs/${jobId}/products/${productId}/available-devices`);
      setAvailableDevices(r.data.devices || []);
    } catch {
      setAssignError('Fehler beim Laden der verfügbaren Geräte');
    } finally {
      setLoadingDevices(false);
    }
  };

  const assignDevice = async (deviceId: string) => {
    try {
      await api.post(`/jobs/${jobId}/devices/${deviceId}`, {});
      load();
      onDeviceAssigned();
      // Reload device list in-place so user can keep assigning without reopening
      if (assigning !== null) {
        const r = await api.get(`/jobs/${jobId}/products/${assigning}/available-devices`);
        setAvailableDevices(r.data.devices || []);
      }
      setAssignError('');
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setAssignError(err.response?.data?.error || 'Zuweisung fehlgeschlagen');
    }
  };

  const removeDevice = async (deviceId: string) => {
    setRemovingDevice(deviceId);
    try {
      await api.delete(`/jobs/${jobId}/devices/${deviceId}`);
      load();
      onDeviceAssigned();
    } catch (e) {
      toast.error(e);
    } finally {
      setRemovingDevice(null);
    }
  };

  const unplannedDevices = devices.filter((device) => !requirements.some((req) => req.product_id === device.device?.product?.productID));

  return (
    <section className="jobs-card jobs-card-body">
      <div className="flex items-center gap-2 mb-2">
        <Package className="w-4 h-4 text-accent-red" />
        <h3 className="font-semibold text-white">Material und Geräte</h3>
      </div>
      <p className="jobs-form-note mb-4">Bedarf aus Auftragspositionen und zusätzlicher Materialplanung. Zugewiesene Einzelgeräte stehen direkt beim Produkt.</p>
      {requirements.length === 0 && <p className="jobs-form-note rounded-lg border border-[var(--border-default)] p-4">Noch kein Produktbedarf vorhanden. Ergänze eine Produktposition oder plane zusätzliches Material.</p>}
      <div className="space-y-2">
        {requirements.map((req) => {
          const done = req.assigned_count >= req.quantity;
          const productName = req.product?.name || `Produkt ${req.product_id}`;
          const assignedDevices = devices.filter((device) => device.device?.product?.productID === req.product_id);
          return (
            <div key={req.id} className="rounded-lg border border-[var(--border-default)] bg-[var(--surface-2)] p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0">
                  <strong className="block text-sm">{productName}</strong>
                  <span className="text-xs text-[var(--text-secondary)]">{req.position_quantity || 0} aus Positionen · {req.manual_quantity || 0} zusätzlich geplant</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold">{req.assigned_count} von {req.quantity} zugewiesen · {done ? 'gedeckt' : 'offen'}</span>
                  {!done && <button onClick={() => openAssignModal(req.product_id)} className="suite-button">Gerät zuweisen</button>}
                </div>
              </div>
              {assignedDevices.length > 0 && <div className="mt-3 border-t border-[var(--border-divider)] pt-3 space-y-2">
                {assignedDevices.map((device) => <div key={device.deviceID} className="flex items-center justify-between gap-2 text-sm">
                  <span>{device.device?.serialnumber || device.deviceID}</span>
                  <button className="suite-button" disabled={removingDevice === device.deviceID} onClick={() => removeDevice(device.deviceID)} aria-label={`Gerät ${device.deviceID} entfernen`}><X className="w-4 h-4" /></button>
                </div>)}
              </div>}
              </div>
          );
        })}
      </div>

      {unplannedDevices.length > 0 && <div className="mt-4 border-t border-[var(--border-divider)] pt-4">
        <h4 className="font-semibold text-sm mb-2">Geräte ohne Produktbedarf</h4>
        {unplannedDevices.map((device) => <div key={device.deviceID} className="flex items-center justify-between gap-2 py-1 text-sm">
          <span>{device.device?.product?.name || 'Unbekanntes Produkt'} · {device.device?.serialnumber || device.deviceID}</span>
          <button className="suite-button" disabled={removingDevice === device.deviceID} onClick={() => removeDevice(device.deviceID)} aria-label={`Gerät ${device.deviceID} entfernen`}><X className="w-4 h-4" /></button>
        </div>)}
      </div>}

      {assigning !== null && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="glass-dark rounded-2xl border border-white/10 p-6 w-full max-w-md space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">Gerät zuweisen</h3>
              <button
                onClick={() => { setAssigning(null); setAssignError(''); }}
                className="p-1.5 hover:bg-white/10 rounded-lg"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            {assignError && <p className="text-sm text-red-400">{assignError}</p>}
            {loadingDevices ? (
              <div className="flex justify-center py-6">
                <div className="w-6 h-6 border-2 border-accent-red/20 border-t-accent-red rounded-full animate-spin" />
              </div>
            ) : availableDevices.length === 0 ? (
              <p className="text-sm text-gray-500 text-center py-4">
                Keine Geräte für dieses Produkt vorhanden.
              </p>
            ) : (
              <div className="space-y-1 max-h-64 overflow-y-auto">
                {availableDevices.map((d) => (
                  <button
                    key={d.DeviceID}
                    onClick={() => d.Available && assignDevice(d.DeviceID)}
                    disabled={!d.Available}
                    className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-left transition-colors ${
                      d.Available
                        ? 'hover:bg-white/10 cursor-pointer'
                        : 'opacity-40 cursor-not-allowed'
                    }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${d.Available ? 'bg-green-400' : 'bg-red-400'}`} />
                      <span className={`text-sm font-mono ${d.Available ? 'text-white' : 'text-gray-500 line-through'}`}>{d.DeviceID}</span>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {!d.Available && (
                        <span className="text-xs text-red-400">
                          {d.AssignedToJob ? 'Bereits zugewiesen' : 'Nicht verfügbar'}
                        </span>
                      )}
                      <span className="text-xs text-gray-400">{d.CaseName || 'Lose'}</span>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

// ── Job Detail ────────────────────────────────────────────────

function JobDetail({ id, onBack }: { id: number; onBack: () => void }) {
  const [job, setJob] = useState<Job | null>(null);
  const [devices, setDevices] = useState<JobDevice[]>([]);
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [totals, setTotals] = useState<JobTotals | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [changingStatus, setChangingStatus] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const loadData = useCallback(async () => {
    setLoading(true);
    const [jRes, dRes, rRes, tRes] = await Promise.allSettled([
      jobsApi.getById(id),
      jobsApi.getDevices(id),
      api.get(`/jobs/${id}/requirements`),
      api.get<JobTotals>(`/jobs/${id}/totals`),
    ]);
    if (jRes.status === 'fulfilled') setJob(jRes.value.data);
    else setJob(null);
    setDevices(dRes.status === 'fulfilled' ? dRes.value.data.devices || [] : []);
    setRequirements(rRes.status === 'fulfilled' ? rRes.value.data.requirements || [] : []);
    setTotals(tRes.status === 'fulfilled' ? tRes.value.data : null);
    setError([jRes, dRes, rRes, tRes].some((result) => result.status === 'rejected')
      ? 'Einige Jobdaten konnten nicht geladen werden. Bitte erneut versuchen.'
      : '');
    setLoading(false);
  }, [id]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleDelete = async () => {
    if (!confirm('Job archivieren? Er verschwindet aus der aktiven Jobliste.')) return;
    setDeleting(true);
    try {
      await jobsApi.delete(id);
      onBack();
    } catch (e) {
      const response = e as { response?: { data?: { error?: string } } };
      setError(response.response?.data?.error || 'Archivieren fehlgeschlagen.');
    } finally {
      setDeleting(false);
    }
  };

  const changeStatus = async (nextStatus: number) => {
    if ((nextStatus === 4 || nextStatus === 6) && !confirm(nextStatus === 4 ? 'Job abschließen?' : 'Job stornieren?')) return;
    setChangingStatus(true);
    setError('');
    try {
      await api.put(`/jobs/${id}`, { status_id: nextStatus, revision: job?.revision });
      loadData();
    } catch (e) {
      const response = e as { response?: { data?: { error?: string } } };
      setError(response.response?.data?.error || 'Statuswechsel fehlgeschlagen.');
    } finally {
      setChangingStatus(false);
    }
  };

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;
  if (!job) return <div className="text-center text-gray-400 py-20">Job nicht gefunden.</div>;

  const fulfilled = requirements.filter((req) => req.assigned_count >= req.quantity).length;
  const statusId = jobStatusId(job);
  const statusActions = statusId === 1
    ? [{ id: 2, label: 'Job bestätigen' }, { id: 6, label: 'Stornieren' }]
    : statusId === 2
      ? [{ id: 4, label: 'Abschließen' }, { id: 1, label: 'Zurück in Planung' }, { id: 6, label: 'Stornieren' }]
      : [{ id: 1, label: 'Wieder öffnen' }];

  return (
    <div className="jobs-workspace">
      <header className="jobs-header">
        <div className="jobs-header-main">
          <button className="suite-button mb-4" onClick={onBack}><ArrowLeft className="w-4 h-4" /> Alle Jobs</button>
          <span className="jobs-eyebrow">{job.job_code} · {job.status?.status || 'Planung'}</span>
          <h1 className="jobs-title">{job.description || job.job_code}</h1>
          <p className="jobs-subtitle">{customerName(job.customer)} · {formatDate(job.startDate)} bis {formatDate(job.endDate)}</p>
        </div>
        <div className="jobs-header-actions">
          <button onClick={() => navigate(`/jobs/${id}/edit`)} className="suite-button"><Edit3 className="w-4 h-4" /> Stammdaten bearbeiten</button>
          <button onClick={handleDelete} disabled={deleting} className="suite-button"><Trash2 className="w-4 h-4" /> Archivieren</button>
        </div>
      </header>

      {error && <div className="jobs-inline-alert" role="alert">{error} <button className="underline ml-2" onClick={loadData}>Erneut versuchen</button></div>}

      <div className="jobs-metrics">
        <div className="jobs-card jobs-metric"><span className="jobs-metric-label">Status</span><span className="jobs-metric-value">{job.status?.status || 'Planung'}</span><span className="jobs-metric-caption">Job-Lebenszyklus</span></div>
        <div className="jobs-card jobs-metric"><span className="jobs-metric-label">Zeitraum</span><span className="jobs-metric-value">{formatDate(job.startDate)} – {formatDate(job.endDate)}</span><span className="jobs-metric-caption">Veranstaltung</span></div>
        <div className="jobs-card jobs-metric"><span className="jobs-metric-label">Material gedeckt</span><span className="jobs-metric-value">{fulfilled} / {requirements.length}</span><span className="jobs-metric-caption">Produktbedarfe mit Gerätezuweisung</span></div>
        <div className="jobs-card jobs-metric"><span className="jobs-metric-label">Auftragswert brutto</span><span className="jobs-metric-value">{totals ? totals.brutto.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' }) : '—'}</span><span className="jobs-metric-caption">Nach Rabatt und Steuer</span></div>
      </div>

      <div className="jobs-detail-grid">
        <div className="jobs-section-stack">
          <JobPositionsPanel jobId={id} onChanged={loadData} />
          <RequirementsPanel jobId={id} devices={devices} onDeviceAssigned={loadData} />
          <JobEmployeesPanel jobId={id} />
        </div>
        <aside className="jobs-section-stack">
          <section className="jobs-card">
            <div className="jobs-card-heading"><h2>Nächster Schritt</h2></div>
            <div className="jobs-card-body">
              <p className="jobs-form-note mb-4">{statusId === 1 ? 'Prüfe Daten, Positionen und Materialbedarf vor der Freigabe.' : statusId === 2 ? 'Der Job ist für die Vorbereitung im Warehouse freigegeben.' : 'Der Job ist geschlossen. Du kannst ihn bei Bedarf wieder in Planung nehmen.'}</p>
              <div className="flex flex-wrap gap-2">
                {statusActions.map((action, index) => <button key={action.id} className={`suite-button ${index === 0 ? 'suite-button--primary' : ''}`} disabled={changingStatus} onClick={() => changeStatus(action.id)}>{action.label}</button>)}
              </div>
            </div>
          </section>
          <section className="jobs-card">
            <div className="jobs-card-heading"><h2>Auftragsdaten</h2></div>
            <div className="jobs-card-body space-y-3">
              <InfoRow label="Jobnummer" value={job.job_code} />
              <InfoRow label="Kunde" value={customerName(job.customer)} />
              <InfoRow label="Veranstaltungsort" value={job.venue?.name || 'Nicht angegeben'} />
              {job.customer?.email && <InfoRow label="E-Mail" value={job.customer.email} />}
              {job.customer?.phonenumber && <InfoRow label="Telefon" value={job.customer.phonenumber} />}
            </div>
          </section>
          <JobActivityPanel jobId={id} />
        </aside>
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between items-center text-sm">
      <span className="text-[var(--text-secondary)]">{label}</span>
      <span className="text-[var(--text-primary)] font-medium">{value}</span>
    </div>
  );
}

// ── Selected Products Summary ─────────────────────────────────

interface ProductSelection {
  product_id: number;
  name: string;
  quantity: number;
}

interface Requirement {
  id: number;
  job_id: number;
  product_id: number;
  quantity: number;
  manual_quantity: number;
  position_quantity: number;
  assigned_count: number;
  product?: { name: string; productid: number };
}

interface AvailableDevice {
  DeviceID: string;
  ProductID: number;
  Status: string;
  CaseID?: number;
  CaseName?: string;
  AssignedToJob: boolean;
  Available: boolean;
}

function SelectedProductsSummary({
  selections, onChange,
}: {
  selections: ProductSelection[];
  onChange: (s: ProductSelection[]) => void;
}) {
  if (selections.length === 0) return (
    <p className="text-gray-500 text-sm">Noch keine Produkte ausgewählt.</p>
  );
  return (
    <div className="space-y-1">
      {selections.map((s) => (
        <div key={s.product_id} className="flex items-center justify-between px-3 py-2 bg-white/5 rounded-lg">
          <span className="text-sm text-gray-200">{s.name}</span>
          <div className="flex items-center gap-2">
            <input
              type="number" min={1} value={s.quantity}
              onChange={(e) => {
                const qty = Math.max(1, Number(e.target.value));
                onChange(selections.map((x) => x.product_id === s.product_id ? { ...x, quantity: qty } : x));
              }}
              className="w-16 px-2 py-1 text-center text-sm bg-white/5 border border-white/10 rounded-lg focus:outline-none focus:border-accent-red"
            />
            <button
              onClick={() => onChange(selections.filter((x) => x.product_id !== s.product_id))}
              className="p-1 hover:bg-red-500/20 text-red-400 rounded transition-colors"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── PDF Upload ────────────────────────────────────────────────

function PdfImportBanner({ onUploadReady }: {
  onUploadReady: (uploadId: number) => void;
}) {
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const upload = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) { setError('Bitte eine PDF-Datei auswählen.'); return; }
    setError(''); setStatus('Wird hochgeladen…');

    const fd = new FormData();
    fd.append('pdf', file);
    try {
      const up = await fetch(appPath('/api/pdf/upload'), { method: 'POST', body: fd, credentials: 'include' });
      const upData = await up.json();
      if (!up.ok || !upData.upload_id) throw new Error(upData.error || 'Upload fehlgeschlagen');
      setOpen(false);
      setStatus('');
      onUploadReady(upData.upload_id);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fehler beim Import');
      setStatus('');
    }
  };

  return (
    <>
      <div className="flex items-center justify-between p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
        <div className="flex items-center gap-3">
          <FileText className="w-4 h-4 text-blue-400" />
          <div>
            <p className="text-sm font-medium text-blue-300">Angebot / Rechnung als PDF?</p>
            <p className="text-xs text-gray-500">Daten automatisch per OCR einlesen</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="flex items-center gap-2 px-3 py-1.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-300 rounded-lg text-sm transition-colors"
        >
          <Upload className="w-3.5 h-3.5" /> PDF hochladen
        </button>
      </div>

      {open && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="glass-dark rounded-2xl border border-white/10 p-6 w-full max-w-md space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">PDF-Import (OCR)</h3>
              <button onClick={() => { setOpen(false); setStatus(''); setError(''); }} className="p-1.5 hover:bg-white/10 rounded-lg">
                <X className="w-4 h-4" />
              </button>
            </div>
            <p className="text-sm text-gray-400">Lade ein Angebot oder eine Rechnung als PDF hoch. Kunde, Datum und Produkte werden automatisch erkannt.</p>
            <input ref={fileRef} type="file" accept="application/pdf" className="w-full text-sm text-gray-300 file:mr-3 file:px-3 file:py-1.5 file:rounded-lg file:border-0 file:bg-white/10 file:text-white hover:file:bg-white/15 cursor-pointer" />
            {error && <p className="text-sm text-red-400">{error}</p>}
            {status && <p className="text-sm text-green-400">{status}</p>}
            <div className="flex gap-3 pt-1">
              <button type="button" onClick={upload} className="flex items-center gap-2 px-4 py-2 bg-accent-red hover:bg-accent-red/80 text-white rounded-lg text-sm font-medium transition-colors">
                <Upload className="w-4 h-4" /> Importieren
              </button>
              <button type="button" onClick={() => { setOpen(false); setStatus(''); setError(''); }} className="px-4 py-2 bg-white/10 hover:bg-white/15 rounded-lg text-sm transition-colors">
                Abbrechen
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// ── Job Form ──────────────────────────────────────────────────

function JobForm({ jobId, onSaved, onCancel }: { jobId?: number; onSaved: (id: number) => void; onCancel: () => void }) {
  const [form, setForm] = useState<Partial<Job>>({});
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [statuses, setStatuses] = useState<JobStatus[]>([]);
  const [venues, setVenues] = useState<Venue[]>([]);
  const [selections, setSelections] = useState<ProductSelection[]>([]);
  const [showPicker, setShowPicker] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [pendingUploadId, setPendingUploadId] = useState<number | null>(null);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      customersApi.getAll(),
      statusApi.getAll(),
      venuesApi.list(),
      jobId ? jobsApi.getById(jobId) : Promise.resolve(null),
      jobId ? api.get(`/jobs/${jobId}/requirements`) : Promise.resolve(null),
    ]).then(([cRes, sRes, vRes, jRes, rRes]) => {
      setCustomers(cRes.data.customers || []);
      setStatuses(sRes.data.statuses || []);
      setVenues(vRes.data || []);
      if (jRes) setForm(jRes.data);
      if (rRes) {
        const reqs: Requirement[] = rRes.data.requirements || [];
        setSelections(reqs.filter((r) => r.manual_quantity > 0).map((r) => ({
          product_id: r.product_id,
          name: r.product?.name || `Produkt #${r.product_id}`,
          quantity: r.manual_quantity,
        })));
      }
    }).catch((e: any) => toast.error(e)).finally(() => setLoading(false));
  }, [jobId]);

  const isValidDateStr = (s?: string) => !!s && /^\d{4}-\d{2}-\d{2}$/.test(s);

  const handleMappingComplete = (items: MappedItem[], meta: ExtractionMeta) => {
    setPendingUploadId(null);
    if (meta.job_id) {
      onSaved(meta.job_id);
      return;
    }
    if (meta.customer_id || isValidDateStr(meta.start_date) || isValidDateStr(meta.end_date)) {
      setForm(f => ({
        ...f,
        ...(meta.customer_id ? { customer_id: meta.customer_id } : {}),
        ...(isValidDateStr(meta.start_date) ? { startDate: meta.start_date } : {}),
        ...(isValidDateStr(meta.end_date) ? { endDate: meta.end_date } : {}),
      }));
    }
    if (items.length > 0) {
      setSelections((prev) => {
        const merged = [...prev];
        items.forEach((item) => {
          const idx = merged.findIndex((x) => x.product_id === item.product_id);
          if (idx >= 0) merged[idx] = { ...merged[idx], quantity: merged[idx].quantity + item.quantity };
          else merged.push({ product_id: item.product_id, name: item.name, quantity: item.quantity });
        });
        return merged;
      });
    }
  };

  const addProduct = (productId: number, name: string, qty: number) => {
    setSelections((prev) => {
      const idx = prev.findIndex((x) => x.product_id === productId);
      if (idx >= 0) return prev.map((x, i) => i === idx ? { ...x, quantity: x.quantity + qty } : x);
      return [...prev, { product_id: productId, name, quantity: qty }];
    });
    setShowPicker(false);
  };

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        customer_id: form.customer_id,
        status_id: jobId ? form.status_id : 1,
        description: form.description?.trim(),
        startDate,
        endDate,
        venue_id: form.venue_id ?? null,
        selected_products: selections.map((s) => ({ product_id: s.product_id, quantity: s.quantity })),
      };
      if (jobId) {
        payload.revision = form.revision;
        await api.put(`/jobs/${jobId}`, payload);
        onSaved(jobId);
      } else {
        const res = await api.post<{ jobID: number; job_code: string }>('/jobs', payload);
        onSaved(res.data.jobID);
      }
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string };
      setError(axiosErr.response?.data?.error || axiosErr.message || 'Speichern fehlgeschlagen');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;

  const startDate = form.startDate?.slice(0, 10) || '';
  const endDate = form.endDate?.slice(0, 10) || '';

  return (
    <div className="jobs-workspace">
      <header className="jobs-header">
        <div className="jobs-header-main">
          <button onClick={onCancel} className="suite-button mb-4"><ArrowLeft className="w-4 h-4" /> {jobId ? 'Zum Job' : 'Alle Jobs'}</button>
          <span className="jobs-eyebrow">{jobId ? 'Stammdaten' : 'Neuer Auftrag'}</span>
          <h1 className="jobs-title">{jobId ? 'Job bearbeiten' : 'Job anlegen'}</h1>
          <p className="jobs-subtitle">{jobId ? 'Kunde, Ort, Titel und Zeitraum dieses Jobs ändern.' : 'Beginne mit den Stammdaten. Positionen, Personal und Geräte ergänzt du danach im Job.'}</p>
        </div>
      </header>

      {/* PDF import — only for new jobs */}
      {!jobId && <PdfImportBanner onUploadReady={setPendingUploadId} />}

      {pendingUploadId !== null && (
        <MappingModal
          uploadId={pendingUploadId}
          onComplete={(items: MappedItem[], meta: ExtractionMeta) => handleMappingComplete(items, meta)}
          onClose={() => setPendingUploadId(null)}
        />
      )}

      <div className="jobs-form-grid">
      <section className="jobs-card jobs-card-body">
        {error && <div className="jobs-inline-alert" role="alert">{error}</div>}
        <form onSubmit={save} className="space-y-5">
          {/* Customer */}
          <div>
            <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1.5" htmlFor="job-customer">Kunde *</label>
            <select
              id="job-customer"
              value={form.customer_id || ''}
              onChange={(e) => setForm({ ...form, customer_id: Number(e.target.value) })}
              required
              className="w-full px-3 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white focus:outline-none focus:border-accent-red"
            >
              <option value="">— Kunde wählen —</option>
              {customers.map((c) => (
                <option key={c.customer_id} value={c.customer_id}>{customerName(c)}</option>
              ))}
            </select>
          </div>

          <div className="rounded-lg border border-[var(--border-default)] bg-[var(--surface-2)] px-4 py-3 text-sm">
            <span className="font-semibold">Status: {jobId ? statuses.find((s) => s.status_id === form.status_id)?.status || 'Lädt…' : 'Planung'}</span>
            <p className="mt-1 text-[var(--text-secondary)]">Statuswechsel erfolgen nach dem Speichern in der Jobübersicht.</p>
          </div>

          {/* Venue */}
          <div>
            <label className="block text-sm font-medium text-[var(--text-secondary)] mb-1.5" htmlFor="job-venue">Veranstaltungsort</label>
            <select
              id="job-venue"
              value={form.venue_id ?? ''}
              onChange={(e) => setForm({ ...form, venue_id: e.target.value ? Number(e.target.value) : null })}
              className="w-full px-3 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white focus:outline-none focus:border-accent-red"
            >
              <option value="">— Kein Veranstaltungsort —</option>
              {venues.map((v) => (
                <option key={v.id} value={v.id}>{v.name}{v.city ? ` (${v.city})` : ''}</option>
              ))}
            </select>
            <p className="text-xs text-gray-500 mt-1">
              {(() => {
                if (form.venue_id) {
                  const v = venues.find(x => x.id === form.venue_id);
                  if (v) return `Kalender-Ort: ${v.name}${v.city ? `, ${v.city}` : ''}`;
                }
                const c = customers.find(x => x.customer_id === form.customer_id);
                if (c && (c.city || c.street)) return `Kalender-Ort: Kundenadresse${c.city ? ` (${c.city})` : ''}`;
                return 'Kein Ort im Kalender';
              })()}
            </p>
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5" htmlFor="job-title">Jobtitel *</label>
            <input
              id="job-title"
              type="text"
              value={form.description || ''}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="z. B. Sommerfest auf dem Marktplatz"
              required
              className="w-full px-3 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white placeholder-gray-600 focus:outline-none focus:border-accent-red"
            />
          </div>

          {/* Dates */}
          <div className="grid grid-cols-2 gap-4">
            <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5" htmlFor="job-start-date">Startdatum</label>
                <input
                  id="job-start-date"
                  type="date"
                  value={startDate}
                  onChange={(e) => setForm({ ...form, startDate: e.target.value })}
                className="w-full px-3 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white focus:outline-none focus:border-accent-red"
              />
            </div>
            <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5" htmlFor="job-end-date">Enddatum</label>
                <input
                  id="job-end-date"
                  type="date"
                  value={endDate}
                  onChange={(e) => setForm({ ...form, endDate: e.target.value })}
                  min={startDate || undefined}
                className="w-full px-3 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white focus:outline-none focus:border-accent-red"
              />
            </div>
          </div>

          {/* Additional material that is not represented by a commercial position. */}
          <div className="border-t border-white/10 pt-5">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h4 className="text-sm font-semibold text-white">Zusätzlicher Materialbedarf</h4>
                <p className="text-xs text-gray-500 mt-0.5">
                  {startDate && endDate ? 'Bedarf ohne Angebotsposition. Verfügbarkeit basiert auf dem gewählten Zeitraum.' : 'Bitte zuerst den Zeitraum setzen.'}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setShowPicker((v) => !v)}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-white/10 hover:bg-white/15 rounded-lg text-sm transition-colors"
              >
                <Plus className="w-3.5 h-3.5" /> Material hinzufügen
              </button>
            </div>

            {showPicker && (
              <div className="mb-4 p-4 bg-white/3 border border-white/10 rounded-xl">
                <ProductPicker
                  startDate={startDate}
                  endDate={endDate}
                  jobId={jobId}
                  onSelect={addProduct}
                />
              </div>
            )}

            <SelectedProductsSummary selections={selections} onChange={setSelections} />
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={saving}
              className="flex items-center gap-2 px-6 py-2.5 bg-accent-red hover:bg-accent-red/80 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
            >
              <Check className="w-4 h-4" />
              {saving ? 'Wird gespeichert...' : 'Speichern'}
            </button>
            <button type="button" onClick={onCancel} className="flex items-center gap-2 px-4 py-2.5 bg-white/10 hover:bg-white/15 rounded-lg transition-colors">
              <X className="w-4 h-4" /> Abbrechen
            </button>
          </div>
        </form>
      </section>
      <aside className="jobs-section-stack">
        <section className="jobs-card">
          <div className="jobs-card-heading"><h2>Vor dem Speichern</h2></div>
          <div className="jobs-card-body">
            <ul className="jobs-steps">
              <li className={`jobs-step ${form.customer_id ? 'is-done' : ''}`}><span className="jobs-step-icon">{form.customer_id ? '✓' : '1'}</span><span>Kunden auswählen</span></li>
              <li className={`jobs-step ${form.description?.trim() ? 'is-done' : ''}`}><span className="jobs-step-icon">{form.description?.trim() ? '✓' : '2'}</span><span>Jobtitel festlegen</span></li>
              <li className={`jobs-step ${startDate && endDate ? 'is-done' : ''}`}><span className="jobs-step-icon">{startDate && endDate ? '✓' : '3'}</span><span>Zeitraum ergänzen (vor Bestätigung erforderlich)</span></li>
              <li className="jobs-step"><span className="jobs-step-icon">4</span><span>Nach dem Speichern Positionen, Personal und Geräte zuordnen</span></li>
            </ul>
          </div>
        </section>
        <section className="jobs-card">
          <div className="jobs-card-heading"><h2>Materialplanung</h2></div>
          <div className="jobs-card-body jobs-form-note">{selections.reduce((sum, item) => sum + item.quantity, 0)} zusätzlich geplante Produkteinheiten. Produkte aus Auftragspositionen werden automatisch im Bedarf berücksichtigt.</div>
        </section>
      </aside>
      </div>
    </div>
  );
}

// ── Job Employees Panel ───────────────────────────────────────

function JobEmployeesPanel({ jobId }: { jobId: number }) {
  const [assigned, setAssigned] = useState<JobEmployee[]>([]);
  const [available, setAvailable] = useState<Employee[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [selectedId, setSelectedId] = useState<number | ''>('');
  const [role, setRole] = useState('');

  const load = async () => {
    try {
      const [asgn, avail] = await Promise.all([
        jobEmployeesApi.list(jobId),
        employeesApi.listActive(),
      ]);
      setAssigned(asgn.data);
      setAvailable(avail.data);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [jobId]);

  const unassigned = available.filter(e => !assigned.some(a => a.employee_id === e.id));

  const assign = async () => {
    if (!selectedId) return;
    await jobEmployeesApi.assign(jobId, Number(selectedId), role || undefined);
    setAdding(false);
    setSelectedId('');
    setRole('');
    load();
  };

  const remove = async (employeeId: number) => {
    await jobEmployeesApi.remove(jobId, employeeId);
    load();
  };

  return (
    <div className="mt-6 p-4 rounded-xl bg-white/5 border border-white/5">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-white/60 uppercase tracking-wide">Bearbeiter</h3>
        {!adding && (
          <button
            onClick={() => setAdding(true)}
            className="flex items-center gap-1 text-xs px-2 py-1 rounded-lg bg-white/10 text-white/60 hover:bg-white/15 hover:text-white transition-colors"
          >
            <Plus className="w-3 h-3" /> Zuweisen
          </button>
        )}
      </div>

      {loading ? (
        <div className="text-white/30 text-sm">Lädt...</div>
      ) : (
        <>
          {assigned.length === 0 && !adding && (
            <div className="text-white/30 text-sm">Noch keine Bearbeiter zugewiesen.</div>
          )}
          <div className="space-y-1">
            {assigned.map(je => (
              <div key={je.employee_id} className="flex items-center gap-2 text-sm">
                <span className="text-white/80">{je.employee.first_name} {je.employee.last_name}</span>
                {je.role && <span className="text-white/40 text-xs">({je.role})</span>}
                {je.employee.email && <span className="text-white/30 text-xs">{je.employee.email}</span>}
                <button
                  onClick={() => remove(je.employee_id)}
                  className="ml-auto p-1 rounded text-red-400/50 hover:text-red-400 hover:bg-red-400/10"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
          </div>

          {adding && (
            <div className="mt-3 flex flex-wrap gap-2 items-end">
              <div>
                <label className="block text-xs text-white/40 mb-1">Mitarbeiter</label>
                <select
                  className="px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
                  value={selectedId}
                  onChange={e => setSelectedId(Number(e.target.value) || '')}
                >
                  <option value="">Auswählen...</option>
                  {unassigned.map(e => (
                    <option key={e.id} value={e.id}>{e.first_name} {e.last_name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-white/40 mb-1">Rolle (optional)</label>
                <input
                  className="px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
                  placeholder="z.B. Ton"
                  value={role}
                  onChange={e => setRole(e.target.value)}
                />
              </div>
              <button onClick={assign} className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500 flex items-center gap-1">
                <Check className="w-3.5 h-3.5" /> Zuweisen
              </button>
              <button onClick={() => setAdding(false)} className="px-3 py-1.5 rounded-lg bg-white/10 text-white/60 text-sm hover:bg-white/15">
                Abbrechen
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── Jobs List ─────────────────────────────────────────────────

export function JobsPage() {
  const { id: paramId } = useParams<{ id?: string }>();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const detailId = paramId ? Number(paramId) : null;
  const isEdit = !!detailId && pathname === `/jobs/${paramId}/edit`;
  const isNew = pathname === '/jobs/new' || pathname === '/jobs/new/';
  const [jobs, setJobs] = useState<Job[]>([]);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    jobsApi.getAll().then((r) => setJobs(r.data.jobs || [])).catch((e: any) => toast.error(e)).finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const filtered = jobs.filter((j) => {
    const q = search.toLowerCase();
    return (
      (statusFilter === 'all' || String(jobStatusId(j)) === statusFilter) &&
      (
        j.job_code.toLowerCase().includes(q) ||
        (j.description || '').toLowerCase().includes(q) ||
        customerName(j.customer).toLowerCase().includes(q)
      )
    );
  });

  if (isEdit && detailId) {
    return <JobForm jobId={detailId} onSaved={(id) => { navigate(`/jobs/${id}`); load(); }} onCancel={() => navigate(`/jobs/${detailId}`)} />;
  }

  if (isNew) {
    return <JobForm onSaved={(id) => { navigate(`/jobs/${id}`); load(); }} onCancel={() => navigate('/jobs')} />;
  }

  if (detailId && !isNew && !isEdit) {
    return <JobDetail id={detailId} onBack={() => { navigate('/jobs'); load(); }} />;
  }

  return (
    <div className="jobs-workspace">
      <header className="jobs-header">
        <div className="jobs-header-main">
          <span className="jobs-eyebrow">RentalCore · Aufträge</span>
          <h1 className="jobs-title">Jobs</h1>
          <p className="jobs-subtitle">Alle Aufträge, Termine und nächsten Schritte an einem Ort.</p>
        </div>
        <div className="jobs-header-actions">
          <button onClick={() => navigate('/jobs/new')} className="suite-button suite-button--primary">
            <Plus className="w-4 h-4" /> Job anlegen
          </button>
        </div>
      </header>

      <div className="jobs-metrics" aria-label="Jobübersicht">
        {[
          { label: 'Alle Jobs', value: jobs.length, caption: 'Gesamtbestand' },
          { label: 'In Planung', value: jobs.filter((job) => jobStatusId(job) === 1).length, caption: 'Noch nicht bestätigt' },
          { label: 'Bestätigt', value: jobs.filter((job) => jobStatusId(job) === 2).length, caption: 'Für Warehouse freigegeben' },
          { label: 'Abgeschlossen', value: jobs.filter((job) => jobStatusId(job) === 4).length, caption: 'Durchgeführt' },
        ].map((metric) => (
          <div className="jobs-card jobs-metric" key={metric.label}>
            <span className="jobs-metric-label">{metric.label}</span>
            <span className="jobs-metric-value">{metric.value}</span>
            <span className="jobs-metric-caption">{metric.caption}</span>
          </div>
        ))}
      </div>

      <section className="jobs-card" aria-label="Jobliste">
        <div className="jobs-search-row">
          <div className="suite-search-field flex-1">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Job suchen..."
              className="w-full pl-10 pr-4 py-2 bg-white/5 border border-white/10 rounded-lg text-sm focus:outline-none focus:border-accent-red"
            />
          </div>
          <select
            className="jobs-filter"
            aria-label="Nach Status filtern"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
          >
            <option value="all">Alle Status</option>
            <option value="1">Planung</option>
            <option value="2">Bestätigt</option>
            <option value="4">Abgeschlossen</option>
            <option value="6">Storniert</option>
          </select>
          <button onClick={load} className="suite-button" aria-label="Jobliste aktualisieren" title="Aktualisieren">
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>

        {loading ? (
          <div className="flex justify-center py-16">
            <div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="jobs-empty">
            <Briefcase className="w-8 h-8 mx-auto mb-3 opacity-50" />
            <p>{search || statusFilter !== 'all' ? 'Keine Jobs für diese Auswahl gefunden.' : 'Noch keine Jobs vorhanden.'}</p>
            {!search && statusFilter === 'all' && <button className="suite-button suite-button--primary mt-4" onClick={() => navigate('/jobs/new')}>Ersten Job anlegen</button>}
          </div>
        ) : (
          <div className="suite-table-wrap">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-white/10 text-gray-400">
                  <th className="text-left px-6 py-3 font-medium">Job</th>
                  <th className="text-left px-6 py-3 font-medium">Kunde</th>
                  <th className="text-left px-6 py-3 font-medium hidden md:table-cell">Zeitraum</th>
                  <th className="text-left px-6 py-3 font-medium hidden sm:table-cell">Status</th>
                  <th className="text-right px-6 py-3 font-medium hidden lg:table-cell">Auftragswert</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {filtered.map((job) => (
                  <tr
                    key={job.jobID}
                    className="jobs-table-row"
                    onClick={() => navigate(`/jobs/${job.jobID}`)}
                    onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); navigate(`/jobs/${job.jobID}`); } }}
                    tabIndex={0}
                    aria-label={`${job.job_code}: ${job.description || 'Job öffnen'}`}
                  >
                    <td className="px-6 py-4">
                      <div className="font-medium text-[var(--text-primary)]">{job.job_code}</div>
                      {job.description && <div className="text-gray-400 text-xs mt-0.5 truncate max-w-xs">{job.description}</div>}
                    </td>
                    <td className="px-6 py-4 text-gray-300">
                      <div className="flex items-center gap-2">
                        <User className="w-3.5 h-3.5 text-gray-500 flex-shrink-0" />
                        {customerName(job.customer)}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-gray-400 hidden md:table-cell">
                      <div className="flex items-center gap-1.5">
                        <Calendar className="w-3.5 h-3.5 flex-shrink-0" />
                        {formatDate(job.startDate)} – {formatDate(job.endDate)}
                      </div>
                    </td>
                    <td className="px-6 py-4 hidden sm:table-cell">
                      {job.status && (
                        <span className={`jobs-status px-2 py-0.5 rounded-full text-xs font-medium border ${statusColor(job.status.status)}`}>
                          {job.status.status}
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right text-gray-300 hidden lg:table-cell">
                      €{(job.final_revenue ?? job.revenue ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2 })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
