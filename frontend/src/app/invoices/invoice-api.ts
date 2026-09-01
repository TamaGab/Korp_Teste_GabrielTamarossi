import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { CreateInvoice, Invoice } from './invoice';

@Injectable({ providedIn: 'root' })
export class InvoiceApi {
  private readonly http = inject(HttpClient);
  private readonly invoicesUrl = 'http://localhost:8082/invoices';

  create(invoice: CreateInvoice) {
    return this.http.post<Invoice>(this.invoicesUrl, invoice);
  }
}
