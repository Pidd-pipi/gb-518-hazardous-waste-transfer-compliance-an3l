
import { CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { useAuth } from './hooks/use-auth';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, RouterOutlet, MatButtonModule],
  template: `
    <div *ngIf="auth.loading()" class="app-loading">正在建立安全会话…</div>
    <router-outlet *ngIf="!auth.loading() && !auth.authenticated()" />
    <div *ngIf="!auth.loading() && auth.authenticated()" class="app-shell">
      <aside>
        <div class="brand"><span>CONTROL DESK</span><strong>危险废物转运合规核验</strong></div>
        <nav><a *ngFor="let item of visibleNavigation" [routerLink]="item.to" routerLinkActive="active">{{ item.label }}</a></nav>
        <div class="user-panel">
          <span>{{ auth.session()?.displayName }}</span><small>{{ auth.session()?.role }}</small>
          <button mat-button (click)="logout()">退出登录</button>
        </div>
      </aside>
      <section class="content">
        <header class="topbar"><span>转运合规运行态势</span><span class="live-dot">服务已连接</span></header>
        <router-outlet />
      </section>
    </div>
  `
})
export class AppComponent implements OnInit {
  readonly auth = useAuth();
  readonly navigation = [
    { to: '/generators', label: '产废单位', reviewerOnly: false },
    { to: '/carriers', label: '承运资质', reviewerOnly: false },
    { to: '/manifests', label: '转运清单', reviewerOnly: false },
    { to: '/checks', label: '合规核验', reviewerOnly: false },
    { to: '/audit', label: '审计记录', reviewerOnly: true }
  ];

  constructor(private readonly changeDetector: ChangeDetectorRef, private readonly router: Router) {
    this.router.events.pipe(filter((event) => event instanceof NavigationEnd)).subscribe(() => {
      setTimeout(() => this.changeDetector.detectChanges());
    });
  }

  get visibleNavigation() {
    return this.navigation.filter((item) => !item.reviewerOnly || this.auth.hasMinimumRole('reviewer'));
  }

  async ngOnInit(): Promise<void> {
    await this.auth.restore();
    if (!this.auth.authenticated() && this.router.url !== '/login') await this.router.navigateByUrl('/login');
    this.changeDetector.detectChanges();
  }

  async logout(): Promise<void> {
    this.auth.logout();
    await this.router.navigateByUrl('/login');
    this.changeDetector.detectChanges();
  }
}
