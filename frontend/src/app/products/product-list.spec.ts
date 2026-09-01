import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { ProductList } from './product-list';

describe('ProductList', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ProductList],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
      ],
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
});
