import { useState, useEffect } from 'react';
import { skillsApi, type Skill } from '../lib/api';
import { Plus, Pencil, Trash2, Save, X } from 'lucide-react';

const DEFAULT_CATEGORIES = [
  'Ton', 'Licht', 'Video', 'Rigging', 'Bühne', 'Logistik', 'Sonstiges',
];

interface SkillFormState {
  name: string;
  category: string;
  description: string;
}

const emptyForm = (): SkillFormState => ({ name: '', category: DEFAULT_CATEGORIES[0], description: '' });

export function SkillsPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<SkillFormState>(emptyForm());
  const [showNew, setShowNew] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    try {
      const res = await skillsApi.list();
      setSkills(res.data);
    } catch {
      setError('Skills konnten nicht geladen werden.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const startEdit = (s: Skill) => {
    setEditingId(s.id);
    setForm({ name: s.name, category: s.category, description: s.description || '' });
    setShowNew(false);
  };

  const cancelEdit = () => { setEditingId(null); setForm(emptyForm()); };

  const saveEdit = async () => {
    if (!editingId) return;
    try {
      await skillsApi.update(editingId, form);
      setEditingId(null);
      await load();
    } catch {
      setError('Speichern fehlgeschlagen.');
    }
  };

  const createSkill = async () => {
    if (!form.name.trim()) return;
    try {
      await skillsApi.create(form);
      setShowNew(false);
      setForm(emptyForm());
      await load();
    } catch {
      setError('Erstellen fehlgeschlagen.');
    }
  };

  const deleteSkill = async (id: number) => {
    if (!confirm('Skill wirklich löschen?')) return;
    try {
      await skillsApi.delete(id);
      await load();
    } catch {
      setError('Löschen fehlgeschlagen.');
    }
  };

  const grouped = DEFAULT_CATEGORIES.reduce<Record<string, Skill[]>>((acc, cat) => {
    acc[cat] = skills.filter(s => s.category === cat);
    return acc;
  }, {});
  const otherCats = [...new Set(skills.map(s => s.category))].filter(c => !DEFAULT_CATEGORIES.includes(c));
  otherCats.forEach(cat => { grouped[cat] = skills.filter(s => s.category === cat); });

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Skills</h1>
        <button
          onClick={() => { setShowNew(true); setEditingId(null); setForm(emptyForm()); }}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-accent text-white hover:bg-accent/80 transition-colors"
        >
          <Plus className="w-4 h-4" /> Neuer Skill
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-500/20 text-red-300 text-sm">{error}</div>
      )}

      {showNew && (
        <div className="mb-6 p-4 rounded-xl bg-white/5 border border-white/10">
          <h2 className="text-sm font-semibold text-white/60 mb-3 uppercase tracking-wide">Neuer Skill</h2>
          <SkillForm form={form} setForm={setForm} />
          <div className="flex gap-2 mt-3">
            <button onClick={createSkill} className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm hover:bg-green-500">
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
        <div className="space-y-6">
          {Object.entries(grouped).filter(([, s]) => s.length > 0).map(([cat, catSkills]) => (
            <div key={cat}>
              <h2 className="text-xs font-semibold text-white/40 uppercase tracking-widest mb-2">{cat}</h2>
              <div className="space-y-1">
                {catSkills.map(skill => (
                  <div key={skill.id} className="flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/8 border border-white/5">
                    {editingId === skill.id ? (
                      <>
                        <div className="flex-1">
                          <SkillForm form={form} setForm={setForm} />
                        </div>
                        <button onClick={saveEdit} className="p-1.5 rounded text-green-400 hover:bg-green-400/10"><Save className="w-4 h-4" /></button>
                        <button onClick={cancelEdit} className="p-1.5 rounded text-white/40 hover:bg-white/10"><X className="w-4 h-4" /></button>
                      </>
                    ) : (
                      <>
                        <div className="flex-1">
                          <span className="text-white font-medium">{skill.name}</span>
                          {skill.description && <span className="ml-2 text-white/40 text-sm">{skill.description}</span>}
                        </div>
                        <button onClick={() => startEdit(skill)} className="p-1.5 rounded text-white/40 hover:text-white hover:bg-white/10"><Pencil className="w-4 h-4" /></button>
                        <button onClick={() => deleteSkill(skill.id)} className="p-1.5 rounded text-red-400/60 hover:text-red-400 hover:bg-red-400/10"><Trash2 className="w-4 h-4" /></button>
                      </>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}
          {skills.length === 0 && (
            <div className="text-white/30 text-center py-12">Noch keine Skills vorhanden.</div>
          )}
        </div>
      )}
    </div>
  );
}

function SkillForm({ form, setForm }: { form: SkillFormState; setForm: (f: SkillFormState) => void }) {
  return (
    <div className="flex flex-wrap gap-2">
      <input
        className="flex-1 min-w-32 px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
        placeholder="Name"
        value={form.name}
        onChange={e => setForm({ ...form, name: e.target.value })}
      />
      <select
        className="px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
        value={form.category}
        onChange={e => setForm({ ...form, category: e.target.value })}
      >
        {DEFAULT_CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
      </select>
      <input
        className="flex-1 min-w-48 px-3 py-1.5 rounded-lg bg-white/10 text-white text-sm border border-white/10 focus:outline-none focus:border-accent"
        placeholder="Beschreibung (optional)"
        value={form.description}
        onChange={e => setForm({ ...form, description: e.target.value })}
      />
    </div>
  );
}
