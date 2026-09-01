import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { InvoiceForm } from './invoice-form';

describe('InvoiceForm', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InvoiceForm],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
  });

  it('creates an Open Invoice with multiple Product quantities', () => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush([
      {
        id: 1,
        code: 'LAP01',
        description: 'Laptop',
        stock: 7,
        createdAt: '2026-08-31T15:00:00Z',
        updatedAt: '2026-08-31T15:00:00Z',
      },
      {
        id: 2,
        code: 'MON01',
        description: 'Monitor',
        stock: 0,
        createdAt: '2026-08-31T15:00:00Z',
        updatedAt: '2026-08-31T15:00:00Z',
      },
    ]);
    fixture.detectChanges();

    setSelect(fixture.nativeElement, 0, 'LAP01');
    setQuantity(fixture.nativeElement, 0, '2');
    (fixture.nativeElement.querySelector('[data-testid="add-line"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    setSelect(fixture.nativeElement, 1, 'MON01');
    setQuantity(fixture.nativeElement, 1, '3');
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));

    const request = http.expectOne('http://localhost:8082/invoices');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({
      lines: [
        { productCode: 'LAP01', quantity: 2 },
        { productCode: 'MON01', quantity: 3 },
      ],
    });
    request.flush({
      number: '0001',
      status: 'OPEN',
      lines: [
        { code: 'LAP01', description: 'Laptop', quantity: 2 },
        { code: 'MON01', description: 'Monitor', quantity: 3 },
      ],
      createdAt: '2026-09-01T15:00:00Z',
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Nota fiscal 0001 cadastrada com sucesso.');
    expect(fixture.nativeElement.textContent).toContain('Status: Aberta');
    http.verify();
  });

  it('rejects duplicate Products and non-positive quantities before submission', () => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush(products);
    fixture.detectChanges();

    setSelect(fixture.nativeElement, 0, 'LAP01');
    setQuantity(fixture.nativeElement, 0, '0');
    (fixture.nativeElement.querySelector('[data-testid="add-line"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    setSelect(fixture.nativeElement, 1, 'LAP01');
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      'Cada produto pode aparecer apenas uma vez.',
    );
    expect(fixture.nativeElement.textContent).toContain('Informe um número inteiro maior que zero');
    http.expectNone('http://localhost:8082/invoices');
    http.verify();
  });

  it.each(['-1', '1.5'])('rejects the invalid quantity %s before submission', (quantity) => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush(products);
    fixture.detectChanges();

    setSelect(fixture.nativeElement, 0, 'LAP01');
    setQuantity(fixture.nativeElement, 0, quantity);
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Informe um número inteiro maior que zero');
    http.expectNone('http://localhost:8082/invoices');
    http.verify();
  });

  it('allows a Product line to be removed before submission', () => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush(products);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('[data-testid="add-line"]') as HTMLButtonElement).click();
    fixture.detectChanges();
    const removeButtons = (
      fixture.nativeElement as HTMLElement
    ).querySelectorAll<HTMLButtonElement>('[aria-label="Remover produto"]');
    expect(removeButtons.length).toBe(2);
    removeButtons[1].click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('[formControlName="productCode"]').length).toBe(
      1,
    );
    http.verify();
  });

  it('ignores repeated submissions and reports a Product removed from inventory', () => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush(products);
    fixture.detectChanges();
    setSelect(fixture.nativeElement, 0, 'LAP01');

    const form = fixture.nativeElement.querySelector('form');
    form.dispatchEvent(new Event('submit'));
    form.dispatchEvent(new Event('submit'));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Cadastrando...');
    const request = http.expectOne('http://localhost:8082/invoices');
    request.flush(
      { error: 'product not found', productCode: 'LAP01' },
      { status: 422, statusText: 'Unprocessable Entity' },
    );
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      'Um dos produtos selecionados não está mais disponível.',
    );
    expect(
      (fixture.nativeElement.querySelector('[formControlName="productCode"]') as HTMLSelectElement)
        .value,
    ).toBe('LAP01');
    expect(fixture.nativeElement.textContent).toContain('Cadastrar nota fiscal');
    http.verify();
  });

  it('shows an error when Products cannot be loaded', () => {
    const fixture = TestBed.createComponent(InvoiceForm);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http
      .expectOne('http://localhost:8081/products')
      .flush({ error: 'unavailable' }, { status: 503, statusText: 'Unavailable' });
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      'Não foi possível carregar os produtos. Tente novamente.',
    );
    expect(fixture.nativeElement.querySelector('form')).toBeNull();
    http.verify();
  });
});

const products = [
  {
    id: 1,
    code: 'LAP01',
    description: 'Laptop',
    stock: 7,
    createdAt: '2026-08-31T15:00:00Z',
    updatedAt: '2026-08-31T15:00:00Z',
  },
  {
    id: 2,
    code: 'MON01',
    description: 'Monitor',
    stock: 0,
    createdAt: '2026-08-31T15:00:00Z',
    updatedAt: '2026-08-31T15:00:00Z',
  },
];

function setSelect(page: HTMLElement, index: number, value: string) {
  const select = page.querySelectorAll<HTMLSelectElement>('[formControlName="productCode"]')[index];
  select.value = value;
  select.dispatchEvent(new Event('change'));
}

function setQuantity(page: HTMLElement, index: number, value: string) {
  const input = page.querySelectorAll<HTMLInputElement>('[formControlName="quantity"]')[index];
  input.value = value;
  input.dispatchEvent(new Event('input'));
}
