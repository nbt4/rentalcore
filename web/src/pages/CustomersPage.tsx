import { useEffect, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Plus, Search, RefreshCw, Users, ArrowLeft, X, Check, Trash2, Mail, Phone, MapPin } from 'lucide-react';
import { customersApi } from '../lib/api';
import type { Customer } from '../lib/api';

function customerDisplayName(c: Customer) {
  if (c.companyname) return c.companyname;
  return `${c.firstname || ''} ${c.lastname || ''}`.trim() || '—';
}

function InfoRow({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div className="flex justify-between text-sm py-1.5 border-b border-white/5 last:border-0">
      <span className="text-gray-400">{label}</span>
      <span className="text-white">{value}</span>
    </div>
  );
}

function CustomerDetail({ id, onBack }: { id: number; onBack: () => void }) {
  const [customer, setCustomer] = useState<Customer | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    customersApi.getById(id).then((r) => setCustomer(r.data)).catch(console.error).finally(() => setLoading(false));
  }, [id]);

  const handleDelete = async () => {
    if (!confirm('Kontakt wirklich löschen?')) return;
    await customersApi.delete(id);
    onBack();
  };

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;
  if (!customer) return <div className="text-center text-gray-400 py-20">Kontakt nicht gefunden.</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button onClick={onBack} className="p-2 hover:bg-white/10 rounded-lg transition-colors"><ArrowLeft className="w-5 h-5" /></button>
        <div>
          <h1 className="text-2xl font-bold">{customerDisplayName(customer)}</h1>
          {customer.customertype && <p className="text-gray-400 text-sm">{customer.customertype}</p>}
          <div className="flex gap-2 mt-1">
            {customer.is_customer && (
              <span className="px-2 py-0.5 text-xs rounded-full bg-blue-500/20 text-blue-400 border border-blue-500/30">Kunde</span>
            )}
            {customer.is_supplier && (
              <span className="px-2 py-0.5 text-xs rounded-full bg-green-500/20 text-green-400 border border-green-500/30">Lieferant</span>
            )}
          </div>
        </div>
        <div className="ml-auto flex gap-2">
          <button onClick={() => navigate(`/customers/${id}/edit`)} className="flex items-center gap-2 px-4 py-2 bg-white/10 hover:bg-white/15 rounded-lg text-sm transition-colors">
            Bearbeiten
          </button>
          <button onClick={handleDelete} className="flex items-center gap-2 px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-400 rounded-lg text-sm transition-colors">
            <Trash2 className="w-4 h-4" /> Löschen
          </button>
        </div>
      </div>

      <div className="glass-dark rounded-xl border border-white/10 p-5 max-w-md">
        <h3 className="font-semibold text-white mb-4">Kontaktdaten</h3>
        <InfoRow label="Unternehmen" value={customer.companyname} />
        <InfoRow label="Vorname" value={customer.firstname} />
        <InfoRow label="Nachname" value={customer.lastname} />
        <InfoRow label="E-Mail" value={customer.email} />
        <InfoRow label="Telefon" value={customer.phonenumber} />
        <InfoRow label="Straße" value={customer.street ? `${customer.street} ${customer.housenumber || ''}` : null} />
        <InfoRow label="PLZ / Stadt" value={customer.ZIP && customer.city ? `${customer.ZIP} ${customer.city}` : customer.city || customer.ZIP} />
        {customer.notes && (
          <div className="mt-3 pt-3 border-t border-white/10">
            <p className="text-gray-400 text-xs mb-1">Notizen</p>
            <p className="text-gray-300 text-sm">{customer.notes}</p>
          </div>
        )}
      </div>
    </div>
  );
}

function CustomerForm({ customerId, onSaved, onCancel }: { customerId?: number; onSaved: (id: number) => void; onCancel: () => void }) {
  const [form, setForm] = useState<Partial<Customer>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const f = (field: keyof Customer) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) =>
    setForm((prev) => ({ ...prev, [field]: e.target.value }));

  useEffect(() => {
    if (!customerId) return;
    setLoading(true);
    customersApi.getById(customerId).then((r) => setForm(r.data)).catch(console.error).finally(() => setLoading(false));
  }, [customerId]);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      if (customerId) {
        await customersApi.update(customerId, form);
        onSaved(customerId);
      } else {
        const res = await customersApi.create(form);
        onSaved(res.data.customer_id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Speichern fehlgeschlagen');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="flex justify-center py-20"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button onClick={onCancel} className="p-2 hover:bg-white/10 rounded-lg transition-colors"><ArrowLeft className="w-5 h-5" /></button>
        <h1 className="text-2xl font-bold">{customerId ? 'Kontakt bearbeiten' : 'Neuer Kontakt'}</h1>
      </div>
      <div className="glass-dark rounded-xl border border-white/10 p-6 max-w-xl">
        {error && <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>}
        <form onSubmit={save} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Kundentyp</label>
            <select value={form.customertype || ''} onChange={f('customertype')} className="w-full px-3 py-2.5 rounded-lg">
              <option value="">— Typ wählen —</option>
              <option value="Unternehmen">Unternehmen</option>
              <option value="Privat">Privat</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Unternehmen</label>
            <input type="text" value={form.companyname || ''} onChange={f('companyname')} placeholder="Firmenname" className="w-full px-3 py-2.5 rounded-lg" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Vorname</label>
              <input type="text" value={form.firstname || ''} onChange={f('firstname')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Nachname</label>
              <input type="text" value={form.lastname || ''} onChange={f('lastname')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">E-Mail</label>
            <input type="email" value={form.email || ''} onChange={f('email')} className="w-full px-3 py-2.5 rounded-lg" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Telefon</label>
            <input type="text" value={form.phonenumber || ''} onChange={f('phonenumber')} className="w-full px-3 py-2.5 rounded-lg" />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Straße</label>
              <input type="text" value={form.street || ''} onChange={f('street')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Nr.</label>
              <input type="text" value={form.housenumber || ''} onChange={f('housenumber')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">PLZ</label>
              <input type="text" value={form.ZIP || ''} onChange={f('ZIP')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Stadt</label>
              <input type="text" value={form.city || ''} onChange={f('city')} className="w-full px-3 py-2.5 rounded-lg" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1.5">Notizen</label>
            <textarea value={form.notes || ''} onChange={f('notes')} rows={3} className="w-full px-3 py-2.5 rounded-lg resize-none" />
          </div>
          <div className="flex gap-6 pt-2">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_customer ?? true}
                onChange={e => setForm(f => ({ ...f, is_customer: e.target.checked }))}
                className="w-4 h-4 rounded"
              />
              <span className="text-gray-300">Kunde</span>
            </label>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_supplier ?? false}
                onChange={e => setForm(f => ({ ...f, is_supplier: e.target.checked }))}
                className="w-4 h-4 rounded"
              />
              <span className="text-gray-300">Lieferant</span>
            </label>
          </div>
          <div className="flex gap-3 pt-2">
            <button type="submit" disabled={saving} className="flex items-center gap-2 px-6 py-2.5 bg-accent-red hover:bg-accent-red/80 disabled:opacity-50 text-white rounded-lg font-medium transition-colors">
              <Check className="w-4 h-4" />{saving ? 'Wird gespeichert...' : 'Speichern'}
            </button>
            <button type="button" onClick={onCancel} className="flex items-center gap-2 px-4 py-2.5 bg-white/10 hover:bg-white/15 rounded-lg transition-colors">
              <X className="w-4 h-4" /> Abbrechen
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

type RoleFilter = 'all' | 'customer' | 'supplier';

export function CustomersPage() {
  const { id: paramId } = useParams<{ id?: string }>();
  const navigate = useNavigate();
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');

  const load = useCallback(() => {
    setLoading(true);
    customersApi.getAll(roleFilter === 'all' ? {} : { role: roleFilter })
      .then((r) => setCustomers(r.data.customers || [])).catch(console.error).finally(() => setLoading(false));
  }, [roleFilter]);

  useEffect(() => { load(); }, [load]);

  if (paramId && paramId !== 'new') {
    const isEdit = window.location.pathname.endsWith('/edit');
    if (isEdit) {
      return <CustomerForm customerId={Number(paramId)} onSaved={(id) => { navigate(`/customers/${id}`); load(); }} onCancel={() => navigate(`/customers/${paramId}`)} />;
    }
    return <CustomerDetail id={Number(paramId)} onBack={() => { navigate('/customers'); load(); }} />;
  }

  if (paramId === 'new') {
    return <CustomerForm onSaved={(id) => { navigate(`/customers/${id}`); load(); }} onCancel={() => navigate('/customers')} />;
  }

  const filtered = customers.filter((c) => {
    const q = search.toLowerCase();
    return customerDisplayName(c).toLowerCase().includes(q) || (c.email || '').toLowerCase().includes(q) || (c.city || '').toLowerCase().includes(q);
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-accent-red" /> Kontakte</h1>
          <p className="text-gray-400 text-sm mt-1">{customers.length} Kontakte insgesamt</p>
        </div>
        <button onClick={() => navigate('/customers/new')} className="flex items-center gap-2 px-4 py-2.5 bg-accent-red hover:bg-accent-red/80 text-white rounded-lg font-medium transition-colors">
          <Plus className="w-4 h-4" /> Neuer Kontakt
        </button>
      </div>

      <div className="flex gap-1 mb-4">
        {([
          { key: 'all', label: 'Alle' },
          { key: 'customer', label: 'Kunden' },
          { key: 'supplier', label: 'Lieferanten' },
        ] as { key: RoleFilter; label: string }[]).map(tab => (
          <button
            key={tab.key}
            onClick={() => setRoleFilter(tab.key)}
            className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
              roleFilter === tab.key
                ? 'bg-accent-red text-white'
                : 'bg-white/5 text-gray-400 hover:bg-white/10'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="glass-dark rounded-xl border border-white/10">
        <div className="flex items-center gap-3 p-4 border-b border-white/10">
          <div className="flex-1 relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input type="search" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Kontakt suchen..." className="w-full pl-10 pr-4 py-2 bg-white/5 border border-white/10 rounded-lg text-sm focus:outline-none focus:border-accent-red" />
          </div>
          <button onClick={load} className="p-2 hover:bg-white/10 rounded-lg transition-colors text-gray-400"><RefreshCw className="w-4 h-4" /></button>
        </div>

        {loading ? (
          <div className="flex justify-center py-16"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-16 text-gray-500">
            <Users className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p>{search ? 'Keine Kontakte gefunden' : 'Noch keine Kontakte vorhanden'}</p>
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {filtered.map((c) => (
              <div key={c.customer_id} className="flex items-center justify-between px-6 py-4 hover:bg-white/5 cursor-pointer transition-colors" onClick={() => navigate(`/customers/${c.customer_id}`)}>
                <div>
                  <div className="font-medium text-white">{customerDisplayName(c)}</div>
                  <div className="flex items-center gap-3 mt-0.5">
                    {c.email && <span className="flex items-center gap-1 text-xs text-gray-400"><Mail className="w-3 h-3" />{c.email}</span>}
                    {c.phonenumber && <span className="flex items-center gap-1 text-xs text-gray-400"><Phone className="w-3 h-3" />{c.phonenumber}</span>}
                    {c.city && <span className="flex items-center gap-1 text-xs text-gray-400"><MapPin className="w-3 h-3" />{c.city}</span>}
                  </div>
                  <div className="flex gap-1 mt-0.5">
                    {c.is_customer && <span className="text-xs text-blue-400">Kunde</span>}
                    {c.is_supplier && <span className="text-xs text-green-400">Lieferant</span>}
                  </div>
                </div>
                {c.customertype && (
                  <span className="px-2 py-0.5 rounded-full text-xs border border-white/20 text-gray-400">{c.customertype}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
