import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { CreateProduct, Product } from './product';

@Injectable({ providedIn: 'root' })
export class ProductApi {
  private readonly http = inject(HttpClient);
  private readonly productsUrl = 'http://localhost:8081/products';

  list() {
    return this.http.get<Product[]>(this.productsUrl);
  }

  get(id: number) {
    return this.http.get<Product>(`${this.productsUrl}/${id}`);
  }

  create(product: CreateProduct) {
    return this.http.post<Product>(this.productsUrl, product);
  }

  update(id: number, product: CreateProduct) {
    return this.http.put<Product>(`${this.productsUrl}/${id}`, product);
  }

  delete(id: number) {
    return this.http.delete<void>(`${this.productsUrl}/${id}`);
  }
}
