
import { BehaviorSubject } from 'rxjs'; import { request } from '../api/client'; import type { DomainRecord, PageMeta } from '../types/domain';
export interface EntityState { items: DomainRecord[]; meta: PageMeta; loading: boolean; error: string }
const initialState: EntityState = { items: [], meta: { page: 1, pageSize: 20, total: 0 }, loading: false, error: '' };
export class EntityStore {
  private readonly subject = new BehaviorSubject<EntityState>(initialState); readonly state$ = this.subject.asObservable(); get snapshot() { return this.subject.value; }
  async load(path: string, search = '', page = 1, pageSize = 20) { this.patch({ loading: true, error: '' }); try { const result = await request<DomainRecord[]>(`/${path}?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`); this.subject.next({ items: result.data, meta: result.meta || { page, pageSize, total: result.data.length }, loading: false, error: '' }); } catch (error) { this.patch({ loading: false, error: error instanceof Error ? error.message : String(error) }); } }
  async createRecord(path: string, input: Partial<DomainRecord>) { this.patch({ loading: true }); try { await request<DomainRecord>(`/${path}`, { method: 'POST', body: JSON.stringify(input) }); await this.load(path); } catch (error) { this.patch({ loading: false, error: error instanceof Error ? error.message : String(error) }); throw error; } }
  async transition(path: string, item: DomainRecord, status: string) { this.patch({ loading: true }); try { await request<DomainRecord>(`/${path}/${item.id}/transition`, { method: 'POST', body: JSON.stringify({ status, expectedVersion: item.version, reason: '前端工作台人工确认' }) }); await this.load(path); } catch (error) { this.patch({ loading: false, error: error instanceof Error ? error.message : String(error) }); throw error; } }
  private patch(value: Partial<EntityState>) { this.subject.next({ ...this.subject.value, ...value }); }
}
