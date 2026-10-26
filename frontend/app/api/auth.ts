import {
  loginReal,
  registerReal,
  logoutReal,
  getCurrentUserReal,
  updateUserNameReal,
  deleteCurrentUserReal,
  updateAvatarReal,
  getSessionsReal,
  getFreeCoinReal,
} from './real/auth';
import type { User, TokenResponse, UserSession } from './types';

const USE_MOCKS = false;

export async function login(email: string, password?: string): Promise<TokenResponse> {
  return loginReal(email, password);
}

export async function register(
  name: string,
  email: string,
  _avatarId?: string,
  password?: string
): Promise<TokenResponse> {
  return registerReal(name, email, password);
}

export async function logout(): Promise<void> {
  return logoutReal();
}

export async function getCurrentUser(): Promise<User> {
  return getCurrentUserReal();
}

export async function updateUserName(name: string): Promise<User> {
  return updateUserNameReal(name);
}

export async function deleteCurrentUser(): Promise<void> {
  return deleteCurrentUserReal();
}

export async function uploadAvatar(file: File | Blob): Promise<{ image: string }> {
  return updateAvatarReal(file);
}

export async function getSessions(current?: boolean): Promise<UserSession[]> {
  return getSessionsReal(current);
}

export async function getFreeCoin(): Promise<User> {
  return getFreeCoinReal();
}
