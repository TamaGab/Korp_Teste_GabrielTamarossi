import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { ProductList } from './product-list';

describe('ProductList', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProductList],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
  });

  it('shows Products returned by the inventory API', async () => {
    const fixture = TestBed.createComponent(ProductList);
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
    ]);
    fixture.detectChanges();
    await fixture.whenStable();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.textContent).toContain('LAP01');
    expect(page.textContent).toContain('Laptop');
    expect(page.textContent).toContain('7');
    expect(page.querySelectorAll('tbody tr')).toHaveLength(1);
    http.verify();
  });

  it('offers editing for each Product', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();

    TestBed.inject(HttpTestingController)
      .expectOne('http://localhost:8081/products')
      .flush([
        {
          id: 7,
          code: 'LAP01',
          description: 'Laptop',
          stock: 7,
          createdAt: '2026-08-31T15:00:00Z',
          updatedAt: '2026-08-31T15:00:00Z',
        },
      ]);
    fixture.detectChanges();

    const editLink = fixture.nativeElement.querySelector(
      'a[href="/products/7/edit"]',
    ) as HTMLAnchorElement;
    expect(editLink).not.toBeNull();
    expect(editLink.textContent).toContain('Editar');
  });

  it('asks for confirmation and identifies the Product before deletion', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();
    flushProducts([{ id: 7, code: 'LAP01', description: 'Laptop', stock: 7 }]);
    fixture.detectChanges();

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();

    const dialog = document.querySelector('[role="dialog"]') as HTMLElement;
    expect(dialog).not.toBeNull();
    expect(dialog.textContent).toContain('LAP01');
    expect(dialog.textContent).toContain('Laptop');
    expect(dialog.textContent).toContain('Cancelar');
    expect(dialog.textContent).toContain('Excluir');
  });

  it('keeps the Product and does not call DELETE when deletion is canceled', async () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();
    const http = flushProducts([{ id: 7, code: 'LAP01', description: 'Laptop', stock: 7 }]);
    fixture.detectChanges();

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();
    dialogButton('Cancelar').click();
    await fixture.whenStable();
    fixture.detectChanges();

    http.expectNone('http://localhost:8081/products/7');
    expect(fixture.nativeElement.textContent).toContain('LAP01');
    http.verify();
  });

  it('calls DELETE and disables deletion actions while deletion is pending', async () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();
    const http = flushProducts([{ id: 7, code: 'LAP01', description: 'Laptop', stock: 7 }]);
    fixture.detectChanges();

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();
    dialogButton('Excluir').click();
    await fixture.whenStable();
    fixture.detectChanges();

    const request = http.expectOne('http://localhost:8081/products/7');
    expect(request.request.method).toBe('DELETE');
    expect(deleteButton(fixture.nativeElement).disabled).toBe(true);
    expect(fixture.nativeElement.querySelector('.page-header button').disabled).toBe(true);
    expect(fixture.nativeElement.querySelector('a[aria-disabled="true"]')).not.toBeNull();
    request.flush(null, { status: 204, statusText: 'No Content' });
    http.verify();
  });

  it('removes the Product and shows success feedback after deletion', async () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();
    const http = flushProducts([{ id: 7, code: 'LAP01', description: 'Laptop', stock: 7 }]);
    fixture.detectChanges();

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();
    dialogButton('Excluir').click();
    await fixture.whenStable();
    http.expectOne('http://localhost:8081/products/7').flush(null, {
      status: 204,
      statusText: 'No Content',
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Nenhum produto cadastrado');
    expect(document.body.textContent).toContain('Produto excluído com sucesso');
    http.verify();
  });

  it('keeps the Product and shows persistent feedback when deletion fails', async () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();
    const http = flushProducts([{ id: 7, code: 'LAP01', description: 'Laptop', stock: 7 }]);
    fixture.detectChanges();

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();
    dialogButton('Excluir').click();
    await fixture.whenStable();
    http
      .expectOne('http://localhost:8081/products/7')
      .flush({ error: 'could not delete product' }, { status: 500, statusText: 'Error' });
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('LAP01');
    expect(fixture.nativeElement.textContent).toContain(
      'Não foi possível excluir o produto. Tente novamente.',
    );
    expect(deleteButton(fixture.nativeElement).disabled).toBe(false);

    deleteButton(fixture.nativeElement).click();
    fixture.detectChanges();
    dialogButton('Cancelar').click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain(
      'Não foi possível excluir o produto. Tente novamente.',
    );
    http.expectNone('http://localhost:8081/products/7');
    http.verify();
  });

  it('searches Products by Product Code or Description', async () => {
    const fixture = TestBed.createComponent(ProductList);
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
        code: 'MON02',
        description: 'Monitor',
        stock: 3,
        createdAt: '2026-08-31T16:00:00Z',
        updatedAt: '2026-08-31T16:00:00Z',
      },
    ]);
    fixture.detectChanges();

    const search = fixture.nativeElement.querySelector('input') as HTMLInputElement;
    search.value = 'lap';
    search.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('tbody tr');
    expect(rows).toHaveLength(1);
    expect(rows[0].textContent).toContain('LAP01');

    search.value = 'monitor';
    search.dispatchEvent(new Event('input'));
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('tbody tr').textContent).toContain('MON02');
    http.verify();
  });

  it('combines textual search with Available Stock filter', () => {
    const fixture = TestBed.createComponent(ProductList);
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
        code: 'MON02',
        description: 'Monitor',
        stock: 0,
        createdAt: '2026-08-31T16:00:00Z',
        updatedAt: '2026-08-31T16:00:00Z',
      },
      {
        id: 3,
        code: 'MOU03',
        description: 'Mouse',
        stock: 4,
        createdAt: '2026-08-31T17:00:00Z',
        updatedAt: '2026-08-31T17:00:00Z',
      },
    ]);
    fixture.detectChanges();

    const search = fixture.nativeElement.querySelector('input') as HTMLInputElement;
    search.value = 'mo';
    search.dispatchEvent(new Event('input'));

    const stockFilter = fixture.nativeElement.querySelector('select') as HTMLSelectElement;
    stockFilter.value = 'out-of-stock';
    stockFilter.dispatchEvent(new Event('change'));
    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('tbody tr');
    expect(rows).toHaveLength(1);
    expect(rows[0].textContent).toContain('MON02');
    http.verify();
  });

  it('starts sorted by Product Code and sorts all table columns', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush([
      {
        id: 1,
        code: 'MON02',
        description: 'Alpha',
        stock: 9,
        createdAt: '2026-08-31T16:00:00Z',
        updatedAt: '2026-08-31T17:00:00Z',
      },
      {
        id: 2,
        code: 'LAP01',
        description: 'Zulu',
        stock: 2,
        createdAt: '2026-08-31T15:00:00Z',
        updatedAt: '2026-08-31T18:00:00Z',
      },
    ]);
    fixture.detectChanges();

    const rows = () => fixture.nativeElement.querySelectorAll('tbody tr');
    expect(rows()[0].textContent).toContain('LAP01');

    const sortHeaders = fixture.nativeElement.querySelectorAll('.mat-sort-header');
    expect(sortHeaders).toHaveLength(5);

    (sortHeaders[1] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('MON02');

    (sortHeaders[2] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('LAP01');

    (sortHeaders[3] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('LAP01');

    (sortHeaders[4] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('MON02');

    (sortHeaders[0] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('LAP01');

    (sortHeaders[0] as HTMLElement).click();
    fixture.detectChanges();
    expect(rows()[0].textContent).toContain('MON02');
    http.verify();
  });

  it('paginates 10 Products initially and offers 10, 25, and 50 per page', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush(
      Array.from({ length: 12 }, (_, index) => ({
        id: index + 1,
        code: `AAA${String(index + 1).padStart(2, '0')}`,
        description: `Product ${index + 1}`,
        stock: index,
        createdAt: '2026-08-31T15:00:00Z',
        updatedAt: '2026-08-31T15:00:00Z',
      })),
    );
    fixture.detectChanges();

    const rows = () => fixture.nativeElement.querySelectorAll('tbody tr');
    expect(rows()).toHaveLength(10);

    const pageSize = fixture.nativeElement.querySelector(
      'select[aria-label="Produtos por página"]',
    ) as HTMLSelectElement;
    expect(Array.from(pageSize.options).map((option) => option.value)).toEqual(['10', '25', '50']);

    const nextPage = fixture.nativeElement.querySelector(
      'button[aria-label="Próxima página"]',
    ) as HTMLButtonElement;
    nextPage.click();
    fixture.detectChanges();
    expect(rows()).toHaveLength(2);

    pageSize.value = '25';
    pageSize.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    expect(rows()).toHaveLength(12);
    http.verify();
  });

  it('shows loading state, disables actions, then shows empty-list state', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.querySelector('mat-spinner')).not.toBeNull();
    expect(page.querySelector('button[disabled]')).not.toBeNull();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush([]);
    fixture.detectChanges();

    expect(page.textContent).toContain('Nenhum produto cadastrado');
    http.verify();
  });

  it('formats dates for Brazil and shows empty-filter state', () => {
    const fixture = TestBed.createComponent(ProductList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne('http://localhost:8081/products').flush([
      {
        id: 1,
        code: 'LAP01',
        description: 'Laptop',
        stock: 7,
        createdAt: '2026-08-31T15:00:00Z',
        updatedAt: '2026-08-31T16:30:00Z',
      },
    ]);
    fixture.detectChanges();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.textContent).toContain('31/08/2026 12:00');
    expect(page.textContent).toContain('31/08/2026 13:30');

    const search = page.querySelector('input') as HTMLInputElement;
    search.value = 'inexistente';
    search.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(page.textContent).toContain('Nenhum produto encontrado');
    expect(page.querySelector('table')).toBeNull();
    http.verify();
  });
});

function flushProducts(
  products: { id: number; code: string; description: string; stock: number }[],
) {
  const http = TestBed.inject(HttpTestingController);
  http.expectOne('http://localhost:8081/products').flush(
    products.map((product) => ({
      ...product,
      createdAt: '2026-08-31T15:00:00Z',
      updatedAt: '2026-08-31T15:00:00Z',
    })),
  );
  return http;
}

function deleteButton(page: HTMLElement) {
  return Array.from(page.querySelectorAll('button')).find((button) =>
    button.textContent?.includes('Excluir'),
  ) as HTMLButtonElement;
}

function dialogButton(label: string) {
  return Array.from(document.querySelectorAll('[role="dialog"] button')).find((button) =>
    button.textContent?.includes(label),
  ) as HTMLButtonElement;
}
