import { DatePipe } from '@angular/common';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSortModule, Sort, SortDirection } from '@angular/material/sort';
import { MatTableModule } from '@angular/material/table';
import { RouterLink } from '@angular/router';
import { Product } from './product';
import { ProductApi } from './product-api';

type StockFilter = 'all' | 'in-stock' | 'out-of-stock';
type SortColumn = 'code' | 'description' | 'stock' | 'createdAt' | 'updatedAt';

@Component({
  selector: 'app-product-list',
  imports: [
    DatePipe,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatProgressSpinnerModule,
    MatSortModule,
    MatTableModule,
    ReactiveFormsModule,
    RouterLink,
  ],
  templateUrl: './product-list.html',
  styleUrl: './product-list.css',
})
export class ProductList implements OnInit {
  private readonly productApi = inject(ProductApi);

  readonly displayedColumns = ['code', 'description', 'stock', 'createdAt', 'updatedAt'];
  readonly products = signal<Product[]>([]);
  readonly loading = signal(true);
  readonly loadFailed = signal(false);
  readonly searchControl = new FormControl('', { nonNullable: true });
  readonly stockFilterControl = new FormControl<StockFilter>('all', { nonNullable: true });
  readonly pageSizeControl = new FormControl(10, { nonNullable: true });
  readonly search = signal('');
  readonly stockFilter = signal<StockFilter>('all');
  readonly sortColumn = signal<SortColumn>('code');
  readonly sortDirection = signal<SortDirection>('asc');
  readonly pageIndex = signal(0);
  readonly pageSize = signal(10);
  readonly filteredProducts = computed(() => {
    const search = this.search().trim().toLocaleLowerCase();
    const stockFilter = this.stockFilter();

    return this.products().filter((product) => {
      const matchesSearch =
        !search ||
        product.code.toLocaleLowerCase().includes(search) ||
        product.description.toLocaleLowerCase().includes(search);
      const matchesStock =
        stockFilter === 'all' ||
        (stockFilter === 'in-stock' && product.stock > 0) ||
        (stockFilter === 'out-of-stock' && product.stock === 0);

      return matchesSearch && matchesStock;
    });
  });
  readonly sortedProducts = computed(() => {
    const column = this.sortColumn();
    const direction = this.sortDirection() === 'asc' ? 1 : -1;

    return [...this.filteredProducts()].sort((first, second) => {
      const firstValue = first[column];
      const secondValue = second[column];
      const comparison =
        typeof firstValue === 'number'
          ? firstValue - (secondValue as number)
          : firstValue.localeCompare(secondValue as string);

      return comparison * direction;
    });
  });
  readonly totalPages = computed(() =>
    Math.max(1, Math.ceil(this.sortedProducts().length / this.pageSize())),
  );
  readonly pagedProducts = computed(() => {
    const first = this.pageIndex() * this.pageSize();
    return this.sortedProducts().slice(first, first + this.pageSize());
  });

  ngOnInit() {
    this.searchControl.valueChanges.subscribe((search) => {
      this.search.set(search);
      this.pageIndex.set(0);
    });
    this.stockFilterControl.valueChanges.subscribe((filter) => {
      this.stockFilter.set(filter);
      this.pageIndex.set(0);
    });
    this.pageSizeControl.valueChanges.subscribe((size) => {
      this.pageSize.set(Number(size));
      this.pageIndex.set(0);
    });

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

  setSort(sort: Sort) {
    this.sortColumn.set(sort.active as SortColumn);
    this.sortDirection.set(sort.direction || 'asc');
  }

  previousPage() {
    this.pageIndex.update((page) => Math.max(0, page - 1));
  }

  nextPage() {
    this.pageIndex.update((page) => Math.min(this.totalPages() - 1, page + 1));
  }
}
