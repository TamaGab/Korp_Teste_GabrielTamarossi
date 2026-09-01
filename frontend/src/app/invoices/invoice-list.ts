import { DatePipe } from "@angular/common";
import { Component, OnInit, inject, signal } from "@angular/core";
import { MatButtonModule } from "@angular/material/button";
import { MatProgressSpinnerModule } from "@angular/material/progress-spinner";
import { MatTableModule } from "@angular/material/table";
import { RouterLink } from "@angular/router";
import { InvoiceApi } from "./invoice-api";
import { InvoiceSummary, invoiceStatusLabel } from "./invoice";

@Component({
  selector: "app-invoice-list",
  imports: [
    DatePipe,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatTableModule,
    RouterLink,
  ],
  templateUrl: "./invoice-list.html",
  styleUrl: "./invoice-list.css",
})
export class InvoiceList implements OnInit {
  private readonly invoiceApi = inject(InvoiceApi);

  readonly displayedColumns = ["number", "status", "createdAt", "actions"];
  readonly invoices = signal<InvoiceSummary[]>([]);
  readonly loading = signal(true);
  readonly loadFailed = signal(false);
  readonly statusLabel = invoiceStatusLabel;

  ngOnInit() {
    this.invoiceApi.list().subscribe({
      next: (invoices) => {
        this.invoices.set(invoices);
        this.loading.set(false);
      },
      error: () => {
        this.loadFailed.set(true);
        this.loading.set(false);
      },
    });
  }
}
