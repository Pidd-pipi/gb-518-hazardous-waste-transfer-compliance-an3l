
import { Injectable, computed, inject, signal } from '@angular/core';
import { currentSession, login as requestLogin } from '../api/auth';
import { clearSession, readSession, saveSession } from '../api/client';
import type { UserRole, UserSession } from '../types/domain';

const roleRank: Record<UserRole, number> = { viewer: 1, operator: 2, reviewer: 3, admin: 4 };

@Injectable({ providedIn: 'root' })
export class AuthService {
  readonly session = signal<UserSession | null>(readSession());
  readonly loading = signal(false);
  readonly authenticated = computed(() => Boolean(this.session()?.token));

  hasMinimumRole(minimum: UserRole): boolean {
    const role = this.session()?.role;
    return Boolean(role && roleRank[role] >= roleRank[minimum]);
  }

  async restore(): Promise<void> {
    const cached = readSession();
    if (!cached) {
      this.session.set(null);
      return;
    }
    this.loading.set(true);
    try {
      const live = await currentSession();
      const next: UserSession = { ...cached, ...live };
      saveSession(next);
      this.session.set(next);
    } catch {
      this.logout();
    } finally {
      this.loading.set(false);
    }
  }

  async login(username: string, password: string): Promise<void> {
    this.loading.set(true);
    try {
      const next = await requestLogin(username.trim(), password);
      saveSession(next);
      this.session.set(next);
    } finally {
      this.loading.set(false);
    }
  }

  logout(): void {
    clearSession();
    this.session.set(null);
  }
}

export function useAuth(): AuthService {
  return inject(AuthService);
}
