import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { TestBed } from "@angular/core/testing";
import { provideRouter } from "@angular/router";
import { InvoiceList } from "./invoice-list";

describe("InvoiceList", () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InvoiceList],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
      ],
    }).compileComponents();
  });

  it("lists Invoices with formatted numbers and Portuguese statuses", () => {
    const fixture = TestBed.createComponent(InvoiceList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    const request = http.expectOne("http://localhost:8082/invoices");
    expect(request.request.method).toBe("GET");
    request.flush([
      { number: "0002", status: "CLOSED", createdAt: "2026-09-01T16:00:00Z" },
      { number: "0001", status: "OPEN", createdAt: "2026-09-01T15:00:00Z" },
    ]);
    fixture.detectChanges();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.textContent).toContain("0002");
    expect(page.textContent).toContain("Fechada");
    expect(page.textContent).toContain("0001");
    expect(page.textContent).toContain("Aberta");
    expect(page.querySelector('a[href="/invoices/0001"]')).not.toBeNull();
    http.verify();
  });

  it("shows a loading state while Invoices are requested", () => {
    const fixture = TestBed.createComponent(InvoiceList);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Carregando notas fiscais...",
    );

    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices").flush([]);
    http.verify();
  });

  it("shows an empty state when no Invoice is registered", () => {
    const fixture = TestBed.createComponent(InvoiceList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices").flush([]);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Nenhuma nota fiscal cadastrada",
    );
    http.verify();
  });

  it("shows Portuguese feedback when Invoices cannot be loaded", () => {
    const fixture = TestBed.createComponent(InvoiceList);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http
      .expectOne("http://localhost:8082/invoices")
      .flush(
        { error: "unavailable" },
        { status: 503, statusText: "Unavailable" },
      );
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Não foi possível carregar as notas fiscais. Tente novamente.",
    );
    expect(
      fixture.nativeElement.querySelector('[role="alert"]'),
    ).not.toBeNull();
    http.verify();
  });
});
