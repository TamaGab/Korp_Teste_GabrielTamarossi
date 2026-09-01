import { HttpClient } from "@angular/common/http";
import { Injectable, inject } from "@angular/core";
import { BILLING_API_URL } from "../api-config";
import {
  CreateInvoice,
  Invoice,
  InvoiceSummary,
  PreparedInvoicePrint,
} from "./invoice";

@Injectable({ providedIn: "root" })
export class InvoiceApi {
  private readonly http = inject(HttpClient);
  private readonly invoicesUrl = `${inject(BILLING_API_URL)}/invoices`;

  create(invoice: CreateInvoice) {
    return this.http.post<Invoice>(this.invoicesUrl, invoice);
  }

  list() {
    return this.http.get<InvoiceSummary[]>(this.invoicesUrl);
  }

  get(number: string) {
    return this.http.get<Invoice>(`${this.invoicesUrl}/${number}`);
  }

  preparePrint(number: string) {
    return this.http.post<PreparedInvoicePrint>(
      `${this.invoicesUrl}/${number}/prepare-print`,
      {},
    );
  }

  close(number: string) {
    return this.http.post<Pick<Invoice, "number" | "status">>(
      `${this.invoicesUrl}/${number}/close`,
      {},
    );
  }
}
