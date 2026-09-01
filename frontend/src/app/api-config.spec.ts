import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { InvoiceApi } from './invoices/invoice-api';
import { ProductApi } from './products/product-api';
import { BILLING_API_URL, INVENTORY_API_URL } from './api-config';

describe('API configuration', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: INVENTORY_API_URL, useValue: 'http://localhost:9081' },
        { provide: BILLING_API_URL, useValue: 'http://localhost:9082' },
      ],
    });
  });

  it('uses the configured Inventory API URL', () => {
    TestBed.inject(ProductApi).list().subscribe();

    TestBed.inject(HttpTestingController).expectOne('http://localhost:9081/products');
  });

  it('uses the configured Billing API URL', () => {
    TestBed.inject(InvoiceApi).list().subscribe();

    TestBed.inject(HttpTestingController).expectOne('http://localhost:9082/invoices');
  });
});
