import { CommonModule } from '@angular/common';
import { ApplicationRef, ChangeDetectorRef, Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { Router } from '@angular/router';
import { useAuth } from '../hooks/use-auth';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule],
  template: `
    <main class="login-page">
      <section class="login-panel" aria-labelledby="login-title">
        <p class="eyebrow">CONTROL DESK</p>
        <h1 id="login-title">危险废物转运合规核验</h1>
        <p class="login-intro">进入许可、联单和核验工作台</p>
        <form (ngSubmit)="submit()">
          <label>账号<input name="username" [(ngModel)]="username" autocomplete="username" required /></label>
          <label>密码<input name="password" [(ngModel)]="password" type="password" autocomplete="current-password" required /></label>
          <div *ngIf="error" class="alert" role="alert">{{ error }}</div>
          <button mat-flat-button color="primary" type="submit" [disabled]="auth.loading() || !username || !password">
            {{ auth.loading() ? '正在验证…' : '登录工作台' }}
          </button>
        </form>
      </section>
    </main>
  `
})
export class LoginPage {
  readonly auth = useAuth();
  username = '';
  password = '';
  error = '';

  constructor(private readonly router: Router, private readonly changeDetector: ChangeDetectorRef, private readonly application: ApplicationRef) {}

  async submit(): Promise<void> {
    this.error = '';
    try {
      await this.auth.login(this.username, this.password);
	  this.application.tick();
      await this.router.navigateByUrl('/generators');
	  this.application.tick();
    } catch {
      this.error = '账号或密码不正确';
    } finally {
      this.changeDetector.detectChanges();
    }
  }
}
