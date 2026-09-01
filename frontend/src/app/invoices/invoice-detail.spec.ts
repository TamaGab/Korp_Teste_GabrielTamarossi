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
  afterEach(() => {
    vi.restoreAllMocks();
  });

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

  it("disables printing and ignores additional clicks while checking stock", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [{ code: "LAP01", description: "Laptop", quantity: 2 }],
    });
    fixture.detectChanges();

    const button = fixture.nativeElement.querySelector("button") as HTMLButtonElement;
    button.click();
    button.click();
    fixture.detectChanges();

    expect(button.disabled).toBe(true);
    expect(button.textContent).toContain("Verificando estoque...");
    http.expectOne("http://localhost:8082/invoices/0001/prepare-print");
    http.verify();
  });

  it("opens and prints the prepared HTML for an Open Invoice", () => {
    const printFrame = document.createElement("iframe");
    const printWindow = {
      addEventListener: vi.fn(),
      document: {
        close: vi.fn(),
        open: vi.fn(),
        write: vi.fn(),
      },
      focus: vi.fn(),
      print: vi.fn(),
    };
    Object.defineProperty(printFrame, "contentWindow", { value: printWindow });
    const createElement = document.createElement.bind(document);
    vi.spyOn(document, "createElement").mockImplementation(
      ((tagName: string, options?: ElementCreationOptions) =>
        tagName === "iframe"
          ? printFrame
          : createElement(tagName, options)) as typeof document.createElement,
    );
    const open = vi.spyOn(window, "open");
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [{ code: "LAP01", description: "Laptop", quantity: 2 }],
    });
    fixture.detectChanges();

    (fixture.nativeElement.querySelector("button") as HTMLButtonElement).click();
    http
      .expectOne("http://localhost:8082/invoices/0001/prepare-print")
      .flush({ html: "<html><body>Nota Fiscal 0001</body></html>" });
    fixture.detectChanges();

    expect(open).not.toHaveBeenCalled();
    expect(printWindow.document.write).toHaveBeenCalledWith(
      "<html><body>Nota Fiscal 0001</body></html>",
    );
    expect(printWindow.print).toHaveBeenCalledOnce();
    expect(document.body.contains(printFrame)).toBe(true);
    expect(
      (fixture.nativeElement.querySelector("button") as HTMLButtonElement)
        .disabled,
    ).toBe(false);
    http.verify();
  });

  it("shows every stock problem in Portuguese without opening a window", () => {
    const open = vi.spyOn(window, "open");
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [
        { code: "LAP01", description: "Laptop", quantity: 2 },
        { code: "MON01", description: "Monitor", quantity: 1 },
      ],
    });
    fixture.detectChanges();

    (fixture.nativeElement.querySelector("button") as HTMLButtonElement).click();
    http
      .expectOne("http://localhost:8082/invoices/0001/prepare-print")
      .flush(
        {
          error: "print preparation failed",
          problems: [
            {
              productCode: "LAP01",
              reason: "insufficient_stock",
              availableStock: 1,
              requestedQuantity: 2,
            },
            { productCode: "MON01", reason: "product_not_found" },
          ],
        },
        { status: 422, statusText: "Unprocessable Entity" },
      );
    fixture.detectChanges();

    expect(open).not.toHaveBeenCalled();
    expect(fixture.nativeElement.textContent).toContain(
      "Estoque insuficiente para LAP01: disponível 1, necessário 2.",
    );
    expect(fixture.nativeElement.textContent).toContain(
      "O produto MON01 não está mais disponível no estoque.",
    );
    expect(
      (fixture.nativeElement.querySelector("button") as HTMLButtonElement)
        .disabled,
    ).toBe(false);
    http.verify();
  });

  it("shows inventory unavailability and enables printing again", () => {
    const open = vi.spyOn(window, "open");
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "OPEN",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [{ code: "LAP01", description: "Laptop", quantity: 2 }],
    });
    fixture.detectChanges();

    (fixture.nativeElement.querySelector("button") as HTMLButtonElement).click();
    http
      .expectOne("http://localhost:8082/invoices/0001/prepare-print")
      .flush(
        { error: "inventory unavailable" },
        { status: 502, statusText: "Bad Gateway" },
      );
    fixture.detectChanges();

    expect(open).not.toHaveBeenCalled();
    expect(fixture.nativeElement.textContent).toContain(
      "Não foi possível verificar o estoque. Tente novamente.",
    );
    expect(
      (fixture.nativeElement.querySelector("button") as HTMLButtonElement)
        .disabled,
    ).toBe(false);
    http.verify();
  });

  it("does not offer printing for a Closed Invoice", () => {
    const fixture = TestBed.createComponent(InvoiceDetail);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne("http://localhost:8082/invoices/0001").flush({
      number: "0001",
      status: "CLOSED",
      createdAt: "2026-09-01T15:00:00Z",
      lines: [{ code: "LAP01", description: "Laptop", quantity: 2 }],
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector("button")).toBeNull();
    http.verify();
  });
});
