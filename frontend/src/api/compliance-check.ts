
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listComplianceCheck(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/checks?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createComplianceCheck(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/checks', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionComplianceCheck(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/checks/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
