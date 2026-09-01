import { DatePipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { Invoice, PrintPreparationProblem, invoiceStatusLabel } from './invoice';
import { InvoiceApi } from './invoice-api';

@Component({
  selector: 'app-invoice-detail',
  imports: [
    DatePipe,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatTableModule,
    RouterLink,
    MatIconModule,
  ],
  templateUrl: './invoice-detail.html',
  styleUrl: './invoice-detail.css',
})
export class InvoiceDetail implements OnInit {
  private readonly invoiceApi = inject(InvoiceApi);
  private readonly route = inject(ActivatedRoute);

  readonly invoice = signal<Invoice | null>(null);
  readonly loading = signal(true);
  readonly errorMessage = signal('');
  readonly actionErrorMessages = signal<string[]>([]);
  readonly preparingPrint = signal(false);
  readonly closingInvoice = signal(false);
  readonly statusLabel = invoiceStatusLabel;
  readonly displayedColumns = ['code', 'description', 'quantity'];

  ngOnInit() {
    const number = this.route.snapshot.paramMap.get('number') ?? '';
    this.invoiceApi.get(number).subscribe({
      next: (invoice) => {
        this.invoice.set(invoice);
        this.loading.set(false);
      },
      error: (error: HttpErrorResponse) => {
        this.errorMessage.set(
          error.status === 404
            ? 'Nota fiscal não encontrada.'
            : 'Não foi possível carregar a nota fiscal. Tente novamente.',
        );
        this.loading.set(false);
      },
    });
  }

  preparePrint() {
    const current = this.invoice();
    if (
      !current ||
      current.status !== 'OPEN' ||
      current.closingPending ||
      this.preparingPrint() ||
      this.closingInvoice()
    ) {
      return;
    }

    this.preparingPrint.set(true);
    this.actionErrorMessages.set([]);
    this.invoiceApi
      .preparePrint(current.number)
      .pipe(finalize(() => this.preparingPrint.set(false)))
      .subscribe({
        next: (prepared) => {
          const printFrame = document.createElement('iframe');
          printFrame.setAttribute('aria-hidden', 'true');
          printFrame.style.height = '0';
          printFrame.style.position = 'fixed';
          printFrame.style.width = '0';
          document.body.appendChild(printFrame);

          const printWindow = printFrame.contentWindow;
          if (!printWindow) {
            printFrame.remove();
            this.actionErrorMessages.set(['Não foi possível abrir a impressão. Tente novamente.']);
            return;
          }
          let closingStarted = false;
          const finishPrinting = () => {
            if (closingStarted) {
              return;
            }
            closingStarted = true;
            printWindow.removeEventListener('afterprint', finishPrinting);
            window.removeEventListener('afterprint', finishPrinting);
            printFrame.remove();
            this.closeInvoice(current.number);
          };
          printWindow.addEventListener('afterprint', finishPrinting, { once: true });
          window.addEventListener('afterprint', finishPrinting, { once: true });
          printWindow.document.open();
          printWindow.document.write(prepared.html);
          printWindow.document.close();
          printWindow.focus();
          printWindow.print();
          finishPrinting();
        },
        error: (error: HttpErrorResponse) => {
          const problems = error.error?.problems;
          if (error.status === 422 && Array.isArray(problems)) {
            this.actionErrorMessages.set(
              problems.map((problem: PrintPreparationProblem) => this.stockProblemMessage(problem)),
            );
            return;
          }
          this.actionErrorMessages.set(['Não foi possível verificar o estoque. Tente novamente.']);
        },
      });
  }

  retryClosing() {
    const current = this.invoice();
    if (
      !current ||
      current.status !== 'OPEN' ||
      !current.closingPending ||
      this.closingInvoice()
    ) {
      return;
    }
    this.closeInvoice(current.number);
  }

  private closeInvoice(number: string) {
    this.closingInvoice.set(true);
    this.actionErrorMessages.set([]);
    this.invoice.update((current) =>
      current ? { ...current, closingPending: true } : current,
    );
    this.invoiceApi
      .close(number)
      .pipe(finalize(() => this.closingInvoice.set(false)))
      .subscribe({
        next: (closed) => {
          this.invoice.update((current) =>
            current ? { ...current, status: closed.status, closingPending: false } : current,
          );
        },
        error: () => {
          this.actionErrorMessages.set([
            'Não foi possível finalizar a nota fiscal. Tente novamente.',
          ]);
        },
      });
  }

  private stockProblemMessage(problem: PrintPreparationProblem) {
    if (problem.reason === 'product_not_found') {
      return `O produto ${problem.productCode} não está mais disponível no estoque.`;
    }
    return `Estoque insuficiente para ${problem.productCode}: disponível ${problem.availableStock}, necessário ${problem.requestedQuantity}.`;
  }
}
