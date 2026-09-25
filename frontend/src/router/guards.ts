import { inject } from '@angular/core';
import type { CanActivateFn } from '@angular/router';
import { Router } from '@angular/router';
import { AuthService } from '../hooks/use-auth';

export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  return auth.authenticated() || inject(Router).createUrlTree(['/login']);
};

export const reviewerGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  return auth.hasMinimumRole('reviewer') || inject(Router).createUrlTree(['/generators']);
};

export const loginGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  return !auth.authenticated() || inject(Router).createUrlTree(['/generators']);
};
