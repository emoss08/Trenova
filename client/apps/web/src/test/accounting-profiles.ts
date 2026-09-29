import type { AccountingProviderProfile } from "@/lib/graphql/accounting-sync";

export const quickBooksProfile: AccountingProviderProfile = {
  appName: "Intuit app",
  webhookKeyLabel: "Webhook verifier token",
  environments: ["Sandbox", "Production"],
  ledgerAvailable: true,
  ledgerUnavailableReason: null,
  callbackCarriesCompany: true,
  callbackPath: "/admin/integrations/quickbooks/callback",
  lineKind: "Item",
  referenceKinds: ["Account", "Item", "Customer", "Vendor", "Term", "PaymentMethod"],
};

export const XERO_LEDGER_REASON =
  "Xero manual journals cannot post to accounts receivable, accounts payable or bank accounts, or name a customer or supplier, so Trenova's journals cannot be sent to Xero as they are. Send documents instead.";

export const xeroProfile: AccountingProviderProfile = {
  appName: "Xero app",
  webhookKeyLabel: "Webhook key",
  environments: ["Production"],
  ledgerAvailable: false,
  ledgerUnavailableReason: XERO_LEDGER_REASON,
  callbackCarriesCompany: false,
  callbackPath: "/admin/integrations/xero/callback",
  lineKind: "Account",
  referenceKinds: ["Account", "Item", "Customer", "Vendor"],
};
