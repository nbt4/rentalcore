import type { ReactNode } from 'react';
import { useAuth } from '../contexts/AuthContext';

interface RoleGuardProps {
  requiredRoles: string[];
  children: ReactNode;
}

export function RoleGuard({ requiredRoles, children }: RoleGuardProps) {
  const { user, loading } = useAuth();

  if (loading) return <div className="min-h-[60vh] flex items-center justify-center text-gray-400">Lade...</div>;

  const roles = (user?.Roles ?? user?.roles ?? []) as { name?: string; Name?: string }[];
  const hasRole = roles.some((r) => {
    const name = (r?.name || r?.Name || '').toLowerCase();
    return requiredRoles.some((req) => req.toLowerCase() === name);
  });

  if (!hasRole) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center text-center space-y-3">
        <div className="text-5xl font-bold text-accent-red">403</div>
        <p className="text-lg text-gray-300 font-semibold">Zugriff verweigert</p>
        <p className="text-sm text-gray-500 max-w-md">
          Für diesen Bereich werden erweiterte Rechte benötigt.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
