
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listCarrierProfile(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/carriers?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createCarrierProfile(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/carriers', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionCarrierProfile(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/carriers/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
