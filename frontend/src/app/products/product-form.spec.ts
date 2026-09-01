import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { ProductForm } from './product-form';

@Component({ template: '' })
class EmptyPage {}

describe('ProductForm', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProductForm],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([{ path: 'products', component: EmptyPage }]),
      ],
    }).compileComponents();
  });

  it('creates a Product and returns to the list with visual confirmation', async () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();
    TestBed.inject(HttpTestingController).expectOne('http://localhost:8081/products').flush([]);

    setInput(fixture.nativeElement, 'code', 'lap01');
    setInput(fixture.nativeElement, 'description', 'Laptop');
    setInput(fixture.nativeElement, 'stock', '7');
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));

    const http = TestBed.inject(HttpTestingController);
    const request = http.expectOne('http://localhost:8081/products');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({
      code: 'LAP01',
      description: 'Laptop',
      stock: 7,
    });
    request.flush({
      id: 1,
      code: 'LAP01',
      description: 'Laptop',
      stock: 7,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    });
    await fixture.whenStable();

    expect(TestBed.inject(Router).url).toBe('/products');
    expect(document.body.textContent).toContain('Product created successfully');
    http.verify();
  });

  it('suggests the lowest available Product Code from the normalized Description', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();

    TestBed.inject(HttpTestingController)
      .expectOne('http://localhost:8081/products')
      .flush([
        {
          id: 1,
          code: 'MAC01',
          description: 'Outra maçã',
          stock: 1,
          createdAt: '2026-08-31T15:00:00Z',
          updatedAt: '2026-08-31T15:00:00Z',
        },
      ]);
    setInput(fixture.nativeElement, 'description', '  Maçã! verde');

    expect(
      (fixture.nativeElement.querySelector('[formControlName="code"]') as HTMLInputElement).value,
    ).toBe('MAC02');
  });

  it('does not replace a Product Code after the user changes it manually', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();
    TestBed.inject(HttpTestingController).expectOne('http://localhost:8081/products').flush([]);

    setInput(fixture.nativeElement, 'description', 'Laptop');
    setInput(fixture.nativeElement, 'code', 'not01');
    setInput(fixture.nativeElement, 'description', 'Mouse');

    expect(
      (fixture.nativeElement.querySelector('[formControlName="code"]') as HTMLInputElement).value,
    ).toBe('NOT01');
  });

  it('asks for a manual prefix when all suffixes are occupied', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();
    const products = Array.from({ length: 99 }, (_, index) => ({
      id: index + 1,
      code: `MAC${(index + 1).toString().padStart(2, '0')}`,
      description: `Product ${index + 1}`,
      stock: 1,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    }));
    TestBed.inject(HttpTestingController)
      .expectOne('http://localhost:8081/products')
      .flush(products);

    setInput(fixture.nativeElement, 'description', 'Laptop');
    expect(
      (fixture.nativeElement.querySelector('[formControlName="code"]') as HTMLInputElement).value,
    ).toBe('LAP01');

    setInput(fixture.nativeElement, 'description', 'Maçã');
    fixture.detectChanges();

    expect(
      (fixture.nativeElement.querySelector('[formControlName="code"]') as HTMLInputElement).value,
    ).toBe('');
    expect(fixture.nativeElement.textContent).toContain(
      'Todos os códigos deste prefixo estão em uso. Altere o prefixo manualmente.',
    );

    setInput(fixture.nativeElement, 'code', 'alt01');
    fixture.detectChanges();
    expect(
      (fixture.nativeElement.querySelector('[formControlName="code"]') as HTMLInputElement).value,
    ).toBe('ALT01');
    expect(fixture.nativeElement.textContent).not.toContain(
      'Todos os códigos deste prefixo estão em uso. Altere o prefixo manualmente.',
    );
  });

  it('keeps a Product Code validation error until the Product Code changes', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();
    TestBed.inject(HttpTestingController).expectOne('http://localhost:8081/products').flush([]);

    setInput(fixture.nativeElement, 'code', 'AB1');
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Use o formato AAA00');

    setInput(fixture.nativeElement, 'stock', '4');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Use o formato AAA00');
  });

  it('keeps entered values and shows a duplicate code error after failure', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();
    TestBed.inject(HttpTestingController).expectOne('http://localhost:8081/products').flush([]);

    setInput(fixture.nativeElement, 'code', 'LAP01');
    setInput(fixture.nativeElement, 'description', 'Laptop');
    setInput(fixture.nativeElement, 'stock', '7');
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));

    TestBed.inject(HttpTestingController)
      .expectOne('http://localhost:8081/products')
      .flush({ error: 'product code already exists' }, { status: 409, statusText: 'Conflict' });
    fixture.detectChanges();

    const page = fixture.nativeElement as HTMLElement;
    expect((page.querySelector('[formControlName="code"]') as HTMLInputElement).value).toBe(
      'LAP01',
    );
    expect((page.querySelector('[formControlName="description"]') as HTMLInputElement).value).toBe(
      'Laptop',
    );
    expect((page.querySelector('[formControlName="stock"]') as HTMLInputElement).value).toBe('7');
    expect(page.textContent).toContain('Este código já está cadastrado.');

    setInput(page, 'description', 'Notebook');
    fixture.detectChanges();
    expect(page.textContent).toContain('Este código já está cadastrado.');

    setInput(page, 'code', 'LAP02');
    fixture.detectChanges();
    expect(page.textContent).not.toContain('Este código já está cadastrado.');
  });
});

function setInput(page: HTMLElement, controlName: string, value: string) {
  const input = page.querySelector(`[formControlName="${controlName}"]`) as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input'));
}
