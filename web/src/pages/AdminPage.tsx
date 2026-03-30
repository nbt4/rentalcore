import { useEffect, useState } from 'react';
import { Settings, Users, Shield } from 'lucide-react';
import { api } from '../lib/api';

interface AppUser {
  userid?: number;
  UserID?: number;
  username?: string;
  Username?: string;
  email?: string;
  Email?: string;
  is_active?: boolean;
  IsActive?: boolean;
}

export function AdminPage() {
  const [users, setUsers] = useState<AppUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'users' | 'roles'>('users');

  useEffect(() => {
    api.get('/security/auth/users')
      .then((r) => {
        const data = r.data as { users?: AppUser[] } | AppUser[];
        setUsers(Array.isArray(data) ? data : data.users || []);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const tabs = [
    { key: 'users' as const, label: 'Benutzer', icon: Users },
    { key: 'roles' as const, label: 'Rollen', icon: Shield },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Settings className="w-6 h-6 text-accent-red" /> Administration</h1>
        <p className="text-gray-400 text-sm mt-1">Systemeinstellungen und Benutzerverwaltung</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2">
        {tabs.map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              activeTab === key
                ? 'bg-accent-red text-white'
                : 'bg-white/10 text-gray-400 hover:bg-white/15 hover:text-white'
            }`}
          >
            <Icon className="w-4 h-4" />{label}
          </button>
        ))}
      </div>

      {activeTab === 'users' && (
        <div className="glass-dark rounded-xl border border-white/10">
          <div className="px-6 py-4 border-b border-white/10 flex items-center justify-between">
            <h2 className="font-semibold">Benutzer</h2>
            <span className="text-sm text-gray-400">{users.length} Benutzer</span>
          </div>
          {loading ? (
            <div className="flex justify-center py-16"><div className="w-8 h-8 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin" /></div>
          ) : (
            <div className="divide-y divide-white/5">
              {users.map((u) => {
                const id = u.userid ?? u.UserID ?? 0;
                const name = u.username ?? u.Username ?? '—';
                const email = u.email ?? u.Email ?? '—';
                const active = u.is_active ?? u.IsActive ?? false;
                return (
                  <div key={id} className="flex items-center justify-between px-6 py-4 hover:bg-white/5 transition-colors">
                    <div>
                      <div className="font-medium text-white">{name}</div>
                      <div className="text-sm text-gray-400">{email}</div>
                    </div>
                    <span className={`px-2 py-0.5 rounded-full text-xs border ${active ? 'bg-green-500/10 text-green-400 border-green-500/20' : 'bg-gray-500/10 text-gray-400 border-gray-500/20'}`}>
                      {active ? 'Aktiv' : 'Inaktiv'}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {activeTab === 'roles' && (
        <div className="glass-dark rounded-xl border border-white/10 p-8 text-center text-gray-500">
          <Shield className="w-10 h-10 mx-auto mb-3 opacity-30" />
          <p>Rollenverwaltung in Kürze verfügbar</p>
        </div>
      )}
    </div>
  );
}
