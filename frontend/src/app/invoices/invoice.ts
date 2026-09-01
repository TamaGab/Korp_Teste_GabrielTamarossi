export interface Invoice {
  number: string;
  status: "OPEN" | "CLOSED";
  lines: InvoiceLine[];
  createdAt: string;
}

export interface InvoiceLine {
  code: string;
  description: string;
  quantity: number;
}

export type InvoiceSummary = Omit<Invoice, "lines">;

export interface CreateInvoice {
  lines: CreateInvoiceLine[];
}

export interface CreateInvoiceLine {
  productCode: string;
  quantity: number;
}

export interface PreparedInvoicePrint {
  html: string;
}

export interface PrintPreparationProblem {
  productCode: string;
  reason: "product_not_found" | "insufficient_stock";
  availableStock?: number;
  requestedQuantity: number;
}

export function invoiceStatusLabel(status: Invoice["status"]) {
  return status === "OPEN" ? "Aberta" : "Fechada";
}
