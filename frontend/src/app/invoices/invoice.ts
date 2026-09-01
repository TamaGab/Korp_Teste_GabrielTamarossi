export interface Invoice {
  number: string;
  status: 'OPEN' | 'CLOSED';
  lines: InvoiceLine[];
  createdAt: string;
}

export interface InvoiceLine {
  code: string;
  description: string;
  quantity: number;
}

export interface CreateInvoice {
  lines: CreateInvoiceLine[];
}

export interface CreateInvoiceLine {
  productCode: string;
  quantity: number;
}
