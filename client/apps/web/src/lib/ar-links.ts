/** Where a customer's statement opens: what they owe, aged, with every open item and payment. */
export function customerStatementPath(customerId: string): string {
  return `/accounting/ar/customer-statement/${encodeURIComponent(customerId)}`;
}
