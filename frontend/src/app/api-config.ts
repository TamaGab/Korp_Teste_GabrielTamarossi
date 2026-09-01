import { InjectionToken } from '@angular/core';

declare const __INVENTORY_API_URL__: string;
declare const __BILLING_API_URL__: string;

export const INVENTORY_API_URL = new InjectionToken<string>('INVENTORY_API_URL', {
  providedIn: 'root',
  factory: () =>
    typeof __INVENTORY_API_URL__ === 'undefined'
      ? 'http://localhost:8081'
      : __INVENTORY_API_URL__,
});

export const BILLING_API_URL = new InjectionToken<string>('BILLING_API_URL', {
  providedIn: 'root',
  factory: () =>
    typeof __BILLING_API_URL__ === 'undefined'
      ? 'http://localhost:8082'
      : __BILLING_API_URL__,
});
