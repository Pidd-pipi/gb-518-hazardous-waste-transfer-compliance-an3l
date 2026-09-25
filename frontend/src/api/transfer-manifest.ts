
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export async function listTransferManifest(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/manifests?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createTransferManifest(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/manifests', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionTransferManifest(id: number, status: string, expectedVersion: number, reason: string) {
  return request<DomainRecord>(`/manifests/${id}/transition`, {
    method: 'POST', body: JSON.stringify({ status, expectedVersion, reason }),
  });
}
