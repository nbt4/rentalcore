import { useState } from 'react';
import { User, Lock } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';

export function ProfilePage() {
  const { user, changePassword } = useAuth();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [saving, setSaving] = useState(false);

  const handlePwChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (next !== confirm) { setError('Passwörter stimmen nicht überein'); return; }
    if (next.length < 6) { setError('Mindestens 6 Zeichen'); return; }
    setError(''); setSuccess(''); setSaving(true);
    try {
      await changePassword(current, next);
      setSuccess('Passwort erfolgreich geändert');
      setCurrent(''); setNext(''); setConfirm('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Fehler');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6 max-w-xl">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><User className="w-6 h-6 text-accent-red" /> Profil</h1>
      </div>

      <div className="glass-dark rounded-xl border border-white/10 p-6">
        <h2 className="font-semibold mb-4">Accountdaten</h2>
        <div className="space-y-2 text-sm">
          {[
            { label: 'Benutzername', value: user?.Username },
            { label: 'E-Mail', value: user?.Email },
            { label: 'Vorname', value: user?.FirstName },
            { label: 'Nachname', value: user?.LastName },
          ].map(({ label, value }) => (
            <div key={label} className="flex justify-between py-1.5 border-b border-white/5 last:border-0">
              <span className="text-gray-400">{label}</span>
              <span className="text-white">{value || '—'}</span>
            </div>
          ))}
          <div className="flex justify-between py-1.5">
            <span className="text-gray-400">Rollen</span>
            <div className="flex gap-1 flex-wrap justify-end">
              {((user?.Roles ?? user?.roles ?? []) as { name?: string; Name?: string }[]).map((r, i) => (
                <span key={i} className="px-2 py-0.5 bg-accent-red/10 text-accent-red rounded-full text-xs border border-accent-red/20">
                  {r?.name || r?.Name}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="glass-dark rounded-xl border border-white/10 p-6">
        <h2 className="font-semibold mb-4 flex items-center gap-2"><Lock className="w-4 h-4 text-accent-red" /> Passwort ändern</h2>
        {error && <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>}
        {success && <div className="mb-4 p-3 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-sm">{success}</div>}
        <form onSubmit={handlePwChange} className="space-y-4">
          {[
            { label: 'Aktuelles Passwort', val: current, set: setCurrent },
            { label: 'Neues Passwort', val: next, set: setNext },
            { label: 'Bestätigen', val: confirm, set: setConfirm },
          ].map(({ label, val, set }) => (
            <div key={label}>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">{label}</label>
              <input
                type="password"
                value={val}
                onChange={(e) => set(e.target.value)}
                required
                className="w-full px-3 py-2.5 rounded-lg"
              />
            </div>
          ))}
          <button type="submit" disabled={saving} className="px-6 py-2.5 bg-accent-red hover:bg-accent-red/80 disabled:opacity-50 text-white rounded-lg font-medium transition-colors">
            {saving ? 'Wird gespeichert...' : 'Passwort ändern'}
          </button>
        </form>
      </div>
    </div>
  );
}
