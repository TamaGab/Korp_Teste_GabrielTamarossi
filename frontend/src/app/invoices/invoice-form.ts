import { HttpErrorResponse } from '@angular/common/http';
import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  AbstractControl,
  FormArray,
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  Validators,
} from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { RouterLink } from '@angular/router';
import { ProductApi } from '../products/product-api';
import { Product } from '../products/product';
import { InvoiceApi } from './invoice-api';
import { Invoice } from './invoice';

@Component({
  selector: 'app-invoice-form',
  imports: [
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    ReactiveFormsModule,
    RouterLink,
  ],
  templateUrl: './invoice-form.html',
  styleUrl: './invoice-form.css',
})
export class InvoiceForm {
  private readonly productApi = inject(ProductApi);
  private readonly invoiceApi = inject(InvoiceApi);
  private readonly destroyRef = inject(DestroyRef);

  readonly products = signal<Product[]>([]);
  readonly loadingProducts = signal(true);
  readonly loadFailed = signal(false);
  readonly submitting = signal(false);
  readonly generalError = signal('');
  readonly createdInvoice = signal<Invoice | null>(null);
  readonly lines = new FormArray([this.createLine()], distinctProducts);
  readonly form = new FormGroup({ lines: this.lines });

  constructor() {
    this.productApi.list().subscribe({
      next: (products) => {
        this.products.set(products);
        this.loadingProducts.set(false);
      },
      error: () => {
        this.loadingProducts.set(false);
        this.loadFailed.set(true);
        this.generalError.set('Não foi possível carregar os produtos. Tente novamente.');
      },
    });

    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => {
      this.generalError.set('');
    });
  }

  addLine() {
    this.lines.push(this.createLine());
  }

  removeLine(index: number) {
    if (this.lines.length > 1) {
      this.lines.removeAt(index);
    }
  }

  submit() {
    if (this.form.invalid || this.submitting()) {
      this.form.markAllAsTouched();
      return;
    }

    const invoice = {
      lines: this.lines.getRawValue().map((line) => ({
        productCode: line.productCode,
        quantity: Number(line.quantity),
      })),
    };
    this.submitting.set(true);
    this.generalError.set('');

    this.invoiceApi.create(invoice).subscribe({
      next: (createdInvoice) => {
        this.createdInvoice.set(createdInvoice);
        this.submitting.set(false);
      },
      error: (error: HttpErrorResponse) => {
        this.submitting.set(false);
        this.generalError.set(
          error.status === 422
            ? 'Um dos produtos selecionados não está mais disponível.'
            : 'Não foi possível cadastrar a nota fiscal. Tente novamente.',
        );
      },
    });
  }

  createAnother() {
    this.createdInvoice.set(null);
    this.lines.clear();
    this.lines.push(this.createLine());
    this.form.markAsPristine();
    this.form.markAsUntouched();
  }

  private createLine() {
    return new FormGroup({
      productCode: new FormControl('', {
        nonNullable: true,
        validators: [Validators.required],
      }),
      quantity: new FormControl(1, {
        nonNullable: true,
        validators: [Validators.required, Validators.min(1), Validators.pattern(/^\d+$/)],
      }),
    });
  }
}

function distinctProducts(control: AbstractControl): ValidationErrors | null {
  const lines = control as FormArray;
  const productCodes = lines.controls
    .map((line) => line.get('productCode')?.value)
    .filter((productCode) => productCode !== '');
  return new Set(productCodes).size === productCodes.length ? null : { duplicateProducts: true };
}
