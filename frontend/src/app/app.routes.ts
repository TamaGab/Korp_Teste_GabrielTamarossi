import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./home/home').then((module) => module.Home),
  },
  {
    path: 'products',
    loadComponent: () => import('./products/product-list').then((module) => module.ProductList),
  },
  {
    path: 'products/new',
    loadComponent: () => import('./products/product-form').then((module) => module.ProductForm),
  },
  {
    path: 'products/:id/edit',
    loadComponent: () => import('./products/product-form').then((module) => module.ProductForm),
  },
  { path: '**', redirectTo: '' },
];
