import { HttpErrorResponse } from '@angular/common/http';
import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router, RouterLink } from '@angular/router';
import { ProductApi } from './product-api';

@Component({
  selector: 'app-product-form',
  imports: [
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    ReactiveFormsModule,
    RouterLink,
  ],
  templateUrl: './product-form.html',
  styleUrl: './product-form.css',
})
export class ProductForm {
  private readonly productApi = inject(ProductApi);
  private readonly router = inject(Router);
  private readonly snackBar = inject(MatSnackBar);
  private readonly destroyRef = inject(DestroyRef);

  readonly submitting = signal(false);
  readonly generalError = signal('');
  readonly form = new FormGroup({
    code: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(/^[A-Z]{3}[0-9]{2}$/)],
    }),
    description: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required, Validators.maxLength(200)],
    }),
    stock: new FormControl(0, {
      nonNullable: true,
      validators: [Validators.required, Validators.min(0), Validators.pattern(/^\d+$/)],
    }),
  });

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => {
      this.generalError.set('');
    });
    this.form.controls.code.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.clearDuplicateError());
  }

  uppercaseCode() {
    const code = this.form.controls.code.value.toUpperCase();
    this.form.controls.code.setValue(code, { emitEvent: false });
    this.clearDuplicateError();
    this.generalError.set('');
  }

  submit() {
    if (this.form.invalid || this.submitting()) {
      this.form.markAllAsTouched();
      return;
    }

    const value = this.form.getRawValue();
    const product = {
      code: value.code.toUpperCase(),
      description: value.description,
      stock: Number(value.stock),
    };
    this.form.controls.code.setValue(product.code, { emitEvent: false });
    this.submitting.set(true);
    this.generalError.set('');

    this.productApi.create(product).subscribe({
      next: () => {
        this.snackBar.open('Product created successfully', 'Close', {
          duration: 4000,
        });
        void this.router.navigate(['/products']);
      },
      error: (error: HttpErrorResponse) => {
        this.submitting.set(false);
        if (error.status === 409) {
          this.form.controls.code.setErrors({ duplicate: true });
          this.form.controls.code.markAsTouched();
          return;
        }
        this.generalError.set(error.error?.error || 'Product could not be created. Try again.');
      },
    });
  }

  private clearDuplicateError() {
    const control = this.form.controls.code;
    if (!control.hasError('duplicate')) {
      return;
    }
    const { duplicate: _, ...otherErrors } = control.errors ?? {};
    control.setErrors(Object.keys(otherErrors).length ? otherErrors : null);
  }
}
