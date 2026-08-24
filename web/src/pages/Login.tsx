import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { LogIn } from 'lucide-react';
import { useBranding } from '../hooks/useBranding';

export function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const branding = useBranding();
  const companyName = branding.companyName;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(username, password);
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login fehlgeschlagen');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-dark flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8 flex flex-col items-center gap-2">
          <img
            src={branding.assets.stackedOnDark}
            alt={branding.productName}
            className="h-32 max-w-72 object-contain"
          />
          {(branding.brandName || companyName !== branding.productName) && <p className="text-gray-400 text-sm">by {branding.brandName || companyName}</p>}
        </div>

        <div className="glass-dark p-8 rounded-xl border border-white/10">
          <div className="flex items-center gap-3 mb-6">
            <LogIn className="w-6 h-6 text-accent-red" />
            <h2 className="text-2xl font-bold text-white">Anmelden</h2>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-500/10 border border-red-500/20 rounded-lg">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Benutzername</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                disabled={loading}
                className="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-accent-red focus:border-transparent disabled:opacity-50"
                placeholder="admin"
                autoComplete="username"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Passwort</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                disabled={loading}
                className="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-accent-red focus:border-transparent disabled:opacity-50"
                placeholder="••••••••"
                autoComplete="current-password"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-accent-red/50 text-white font-semibold rounded-lg transition-colors flex items-center justify-center gap-2"
            >
              {loading ? (
                <><div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />Anmelden...</>
              ) : (
                <><LogIn className="w-5 h-5" />Anmelden</>
              )}
            </button>
          </form>
        </div>

        <div className="mt-6 text-center">
          <p className="text-xs text-gray-500">{branding.productName} © {new Date().getFullYear()} {companyName}</p>
        </div>
      </div>
    </div>
  );
}
