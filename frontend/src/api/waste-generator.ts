
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listWasteGenerator(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/generators?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createWasteGenerator(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/generators', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionWasteGenerator(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/generators/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
