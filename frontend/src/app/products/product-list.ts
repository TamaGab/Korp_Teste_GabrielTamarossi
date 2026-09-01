import { DatePipe } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { RouterLink } from '@angular/router';
import { Product } from './product';
import { ProductApi } from './product-api';

@Component({
  selector: 'app-product-list',
  imports: [DatePipe, MatButtonModule, MatProgressSpinnerModule, MatTableModule, RouterLink],
  templateUrl: './product-list.html',
  styleUrl: './product-list.css',
})
export class ProductList implements OnInit {
  private readonly productApi = inject(ProductApi);

  readonly displayedColumns = ['code', 'description', 'stock', 'createdAt', 'updatedAt'];
  readonly products = signal<Product[]>([]);
  readonly loading = signal(true);
  readonly loadFailed = signal(false);

  ngOnInit() {
    this.productApi.list().subscribe({
      next: (products) => {
        this.products.set(products);
        this.loading.set(false);
      },
      error: () => {
        this.loadFailed.set(true);
        this.loading.set(false);
      },
    });
  }
}
