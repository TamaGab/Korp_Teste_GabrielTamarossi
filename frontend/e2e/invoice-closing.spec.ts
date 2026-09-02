import { expect, test } from '@playwright/test';

test('user closes an Invoice and consumes real stock', async ({ page, request }) => {
  const inventoryAPI = process.env['INVENTORY_API_URL'] ?? 'http://localhost:8081';
  const seed = Date.now() + Math.floor(Math.random() * 1000);
  const letters = Array.from({ length: 3 }, (_, index) =>
    String.fromCharCode(65 + (Math.floor(seed / 26 ** index) % 26)),
  ).join('');
  const productCode = `${letters}${(seed % 100).toString().padStart(2, '0')}`;
  let productID: number | undefined;

  try {
    await page.addInitScript(() => {
      window.print = () => undefined;
    });
    await page.goto('/products/new');
    await page.getByLabel('Código do produto').fill(productCode);
    await page.getByLabel('Descrição').fill('Produto do teste completo');
    await page.getByLabel('Estoque disponível').fill('5');
    await page.getByRole('button', { name: 'Cadastrar produto' }).click();
    await page.waitForURL('**/products');

    const productsResponse = await request.get(`${inventoryAPI}/products`);
    expect(productsResponse.ok()).toBeTruthy();
    const products = (await productsResponse.json()) as Array<{ id: number; code: string }>;
    productID = products.find((product) => product.code === productCode)?.id;
    expect(productID).toBeDefined();

    await page.goto('/invoices/new');
    await page.getByLabel('Produto', { exact: true }).selectOption(productCode);
    await page.getByLabel('Quantidade').fill('2');
    await page.getByRole('button', { name: 'Cadastrar nota fiscal' }).click();

    const confirmation = page.getByRole('heading', {
      name: /Nota fiscal \d+ cadastrada com sucesso\./,
    });
    await expect(confirmation).toBeVisible();
    const invoiceNumber = (await confirmation.textContent())?.match(/\d+/)?.[0];
    expect(invoiceNumber).toBeTruthy();

    await page.goto(`/invoices/${invoiceNumber}`);
    await page.getByRole('button', { name: 'Imprimir' }).click();
    await expect(page.locator('.status')).toHaveText('Fechada');

    await page.goto('/products');
    const productRow = page.getByRole('row').filter({ hasText: productCode });
    await expect(productRow.locator('.mat-column-stock')).toHaveText('3');
  } finally {
    if (productID !== undefined) {
      await request.delete(`${inventoryAPI}/products/${productID}`);
    }
  }
});
