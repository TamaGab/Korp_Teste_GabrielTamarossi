import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { TestBed } from "@angular/core/testing";
import {
  ActivatedRoute,
  convertToParamMap,
  provideRouter,
} from "@angular/router";
import { InvoiceDetail } from "./invoice-detail";

describe("InvoiceDetail", () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InvoiceDetail],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ number: "0001" }) },
          },
        },
      ],
    }).compileComponents();
  });

  it("shows an Invoice and each historical Product line", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    const request = http.expectOne("http://localhost:8082/invoices/0001");
    expect(request.request.method).toBe("GET");
    request.flush({
      number: "0001",
      status: "CLOSED",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [
        { code: "LAP01", description: "Laptop original", quantity: 2 },
        { code: "MON01", description: "Monitor original", quantity: 3 },
      ],
    });
    fixture.detectChanges();

    const page = fixture.nativeElement as HTMLElement;
    expect(page.textContent).toContain("Nota fiscal 0001");
    expect(page.textContent).toContain("Fechada");
    expect(page.textContent).toContain("LAP01");
    expect(page.textContent).toContain("Laptop original");
    expect(page.textContent).toContain("2");
    expect(page.textContent).toContain("MON01");
    expect(page.querySelectorAll("tbody tr")).toHaveLength(2);
    http.verify();
  });

  it("shows a loading state while the Invoice is requested", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Carregando nota fiscal...",
    );

    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [],
    });
    http.verify();
  });

  it("shows an empty state when an Invoice has no Product lines", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [],
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Nenhum produto nesta nota fiscal",
    );
    expect(fixture.nativeElement.querySelector("table")).toBeNull();
    http.verify();
  });

  it("shows Portuguese not-found feedback for a missing Invoice", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http
      .expectOne("http://localhost:8082/invoices/0001")
      .flush(
        { error: "invoice not found" },
        { status: 404, statusText: "Not Found" },
      );
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Nota fiscal não encontrada.",
    );
    expect(
      fixture.nativeElement.querySelector('[role="alert"]'),
    ).not.toBeNull();
    http.verify();
  });

  it("shows Portuguese feedback when the Invoice cannot be loaded", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();

    const http = TestBed.inject(HttpTestingController);
    http
      .expectOne("http://localhost:8082/invoices/0001")
      .flush(
        { error: "unavailable" },
        { status: 503, statusText: "Unavailable" },
      );
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain(
      "Não foi possível carregar a nota fiscal. Tente novamente.",
    );
    http.verify();
  });
});
