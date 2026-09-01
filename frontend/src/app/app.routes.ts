import { Routes } from "@angular/router";

export const routes: Routes = [
  {
    path: "",
    loadComponent: () => import("./home/home").then((module) => module.Home),
  },
  {
    path: "products",
    loadComponent: () =>
      import("./products/product-list").then((module) => module.ProductList),
  },
  {
    path: "products/new",
    loadComponent: () =>
      import("./products/product-form").then((module) => module.ProductForm),
  },
  {
    path: "products/:id/edit",
    loadComponent: () =>
      import("./products/product-form").then((module) => module.ProductForm),
  },
  {
    path: "invoices",
    loadComponent: () =>
      import("./invoices/invoice-list").then((module) => module.InvoiceList),
  },
  {
    path: "invoices/new",
    loadComponent: () =>
      import("./invoices/invoice-form").then((module) => module.InvoiceForm),
  },
  {
    path: "invoices/:number",
    loadComponent: () =>
      import("./invoices/invoice-detail").then(
        (module) => module.InvoiceDetail,
      ),
  },
  { path: "**", redirectTo: "" },
];
