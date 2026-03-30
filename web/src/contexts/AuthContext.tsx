import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { authService } from '../services/auth';
import type { User } from '../services/auth';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  forcePasswordChange: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [forcePasswordChange, setForcePasswordChange] = useState(false);

  useEffect(() => {
    authService.getCurrentUser().then((u) => {
      setUser(u);
      if (u?.force_password_change) setForcePasswordChange(true);
    }).finally(() => setLoading(false));
  }, []);

  const login = async (username: string, password: string) => {
    const result = await authService.login(username, password);
    setUser(result.user);
    setForcePasswordChange(result.forcePasswordChange);
  };

  const logout = async () => {
    await authService.logout();
    setUser(null);
    setForcePasswordChange(false);
  };

  const changePassword = async (currentPassword: string, newPassword: string) => {
    await authService.changePassword(currentPassword, newPassword);
    setForcePasswordChange(false);
  };

  return (
    <AuthContext.Provider value={{
      user, loading, forcePasswordChange,
      login, logout, changePassword,
      isAuthenticated: user !== null,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
