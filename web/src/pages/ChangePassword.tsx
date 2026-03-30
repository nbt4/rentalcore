import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Lock } from 'lucide-react';

export function ChangePassword() {
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { changePassword } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (next !== confirm) { setError('Passwörter stimmen nicht überein'); return; }
    if (next.length < 6) { setError('Passwort muss mindestens 6 Zeichen haben'); return; }
    setError('');
    setLoading(true);
    try {
      await changePassword(current, next);
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Fehler beim Ändern des Passworts');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-dark flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-2">
            <span className="text-accent-red">Rental</span><span className="text-white">Core</span>
          </h1>
        </div>
        <div className="glass-dark p-8 rounded-xl border border-white/10">
          <div className="flex items-center gap-3 mb-2">
            <Lock className="w-6 h-6 text-accent-red" />
            <h2 className="text-2xl font-bold">Passwort ändern</h2>
          </div>
          <p className="text-gray-400 text-sm mb-6">Bitte wähle ein neues Passwort für deinen Account.</p>

          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {[
              { label: 'Aktuelles Passwort', val: current, set: setCurrent, ph: '••••••••' },
              { label: 'Neues Passwort', val: next, set: setNext, ph: 'Mindestens 6 Zeichen' },
              { label: 'Passwort bestätigen', val: confirm, set: setConfirm, ph: '••••••••' },
            ].map(({ label, val, set, ph }) => (
              <div key={label}>
                <label className="block text-sm font-medium text-gray-300 mb-2">{label}</label>
                <input
                  type="password"
                  value={val}
                  onChange={(e) => set(e.target.value)}
                  required
                  disabled={loading}
                  placeholder={ph}
                  className="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-accent-red disabled:opacity-50"
                />
              </div>
            ))}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-accent-red/50 text-white font-semibold rounded-lg transition-colors flex items-center justify-center gap-2 mt-2"
            >
              {loading ? <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" /> : <Lock className="w-5 h-5" />}
              {loading ? 'Wird gespeichert...' : 'Passwort speichern'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
