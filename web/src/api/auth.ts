import client from './client';
import type { AuthResponse, RefreshResponse, User } from '../types';

export async function login(username: string, password: string): Promise<AuthResponse> {
  const { data } = await client.post<AuthResponse>('/auth/login', { username, password });
  return data;
}

export async function refresh(refreshToken: string): Promise<RefreshResponse> {
  const { data } = await client.post<RefreshResponse>('/auth/refresh', { refresh_token: refreshToken });
  return data;
}

export async function logout(refreshToken: string): Promise<void> {
  await client.post('/auth/logout', { refresh_token: refreshToken });
}

export async function getMe(): Promise<User> {
  const { data } = await client.get<User>('/auth/me');
  return data;
}
