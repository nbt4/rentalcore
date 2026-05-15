import { useState, useEffect } from 'react';
import { employeesApi, skillsApi, type Employee, type Skill } from '../lib/api';
import { Plus, Pencil, Trash2, Save, X, ChevronDown, ChevronUp } from 'lucide-react';

interface EmployeeFormState {
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  mobile: string;
  address: string;
  date_of_birth: string;
  iban: string;
  notes: string;
  is_active: boolean;
  skill_ids: number[];
}

const emptyForm = (): EmployeeFormState => ({
  first_name: '', last_name: '', email: '', phone: '', mobile: '',
  address: '', date_of_birth: '', iban: '', notes: '', is_active: true, skill_ids: [],
});

const toForm = (e: Employee): EmployeeFormState => ({
  first_name: e.first_name,
  last_name: e.last_name,
  email: e.email || '',
  phone: e.phone || '',
  mobile: e.mobile || '',
  address: e.address || '',
  date_of_birth: e.date_of_birth || '',
  iban: e.iban || '',
  notes: e.notes || '',
  is_active: e.is_active,
  skill_ids: e.skills?.map(s => s.id) || [],
});

export function EmployeesPage() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [form, setForm] = useState<EmployeeFormState>(emptyForm());
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    try {
      const [empRes, skillRes] = await Promise.all([employeesApi.list(), skillsApi.list()]);
      setEmployees(empRes.data);
      setSkills(skillRes.data);
    } catch {
      setError('Daten konnten nicht geladen werden.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const startEdit = (e: Employee) => {
    setEditingId(e.id);
    setForm(toForm(e));
    setShowNew(false);
    setExpandedId(e.id);
  };

  const cancelEdit = () => { setEditingId(null); setForm(emptyForm()); };

  const saveEdit = async () => {
    if (!editingId) return;
    try {
      await employeesApi.update(editingId, form);
      setEditingId(null);
      await load();
    } catch {
      setError('Speichern fehlgeschlagen.');
    }
  };

  const createEmployee = async () => {
    if (!form.first_name.trim() || !form.last_name.trim()) return;
    try {
      await employeesApi.create(form);
      setShowNew(false);
      setForm(emptyForm());
      await load();
    } catch {
      setError('Erstellen fehlgeschlagen.');
    }
  };

  const deleteEmployee = async (id: number) => {
    if (!confirm('Mitarbeiter wirklich löschen?')) return;
    try {
      await employeesApi.delete(id);
      await load();
    } catch {
      setError('Löschen fehlgeschlagen.');
    }
  };

  const toggleExpand = (id: number) => setExpandedId(expandedId === id ? null : id);

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Mitarbeiter</h1>
        <button
          onClick={() => { setShowNew(true); setEditingId(null); setForm(emptyForm()); setExpandedId(null); }}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-accent text-white hover:bg-accent/80 transition-colors"
        >
          <Plus className="w-4 h-4" /> Neuer Mitarbeiter
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/20 text-red-300 text-sm">{error}</div>
      )}

      {showNew && (
        <div className="mb-6 p-4 rounded-xl bg-white/5 border border-white/10">
          <h2 className="text-sm font-semibold text-white/60 mb-3 uppercase tracking-wide">Neuer Mitarbeiter</h2>
          <EmployeeForm form={form} setForm={setForm} skills={skills} />
          <div className="flex gap-2 mt-4">
            <button onClick={createEmployee} className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500">
              <Save className="w-3.5 h-3.5" /> Speichern
            </button>
            <button onClick={() => setShowNew(false)} className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white/10 text-white/70 text-sm hover:bg-white/20">
              <X className="w-3.5 h-3.5" /> Abbrechen
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="text-white/40 text-center py-12">Lädt...</div>
      ) : (
        <div className="space-y-2">
          {employees.map(emp => (
            <div key={emp.id} className="rounded-xl bg-white/5 border border-white/5 overflow-hidden">
              <div className="flex items-center gap-3 p-4">
                <button onClick={() => toggleExpand(emp.id)} className="flex-1 flex items-center gap-3 text-left">
                  <div className="flex-1">
                    <span className="text-white font-medium">
                      {emp.first_name} {emp.last_name}
                    </span>
                    {!emp.is_active && (
                      <span className="ml-2 text-xs text-white/30 bg-white/5 px-2 py-0.5 rounded-full">Inaktiv</span>
                    )}
                    {emp.email && <span className="ml-3 text-white/40 text-sm">{emp.email}</span>}
                    {emp.skills && emp.skills.length > 0 && (
                      <span className="ml-3 text-xs text-accent/70">
                        {emp.skills.map(s => s.name).join(', ')}
                      </span>
                    )}
                  </div>
                  {expandedId === emp.id ? <ChevronUp className="w-4 h-4 text-white/30" /> : <ChevronDown className="w-4 h-4 text-white/30" />}
                </button>
                <button onClick={() => startEdit(emp)} className="p-1.5 rounded text-white/40 hover:text-white hover:bg-white/10"><Pencil className="w-4 h-4" /></button>
                <button onClick={() => deleteEmployee(emp.id)} className="p-1.5 rounded text-red-400/60 hover:text-red-400 hover:bg-red-400/10"><Trash2 className="w-4 h-4" /></button>
              </div>

              {expandedId === emp.id && (
                <div className="border-t border-white/5 p-4">
                  {editingId === emp.id ? (
                    <>
                      <EmployeeForm form={form} setForm={setForm} skills={skills} />
                      <div className="flex gap-2 mt-4">
                        <button onClick={saveEdit} className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500">
                          <Save className="w-3.5 h-3.5" /> Speichern
                        </button>
                        <button onClick={cancelEdit} className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white/10 text-white/70 text-sm hover:bg-white/20">
                          <X className="w-3.5 h-3.5" /> Abbrechen
                        </button>
                      </div>
                    </>
                  ) : (
                    <EmployeeDetail emp={emp} />
                  )}
                </div>
              )}
            </div>
          ))}
          {employees.length === 0 && (
            <div className="text-white/30 text-center py-12">Noch keine Mitarbeiter vorhanden.</div>
          )}
        </div>
      )}
    </div>
  );
}

function EmployeeDetail({ emp }: { emp: Employee }) {
  const fields: [string, string | undefined][] = [
    ['Telefon', emp.phone],
    ['Mobil', emp.mobile],
    ['Adresse', emp.address],
    ['Geburtsdatum', emp.date_of_birth],
    ['IBAN', emp.iban],
    ['Notizen', emp.notes],
  ];
  return (
    <div className="grid grid-cols-2 gap-2 text-sm">
      {fields.filter(([, v]) => v).map(([label, value]) => (
        <div key={label}>
          <span className="text-white/40">{label}: </span>
          <span className="text-white/80">{value}</span>
        </div>
      ))}
    </div>
  );
}

function EmployeeForm({ form, setForm, skills }: {
  form: EmployeeFormState;
  setForm: (f: EmployeeFormState) => void;
  skills: Skill[];
}) {
  const toggleSkill = (id: number) => {
    const ids = form.skill_ids.includes(id)
      ? form.skill_ids.filter(s => s !== id)
      : [...form.skill_ids, id];
    setForm({ ...form, skill_ids: ids });
  };

  const field = (key: keyof EmployeeFormState, label: string, type = 'text') => (
    <div>
      <label className="block text-xs text-white/40 mb-1">{label}</label>
      <input
        type={type}
        className="w-full px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
        value={form[key] as string}
        onChange={e => setForm({ ...form, [key]: e.target.value })}
      />
    </div>
  );

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        {field('first_name', 'Vorname')}
        {field('last_name', 'Nachname')}
        {field('email', 'E-Mail', 'email')}
        {field('phone', 'Telefon', 'tel')}
        {field('mobile', 'Mobil', 'tel')}
        {field('date_of_birth', 'Geburtsdatum', 'date')}
        {field('iban', 'IBAN')}
      </div>
      <div>
        <label className="block text-xs text-white/40 mb-1">Adresse</label>
        <textarea
          className="w-full px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent resize-none"
          rows={2}
          value={form.address}
          onChange={e => setForm({ ...form, address: e.target.value })}
        />
      </div>
      <div>
        <label className="block text-xs text-white/40 mb-1">Notizen</label>
        <textarea
          className="w-full px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent resize-none"
          rows={2}
          value={form.notes}
          onChange={e => setForm({ ...form, notes: e.target.value })}
        />
      </div>
      {skills.length > 0 && (
        <div>
          <label className="block text-xs text-white/40 mb-2">Skills</label>
          <div className="flex flex-wrap gap-2">
            {skills.map(s => (
              <button
                key={s.id}
                type="button"
                onClick={() => toggleSkill(s.id)}
                className={`px-3 py-1 rounded-full text-xs transition-colors ${
                  form.skill_ids.includes(s.id)
                    ? 'bg-accent text-white'
                    : 'bg-white/10 text-white/50 hover:bg-white/15'
                }`}
              >
                {s.name}
              </button>
            ))}
          </div>
        </div>
      )}
      <label className="flex items-center gap-2 text-sm text-white/70 cursor-pointer">
        <input
          type="checkbox"
          checked={form.is_active}
          onChange={e => setForm({ ...form, is_active: e.target.checked })}
          className="rounded"
        />
        Aktiv
      </label>
    </div>
  );
}
