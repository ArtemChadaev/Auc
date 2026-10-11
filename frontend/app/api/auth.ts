import { mockDb } from './mocks/db';
import { simulateLatency } from './mocks/latency';
import { loginReal, registerReal, logoutReal, getCurrentUserReal, updateUserNameReal } from './real/auth';
import type { User, TokenResponse } from './types';

const USE_MOCKS = true;

export async function login(email: string, password?: string): Promise<TokenResponse> {
  if (USE_MOCKS) {
    await simulateLatency();
    const user = mockDb.login(email);
    return {
      access_token: `mock-token-${user.id}`,
      refresh_session: {
        refresh_id: 1,
        created_at: new Date().toISOString(),
        expires_at: new Date(Date.now() + 86400000 * 30).toISOString(),
      },
    };
  }
  return loginReal(email, password);
}

export async function register(name: string, email: string, avatarId?: string, password?: string): Promise<TokenResponse> {
  if (USE_MOCKS) {
    await simulateLatency();
    const user = mockDb.register(name, email, avatarId);
    return {
      access_token: `mock-token-${user.id}`,
      refresh_session: {
        refresh_id: 1,
        created_at: new Date().toISOString(),
        expires_at: new Date(Date.now() + 86400000 * 30).toISOString(),
      },
    };
  }
  return registerReal(name, email, password);
}

export async function logout(): Promise<void> {
  if (USE_MOCKS) {
    await simulateLatency();
    return;
  }
  return logoutReal();
}

export async function getCurrentUser(): Promise<User> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.getCurrentUser();
  }
  return getCurrentUserReal();
}

export async function updateUserName(name: string): Promise<User> {
  if (USE_MOCKS) {
    await simulateLatency();
    return mockDb.updateUserName(name);
  }
  return updateUserNameReal(name);
}
