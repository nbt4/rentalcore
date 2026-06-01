import { useState, useEffect } from 'react';
import { venuesApi, type Venue } from '../lib/api';
import { Plus, Pencil, Trash2, Save, X, MapPin } from 'lucide-react';

interface VenueFormState {
  name: string;
  street: string;
  house_number: string;
  zip: string;
  city: string;
  contact_name: string;
  phone: string;
  email: string;
  notes: string;
}

const emptyForm = (): VenueFormState => ({
  name: '', street: '', house_number: '', zip: '', city: '',
  contact_name: '', phone: '', email: '', notes: '',
});

const toForm = (v: Venue): VenueFormState => ({
  name: v.name,
  street: v.street || '',
  house_number: v.house_number || '',
  zip: v.zip || '',
  city: v.city || '',
  contact_name: v.contact_name || '',
  phone: v.phone || '',
  email: v.email || '',
  notes: v.notes || '',
});

const toPayload = (f: VenueFormState) => ({
  name: f.name,
  street: f.street || null,
  house_number: f.house_number || null,
  zip: f.zip || null,
  city: f.city || null,
  contact_name: f.contact_name || null,
  phone: f.phone || null,
  email: f.email || null,
  notes: f.notes || null,
});

export function VenuesPage() {
  const [venues, setVenues] = useState<Venue[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<VenueFormState>(emptyForm());

  const load = async () => {
    try {
      const res = await venuesApi.list();
      setVenues(res.data);
    } catch {
      setError('Veranstaltungsorte konnten nicht geladen werden.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const startEdit = (v: Venue) => {
    setEditingId(v.id);
    setForm(toForm(v));
    setShowNew(false);
  };

  const cancelEdit = () => { setEditingId(null); setForm(emptyForm()); };

  const saveEdit = async () => {
    if (!editingId || !form.name.trim()) return;
    try {
      await venuesApi.update(editingId, toPayload(form));
      setEditingId(null);
      await load();
    } catch {
      setError('Speichern fehlgeschlagen.');
    }
  };

  const create = async () => {
    if (!form.name.trim()) return;
    try {
      await venuesApi.create(toPayload(form));
      setShowNew(false);
      setForm(emptyForm());
      await load();
    } catch {
      setError('Erstellen fehlgeschlagen.');
    }
  };

  const remove = async (id: number) => {
    if (!confirm('Veranstaltungsort wirklich löschen? Verknüpfte Jobs verlieren den Ortsbezug.')) return;
    try {
      await venuesApi.delete(id);
      await load();
    } catch {
      setError('Löschen fehlgeschlagen.');
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <MapPin className="w-6 h-6 text-accent-red" />
          <h1 className="text-2xl font-bold text-white">Veranstaltungsorte</h1>
        </div>
        <button
          onClick={() => { setShowNew(true); setEditingId(null); setForm(emptyForm()); }}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-accent-red text-white hover:bg-accent-red/80 transition-colors"
        >
          <Plus className="w-4 h-4" /> Neu anlegen
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/20 text-red-300 text-sm flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)}><X className="w-4 h-4" /></button>
        </div>
      )}

      {showNew && (
        <div className="mb-6 p-4 rounded-xl bg-white/5 border border-white/10">
          <h2 className="text-sm font-semibold text-white/60 mb-3 uppercase tracking-wide">Neuer Veranstaltungsort</h2>
          <VenueForm form={form} setForm={setForm} />
          <div className="flex gap-2 mt-4">
            <button
              onClick={create}
              disabled={!form.name.trim()}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500 disabled:opacity-40"
            >
              <Save className="w-3.5 h-3.5" /> Speichern
            </button>
            <button
              onClick={() => setShowNew(false)}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white/10 text-white/70 text-sm hover:bg-white/20"
            >
              <X className="w-3.5 h-3.5" /> Abbrechen
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="text-white/40 text-center py-12">Lädt...</div>
      ) : (
        <div className="rounded-xl bg-white/5 border border-white/10 overflow-hidden">
          {venues.length === 0 ? (
            <div className="text-white/30 text-center py-12">Noch keine Veranstaltungsorte vorhanden.</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-white/10">
                  <th className="text-left px-4 py-3 text-white/40 font-medium">Name</th>
                  <th className="text-left px-4 py-3 text-white/40 font-medium">Stadt</th>
                  <th className="text-left px-4 py-3 text-white/40 font-medium hidden sm:table-cell">Ansprechpartner</th>
                  <th className="px-4 py-3 w-24" />
                </tr>
              </thead>
              <tbody>
                {venues.map(v => (
                  <>
                    <tr key={v.id} className="border-b border-white/5 hover:bg-white/3 transition-colors">
                      <td className="px-4 py-3 text-white font-medium">{v.name}</td>
                      <td className="px-4 py-3 text-white/60">
                        {[v.zip, v.city].filter(Boolean).join(' ') || '—'}
                      </td>
                      <td className="px-4 py-3 text-white/60 hidden sm:table-cell">
                        {v.contact_name || '—'}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            onClick={() => startEdit(v)}
                            className="p-1.5 rounded text-white/40 hover:text-white hover:bg-white/10"
                          >
                            <Pencil className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => remove(v.id)}
                            className="p-1.5 rounded text-red-400/60 hover:text-red-400 hover:bg-red-400/10"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                    {editingId === v.id && (
                      <tr key={`edit-${v.id}`} className="bg-white/3">
                        <td colSpan={4} className="px-4 py-4">
                          <VenueForm form={form} setForm={setForm} />
                          <div className="flex gap-2 mt-4">
                            <button
                              onClick={saveEdit}
                              disabled={!form.name.trim()}
                              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500 disabled:opacity-40"
                            >
                              <Save className="w-3.5 h-3.5" /> Speichern
                            </button>
                            <button
                              onClick={cancelEdit}
                              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white/10 text-white/70 text-sm hover:bg-white/20"
                            >
                              <X className="w-3.5 h-3.5" /> Abbrechen
                            </button>
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

function VenueForm({ form, setForm }: { form: VenueFormState; setForm: (f: VenueFormState) => void }) {
  const field = (key: keyof VenueFormState, label: string, required = false) => (
    <div>
      <label className="block text-xs text-white/40 mb-1">
        {label}{required && <span className="text-red-400 ml-0.5">*</span>}
      </label>
      <input
        type="text"
        className="w-full px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent-red"
        value={form[key]}
        onChange={e => setForm({ ...form, [key]: e.target.value })}
      />
    </div>
  );

  return (
    <div className="space-y-3">
      {field('name', 'Name', true)}
      <div className="grid grid-cols-2 gap-3">
        <div className="col-span-2 sm:col-span-1">{field('contact_name', 'Ansprechpartner')}</div>
        <div className="col-span-2 sm:col-span-1">{field('phone', 'Telefon')}</div>
        {field('email', 'E-Mail')}
      </div>
      <div className="grid grid-cols-3 gap-3">
        <div className="col-span-2">{field('street', 'Straße')}</div>
        <div>{field('house_number', 'Hausnr.')}</div>
        <div>{field('zip', 'PLZ')}</div>
        <div className="col-span-2">{field('city', 'Stadt')}</div>
      </div>
      <div>
        <label className="block text-xs text-white/40 mb-1">Notizen</label>
        <textarea
          className="w-full px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent-red resize-none"
          rows={2}
          value={form.notes}
          onChange={e => setForm({ ...form, notes: e.target.value })}
        />
      </div>
    </div>
  );
}
