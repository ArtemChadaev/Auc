import { useState, useEffect, useCallback } from 'react';
import { getCurrentUser, login as apiLogin, register as apiRegister, logout as apiLogout, topUpDailyCoins } from '~/api';
import type { User } from '~/api';
import { useUserStream } from '~/realtime/useUserStream';

export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refreshUser = useCallback(async () => {
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

  const login = async (email: string, password?: string) => {
    setLoading(true);
    try {
      await apiLogin(email, password);
      await refreshUser();
    } finally {
      setLoading(false);
    }
  };

  const register = async (name: string, email: string, avatarId?: string, password?: string) => {
    setLoading(true);
    try {
      await apiRegister(name, email, avatarId, password);
      await refreshUser();
    } finally {
      setLoading(false);
    }
  };

  const logout = async () => {
    setLoading(true);
    try {
      await apiLogout();
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  const claimDailyCoins = async () => {
    const res = await topUpDailyCoins();
    setUser(res.user);
    return res;
  };

  return {
    user,
    loading,
    error,
    login,
    register,
    logout,
    refreshUser,
    claimDailyCoins,
    isAuthenticated: !!user,
  };
}
