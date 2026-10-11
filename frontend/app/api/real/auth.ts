import type { User, TokenResponse } from '~/api/types';

// TODO(real-api): POST /api/auth/login
export async function loginReal(_email: string, _password?: string): Promise<TokenResponse> {
  throw new Error('Real API: login is not implemented');
}

// TODO(real-api): POST /api/auth/register
export async function registerReal(_name: string, _email: string, _password?: string): Promise<TokenResponse> {
  throw new Error('Real API: register is not implemented');
}

// TODO(real-api): POST /api/auth/logout
export async function logoutReal(): Promise<void> {
  throw new Error('Real API: logout is not implemented');
}

// TODO(real-api): GET /api/user/me
export async function getCurrentUserReal(): Promise<User> {
  throw new Error('Real API: getCurrentUser is not implemented');
}

// TODO(real-api): PATCH /api/user/name
export async function updateUserNameReal(_name: string): Promise<User> {
  throw new Error('Real API: updateUserName is not implemented');
}
