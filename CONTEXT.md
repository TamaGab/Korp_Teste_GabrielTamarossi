# Inventory and Billing

This context describes products held in inventory for later use in invoices.

## Language

**Product**:
An item registered in inventory so it can later be referenced by an invoice.
_Avoid_: Item, merchandise

**Product Code**:
A unique identifier for a Product, formed by exactly three uppercase letters followed by two digits.
_Avoid_: SKU, product ID

**Description**:
The human-readable name of a Product.
_Avoid_: Name, product name

**Available Stock**:
The non-negative whole number of units of a Product currently available. It begins with the informed initial quantity and is reduced by quantities used on invoices.
_Avoid_: Balance, quantity

**Invoice**:
An internal document that records Products and their respective quantities for stock usage. It is not a Brazilian electronic tax document and has no tax-authority integration.
_Avoid_: NF-e, electronic invoice

**Invoice Number**:
The unique, increasing identifier assigned when an Invoice is created, displayed with at least four digits starting at `0001`. A number is never reused after assignment.
_Avoid_: Invoice ID, access key

**Open Invoice**:
An Invoice that can still be printed and has not yet reduced Available Stock.
_Avoid_: Draft invoice, pending invoice

**Closed Invoice**:
An Invoice whose printing flow has been completed and whose Product quantities have been deducted from Available Stock. It cannot be printed again.
_Avoid_: Printed invoice, completed invoice

**Invoice Line**:
The association of one Product with the positive whole-number quantity used by an Invoice, reflecting the Product Code and Description from the time the Invoice was created. A Product appears at most once in an Invoice.
_Avoid_: Item, invoice product

**Invoice Closing**:
The all-or-nothing transition of an Open Invoice to a Closed Invoice after its print dialog is dismissed. It succeeds only when every referenced Product still exists and has sufficient Available Stock.
_Avoid_: Printing, completion

**Pending Invoice Closing**:
The recoverable condition of an Open Invoice whose print dialog has been dismissed but whose Invoice Closing has not yet completed. It is not a third Invoice status and remains until closing succeeds through an initial attempt or retry.
_Avoid_: Processing status, partially closed invoice

**Stock Consumption**:
The all-or-nothing reduction of Available Stock associated with one Invoice Closing. The same Invoice can cause Stock Consumption at most once, even when closing is retried after a failure.
_Avoid_: Stock update, balance deduction
