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

  it('keeps entered values and shows a duplicate code error after failure', () => {
    const fixture = TestBed.createComponent(ProductForm);
    fixture.detectChanges();

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
    expect(page.textContent).toContain('Product Code already exists');
  });
});

function setInput(page: HTMLElement, controlName: string, value: string) {
  const input = page.querySelector(`[formControlName="${controlName}"]`) as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input'));
}
