import { appPath } from '../lib/app-paths';

const API_BASE = appPath('/api/v1');

export interface Role {
  id: number;
  name: string;
  display_name?: string;
}

export interface User {
  UserID: number;
  Username: string;
  Email: string;
  FirstName: string;
  LastName: string;
  IsActive: boolean;
  Roles?: Role[];
  roles?: Role[];
  force_password_change?: boolean;
}

class AuthService {
  async login(username: string, password: string): Promise<{ user: User; forcePasswordChange: boolean }> {
    const response = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Login fehlgeschlagen' }));
      throw new Error(error.message || 'Login fehlgeschlagen');
    }

    const data = await response.json();
    if (!data.success || !data.user) {
      throw new Error(data.message || 'Login fehlgeschlagen');
    }

    return { user: data.user, forcePasswordChange: data.force_password_change || false };
  }

  async logout(): Promise<void> {
    await fetch(`${API_BASE}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });
  }

  async getCurrentUser(): Promise<User | null> {
    try {
      const response = await fetch(`${API_BASE}/auth/me`, { credentials: 'include' });
      if (response.status === 401) return null;
      if (!response.ok) return null;
      return await response.json();
    } catch {
      return null;
    }
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const response = await fetch(`${API_BASE}/auth/change-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Passwortänderung fehlgeschlagen' }));
      throw new Error(error.message || 'Passwortänderung fehlgeschlagen');
    }
  }
}

export const authService = new AuthService();
