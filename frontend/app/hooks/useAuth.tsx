import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react';
import { getCurrentUser, login as apiLogin, register as apiRegister, logout as apiLogout, topUpDailyCoins, getFreeCoin } from '~/api';
import type { User } from '~/api';
import { useUserStream } from '~/realtime/useUserStream';
import { getStoredToken } from '~/api/client';

export interface AuthContextType {
  user: User | null;
  loading: boolean;
  error: string | null;
  login: (email: string, password?: string) => Promise<void>;
  register: (name: string, email: string, avatarId?: string, password?: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
  claimDailyCoins: () => Promise<any>;
  getFreeCoin: () => Promise<User>;
  isAuthenticated: boolean;
  isDailyClaimed: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

function getTodayString(): string {
  return new Date().toISOString().slice(0, 10);
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dailyClaimed, setDailyClaimed] = useState(false);

  // Проверка статуса ежедневного бонуса для текущего пользователя
  useEffect(() => {
    if (user?.id && typeof window !== 'undefined') {
      const today = getTodayString();
      const claimed = localStorage.getItem(`auc_daily_claimed_${user.id}_${today}`);
      setDailyClaimed(claimed === 'true');
    } else {
      setDailyClaimed(false);
    }
  }, [user?.id]);

  const refreshUser = useCallback(async () => {
    const token = getStoredToken();
    if (!token) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const u = await getCurrentUser();
      setUser(u);
      setError(null);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshUser();
  }, [refreshUser]);

  // Слушаем события обновления баланса из user stream
  useUserStream((event) => {
    if (event.type === 'balance_changed' && event.data.balance !== undefined && event.data.hold !== undefined) {
      setUser((prev) => {
        if (!prev) return null;
        return {
          ...prev,
          balance: event.data.balance!,
          hold: event.data.hold!,
        };
      });
    }
  });

  const login = useCallback(async (email: string, password?: string) => {
    setLoading(true);
    try {
      await apiLogin(email, password);
      const u = await getCurrentUser();
      setUser(u);
      setError(null);
    } catch (err: any) {
      setError(err?.message || 'Ошибка авторизации');
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const register = useCallback(async (name: string, email: string, avatarId?: string, password?: string) => {
    setLoading(true);
    try {
      await apiRegister(name, email, avatarId, password);
      const u = await getCurrentUser();
      setUser(u);
      setError(null);
    } catch (err: any) {
      setError(err?.message || 'Ошибка регистрации');
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    setLoading(true);
    try {
      await apiLogout();
    } finally {
      setUser(null);
      setDailyClaimed(false);
      setLoading(false);
    }
  }, []);

  const claimDailyCoins = useCallback(async () => {
    try {
      const updatedUser = await getFreeCoin();
      setUser(updatedUser);
      setDailyClaimed(true);
      if (updatedUser?.id && typeof window !== 'undefined') {
        const today = getTodayString();
        localStorage.setItem(`auc_daily_claimed_${updatedUser.id}_${today}`, 'true');
      }
      return { user: updatedUser, added: true };
    } catch (err: any) {
      const errMsg = (err?.message || '').toLowerCase();
      if (
        err?.status === 429 ||
        errMsg.includes('limit') ||
        errMsg.includes('once per day') ||
        errMsg.includes('лимит')
      ) {
        setDailyClaimed(true);
        if (user?.id && typeof window !== 'undefined') {
          const today = getTodayString();
          localStorage.setItem(`auc_daily_claimed_${user.id}_${today}`, 'true');
        }
        throw new Error('Бесплатные монеты можно получить только один раз в день (сброс в 00:00 UTC)');
      }
      throw err;
    }
  }, [user]);

  const value = useMemo<AuthContextType>(
    () => ({
      user,
      loading,
      error,
      login,
      register,
      logout,
      refreshUser,
      claimDailyCoins,
      getFreeCoin: async () => {
        const u = await getFreeCoin();
        setUser(u);
        return u;
      },
      isAuthenticated: !!user,
      isDailyClaimed: dailyClaimed,
    }),
    [user, loading, error, login, register, logout, refreshUser, claimDailyCoins, dailyClaimed]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
