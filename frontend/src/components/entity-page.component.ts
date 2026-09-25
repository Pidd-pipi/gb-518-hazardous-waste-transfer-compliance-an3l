import { AsyncPipe, CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { useAuth } from '../hooks/use-auth';
import { createPagination } from '../hooks/use-pagination';
import type { EntityStore } from '../stores/factory';
import type { DomainRecord, EntityConfig } from '../types/domain';
import { TRANSITIONS } from '../types/status';
import { formatDate } from '../utils/format';
import { ConfirmDialogComponent } from './common/confirm-dialog.component';
import { LicensePanelComponent } from './common/license-panel.component';
import { MetricCardComponent } from './common/metric-card.component';
import { StatusBadgeComponent } from './common/status-badge.component';

@Component({
  selector: 'app-entity-page',
  standalone: true,
  imports: [CommonModule, AsyncPipe, FormsModule, MatButtonModule, MatInputModule, StatusBadgeComponent, MetricCardComponent, ConfirmDialogComponent, LicensePanelComponent],
  template: `
    <main class="workspace" *ngIf="store.state$ | async as state">
      <header class="page-header">
        <div><p class="eyebrow">业务工作台</p><h1>{{ config.label }}</h1><p>{{ pageDescription() }}</p></div>
        <button *ngIf="auth.hasMinimumRole('operator')" mat-flat-button color="primary" (click)="openCreate()">新增{{ config.label }}</button>
      </header>

      <app-license-panel *ngIf="isLicensePage()" [records]="state.items" />

      <section class="metrics">
        <app-metric-card label="记录总数" [value]="state.meta.total" detail="当前查询结果" />
        <app-metric-card label="高风险" [value]="highRisk(state.items)" detail="需要优先复核" />
        <app-metric-card label="状态种类" [value]="statusCount(state.items)" detail="当前页状态覆盖" />
      </section>

      <section class="toolbar">
        <input matInput aria-label="搜索" [(ngModel)]="search" [placeholder]="'搜索' + config.label + '编码或名称'" (keyup.enter)="query()" />
        <button mat-flat-button color="primary" (click)="query()">查询</button>
        <button mat-button (click)="reset()">重置</button>
      </section>
      <div *ngIf="state.error" class="alert" role="alert">{{ state.error }}</div>

      <section class="table-shell">
        <table>
          <thead><tr><th>编码</th><th>名称</th><th>状态</th><th>业务凭证</th><th>风险</th><th>责任人</th><th>指标</th><th>更新时间</th><th>操作</th></tr></thead>
          <tbody>
            <tr *ngFor="let item of state.items; trackBy: trackById">
              <td><strong>{{ item.code }}</strong></td>
              <td>{{ item.name }}<small>{{ item.facility }}</small></td>
              <td><app-status-badge [status]="item.status" /></td>
              <td><span class="domain-detail">{{ domainDetail(item) }}</span><small>{{ item.evidence }}</small></td>
              <td><span [class]="'risk risk--' + item.riskLevel">{{ item.riskLevel }}</span></td>
              <td>{{ item.owner }}</td>
              <td>{{ item.metricValue }} {{ item.metricUnit }}</td>
              <td>{{ formatDate(item.updatedAt) }}</td>
              <td class="actions">
                <ng-container *ngIf="canTransition()">
                  <button *ngFor="let target of transitions(item)" class="table-action" (click)="openTransition(item, target)">{{ transitionLabel(target) }}</button>
                </ng-container>
                <span *ngIf="!canTransition() || transitions(item).length === 0" class="muted">{{ auth.hasMinimumRole('operator') ? '流程结束' : '只读' }}</span>
              </td>
            </tr>
            <tr *ngIf="!state.items.length && !state.loading"><td colspan="9" class="empty">暂无记录</td></tr>
          </tbody>
        </table>
        <div *ngIf="state.loading" class="loading">正在同步业务数据…</div>
      </section>

      <footer class="pager" *ngIf="state.meta.total > state.meta.pageSize">
        <span>第 {{ state.meta.page }} / {{ pagination.pages() }} 页</span>
        <button mat-button [disabled]="pagination.page() <= 1" (click)="previousPage()">上一页</button>
        <button mat-button [disabled]="pagination.page() >= pagination.pages()" (click)="nextPage()">下一页</button>
      </footer>

      <app-confirm-dialog [open]="showCreate" [title]="'新增' + config.label" (cancel)="closeCreate()" (confirm)="createDemo()">
        <p>确认创建一条包含责任人、业务关联、风险和证据信息的记录。</p>
      </app-confirm-dialog>
      <app-confirm-dialog [open]="!!pending" title="确认状态迁移" (cancel)="closeTransition()" (confirm)="confirmTransition()">
        <p>状态迁移会校验关联资质，并与请求 ID 审计记录在同一事务中保存。</p>
        <strong>{{ pending?.item?.status }} → {{ pending?.status }}</strong>
      </app-confirm-dialog>
    </main>
  `
})
export class EntityPageComponent implements OnInit {
  @Input({ required: true }) config!: EntityConfig;
  @Input({ required: true }) store!: EntityStore;
  readonly auth = useAuth();
  readonly formatDate = formatDate;
  readonly pagination = createPagination(() => this.store?.snapshot.meta.total ?? 0, 10);
  search = '';
  showCreate = false;
  pending: { item: DomainRecord; status: string } | null = null;

  constructor(private readonly changeDetector: ChangeDetectorRef) {}

  async ngOnInit(): Promise<void> { await this.load(); }
  trackById(_index: number, item: DomainRecord): number { return item.id; }
  highRisk(items: DomainRecord[]): number { return items.filter((item) => ['high', 'critical'].includes(item.riskLevel)).length; }
  statusCount(items: DomainRecord[]): number { return new Set(items.map((item) => item.status)).size; }
  isLicensePage(): boolean { return this.config.key === 'wasteGenerator' || this.config.key === 'carrierProfile'; }
  canTransition(): boolean { return this.auth.hasMinimumRole(this.config.transitionRole); }
  transitions(item: DomainRecord): readonly string[] { return TRANSITIONS[this.config.key]?.[item.status] ?? []; }

  pageDescription(): string {
    const descriptions: Record<string, string> = {
      wasteGenerator: '核对产废许可有效期、废物类别与证据文件。',
      carrierProfile: '复核承运许可证、有效车辆和资质证据。',
      transferManifest: '关联产废单位与承运方，跟踪联单全流程。',
      complianceCheck: '基于联单证据形成不可回退的核验决定。'
    };
    return descriptions[this.config.key] || `管理${this.config.label}状态和证据。`;
  }

  domainDetail(item: DomainRecord): string {
    if (this.config.key === 'wasteGenerator') return `${item.permitNumber || '-'} · ${item.wasteCategories || '-'}`;
    if (this.config.key === 'carrierProfile') return `${item.licenseNumber || '-'} · ${item.vehicleCount || 0} 辆`;
    if (this.config.key === 'transferManifest') return `${item.generatorCode} → ${item.carrierCode} · ${item.quantityKg} kg`;
    return `${item.manifestCode || '-'} · ${item.decisionBasis || '待决定'}`;
  }

  transitionLabel(status: string): string {
    const labels: Record<string, string> = { submitted: '提交', in_transit: '发运', received: '签收', rejected: '驳回', verified: '核准', restricted: '限制', expired: '到期', active: '恢复', suspended: '停用', pass: '通过', fail: '不通过', escalated: '升级复核' };
    return labels[status] || status;
  }

  async query(): Promise<void> { this.pagination.reset(); await this.load(); }
  async reset(): Promise<void> { this.search = ''; this.pagination.reset(); await this.load(); }
  async previousPage(): Promise<void> { this.pagination.previous(); await this.load(); }
  async nextPage(): Promise<void> { this.pagination.next(); await this.load(); }
  openCreate(): void { this.showCreate = true; this.changeDetector.detectChanges(); }
  closeCreate(): void { this.showCreate = false; this.changeDetector.detectChanges(); }
  openTransition(item: DomainRecord, status: string): void { this.pending = { item, status }; this.changeDetector.detectChanges(); }
  closeTransition(): void { this.pending = null; this.changeDetector.detectChanges(); }

  async createDemo(): Promise<void> {
    const now = Date.now();
    const common: Partial<DomainRecord> = {
      code: `${this.config.key.toUpperCase()}-${String(now).slice(-6)}`,
      name: `新增${this.config.label}`,
      description: '通过合规工作台创建的业务记录', facility: '东区危废暂存区', owner: '现场操作员',
      category: '危废转运', riskLevel: 'medium', metricValue: 25, metricUnit: 'score',
      effectiveAt: new Date().toISOString(), evidence: `minio://evidence/${this.config.path}/${now}.pdf`, relatedCode: ''
    };
    const expiresAt = new Date(now + 365 * 86_400_000).toISOString();
    const specific: Partial<DomainRecord> = this.config.key === 'wasteGenerator'
      ? { permitNumber: `PERMIT-${String(now).slice(-8)}`, permitExpiresAt: expiresAt, wasteCategories: 'HW08 废矿物油' }
      : this.config.key === 'carrierProfile'
        ? { licenseNumber: `CARRIER-${String(now).slice(-8)}`, licenseExpiresAt: expiresAt, vehicleCount: 6 }
        : this.config.key === 'transferManifest'
          ? { generatorCode: 'WG-001', carrierCode: 'CP-002', wasteCode: 'HW08-900-249-08', quantityKg: 640, destination: '合规处置中心 A' }
          : { manifestCode: 'TM-002', checklist: '产废许可、承运资质、联单数量、处置去向', decisionBasis: '' };
    try {
      await this.store.createRecord(this.config.path, { ...common, ...specific });
      this.showCreate = false;
    } catch { /* Store exposes the request error in its observable state. */ }
    finally { this.changeDetector.detectChanges(); }
  }

  async confirmTransition(): Promise<void> {
    if (!this.pending) return;
    try {
      await this.store.transition(this.config.path, this.pending.item, this.pending.status);
      this.pending = null;
    } catch { /* Store exposes the request error in its observable state. */ }
    finally { this.changeDetector.detectChanges(); }
  }

  private async load(): Promise<void> {
    await this.store.load(this.config.path, this.search, this.pagination.page(), this.pagination.pageSize());
    this.changeDetector.detectChanges();
  }
}
