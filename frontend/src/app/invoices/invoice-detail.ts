import { DatePipe } from "@angular/common";
import { HttpErrorResponse } from "@angular/common/http";
import { Component, OnInit, inject, signal } from "@angular/core";
import { MatButtonModule } from "@angular/material/button";
import { MatProgressSpinnerModule } from "@angular/material/progress-spinner";
import { MatTableModule } from "@angular/material/table";
import { ActivatedRoute, RouterLink } from "@angular/router";
import { InvoiceApi } from "./invoice-api";
import { Invoice, invoiceStatusLabel } from "./invoice";

@Component({
  selector: "app-invoice-detail",
  imports: [
    DatePipe,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatTableModule,
    RouterLink,
  ],
  templateUrl: "./invoice-detail.html",
  styleUrl: "./invoice-detail.css",
})
export class InvoiceDetail implements OnInit {
  private readonly invoiceApi = inject(InvoiceApi);
  private readonly route = inject(ActivatedRoute);

  readonly invoice = signal<Invoice | null>(null);
  readonly loading = signal(true);
  readonly errorMessage = signal("");
  readonly statusLabel = invoiceStatusLabel;
  readonly displayedColumns = ["code", "description", "quantity"];

  ngOnInit() {
    const number = this.route.snapshot.paramMap.get("number") ?? "";
    this.invoiceApi.get(number).subscribe({
      next: (invoice) => {
        this.invoice.set(invoice);
        this.loading.set(false);
      },
      error: (error: HttpErrorResponse) => {
        this.errorMessage.set(
          error.status === 404
            ? "Nota fiscal não encontrada."
            : "Não foi possível carregar a nota fiscal. Tente novamente.",
        );
        this.loading.set(false);
      },
    });
  }
}
