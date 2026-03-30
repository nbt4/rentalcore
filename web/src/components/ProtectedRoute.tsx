import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

interface ProtectedRouteProps {
  children: React.ReactNode;
  bypassForcePasswordChange?: boolean;
}

export function ProtectedRoute({ children, bypassForcePasswordChange = false }: ProtectedRouteProps) {
  const { isAuthenticated, loading, forcePasswordChange } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen bg-dark flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-accent-red/20 border-t-accent-red rounded-full animate-spin mx-auto mb-4" />
          <p className="text-gray-400">Lade...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (forcePasswordChange && !bypassForcePasswordChange) return <Navigate to="/change-password" replace />;

  return <>{children}</>;
}
