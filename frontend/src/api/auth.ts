
import { request } from './client';
import type { UserSession } from '../types/domain';
export async function login(username: string, password: string): Promise<UserSession> {
  return (await request<UserSession>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) })).data;
}
export async function currentSession(): Promise<Omit<UserSession, 'token' | 'expiresIn'>> {
	return (await request<Omit<UserSession, 'token' | 'expiresIn'>>('/session')).data;
}
