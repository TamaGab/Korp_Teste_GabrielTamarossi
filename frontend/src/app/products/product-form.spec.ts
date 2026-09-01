import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
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
        provideRouter([
          { path: 'products', component: EmptyPage },
          { path: 'products/:id/edit', component: ProductForm },
        ]),
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
    expect(document.body.textContent).toContain('Produto cadastrado com sucesso');
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

  it('loads and edits a Product without regenerating its Product Code', async () => {
    const harness = await RouterTestingHarness.create('/products/1/edit');
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products/1').flush({
      id: 1,
      code: 'LAP01',
      description: 'Laptop',
      stock: 7,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    });
    harness.detectChanges();

    const page = harness.routeNativeElement as HTMLElement;
    expect((page.querySelector('[formControlName="code"]') as HTMLInputElement).value).toBe(
      'LAP01',
    );
    setInput(page, 'description', 'Notebook');
    expect((page.querySelector('[formControlName="code"]') as HTMLInputElement).value).toBe(
      'LAP01',
    );
    setInput(page, 'code', 'not02');
    page.querySelector('form')?.dispatchEvent(new Event('submit'));

    const request = http.expectOne('http://localhost:8081/products/1');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({
      code: 'NOT02',
      description: 'Notebook',
      stock: 7,
    });
    request.flush({
      id: 1,
      code: 'NOT02',
      description: 'Notebook',
      stock: 7,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-09-01T15:00:00Z',
    });
    await harness.fixture.whenStable();

    expect(TestBed.inject(Router).url).toBe('/products');
    expect(document.body.textContent).toContain('Produto atualizado com sucesso');
    http.verify();
  });

  it('shows loading and not-found states when the Product does not exist', async () => {
    const harness = await RouterTestingHarness.create('/products/999/edit');
    const page = harness.routeNativeElement as HTMLElement;
    expect(page.textContent).toContain('Carregando produto...');

    const http = TestBed.inject(HttpTestingController);
    http
      .expectOne('http://localhost:8081/products/999')
      .flush({ error: 'product not found' }, { status: 404, statusText: 'Not Found' });
    harness.detectChanges();

    expect(page.textContent).toContain('Produto não encontrado.');
    expect(page.querySelector('form')).toBeNull();
    http.verify();
  });

  it('does not turn an invalid edit URL into Product creation', async () => {
    const harness = await RouterTestingHarness.create('/products/invalid/edit');
    const page = harness.routeNativeElement as HTMLElement;

    expect(page.textContent).toContain('Produto não encontrado.');
    expect(page.querySelector('form')).toBeNull();
    TestBed.inject(HttpTestingController).verify();
  });

  it('preserves edited values and a general failure until the form changes', async () => {
    const harness = await RouterTestingHarness.create('/products/1/edit');
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products/1').flush({
      id: 1,
      code: 'LAP01',
      description: 'Laptop',
      stock: 7,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    });
    harness.detectChanges();

    const page = harness.routeNativeElement as HTMLElement;
    setInput(page, 'description', 'Notebook');
    page.querySelector('form')?.dispatchEvent(new Event('submit'));
    http
      .expectOne('http://localhost:8081/products/1')
      .flush({ error: 'could not update product' }, { status: 500, statusText: 'Error' });
    harness.detectChanges();

    expect(page.textContent).toContain('Não foi possível atualizar o produto. Tente novamente.');
    expect((page.querySelector('[formControlName="description"]') as HTMLInputElement).value).toBe(
      'Notebook',
    );
    harness.detectChanges();
    expect(page.textContent).toContain('Não foi possível atualizar o produto. Tente novamente.');

    setInput(page, 'stock', '8');
    harness.detectChanges();
    expect(page.textContent).not.toContain(
      'Não foi possível atualizar o produto. Tente novamente.',
    );
    http.verify();
  });

  it('keeps an edit conflict beside Product Code until that field changes', async () => {
    const harness = await RouterTestingHarness.create('/products/1/edit');
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products/1').flush({
      id: 1,
      code: 'LAP01',
      description: 'Laptop',
      stock: 7,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    });
    harness.detectChanges();

    const page = harness.routeNativeElement as HTMLElement;
    setInput(page, 'code', 'MON01');
    page.querySelector('form')?.dispatchEvent(new Event('submit'));
    http
      .expectOne('http://localhost:8081/products/1')
      .flush({ error: 'product code already exists' }, { status: 409, statusText: 'Conflict' });
    harness.detectChanges();

    expect(page.textContent).toContain('Este código já está cadastrado.');
    expect((page.querySelector('[formControlName="code"]') as HTMLInputElement).value).toBe(
      'MON01',
    );
    setInput(page, 'description', 'Monitor');
    harness.detectChanges();
    expect(page.textContent).toContain('Este código já está cadastrado.');

    setInput(page, 'code', 'MON02');
    harness.detectChanges();
    expect(page.textContent).not.toContain('Este código já está cadastrado.');
    http.verify();
  });
});

function setInput(page: HTMLElement, controlName: string, value: string) {
  const input = page.querySelector(`[formControlName="${controlName}"]`) as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input'));
}
