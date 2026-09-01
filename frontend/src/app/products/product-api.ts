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

  create(product: CreateProduct) {
    return this.http.post<Product>(this.productsUrl, product);
  }
}
