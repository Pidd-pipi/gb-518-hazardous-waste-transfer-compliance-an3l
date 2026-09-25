import type { Routes } from '@angular/router';
import { WasteGeneratorPage } from '../pages/waste-generator.page';
import { CarrierProfilePage } from '../pages/carrier-profile.page';
import { TransferManifestPage } from '../pages/transfer-manifest.page';
import { ComplianceCheckPage } from '../pages/compliance-check.page';
import { AuditPage } from '../pages/audit.page';
import { LoginPage } from '../pages/login.page';
import { authGuard, loginGuard, reviewerGuard } from './guards';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'generators' },
  { path: 'login', component: LoginPage, canActivate: [loginGuard] },
  { path: 'generators', component: WasteGeneratorPage, canActivate: [authGuard] },
  { path: 'carriers', component: CarrierProfilePage, canActivate: [authGuard] },
  { path: 'manifests', component: TransferManifestPage, canActivate: [authGuard] },
  { path: 'checks', component: ComplianceCheckPage, canActivate: [authGuard] },
  { path: 'audit', component: AuditPage, canActivate: [authGuard, reviewerGuard] },
  { path: '**', redirectTo: 'generators' }
];
