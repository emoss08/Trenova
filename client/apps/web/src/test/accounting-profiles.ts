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
  inboundPaymentsAvailable: true,
  inboundUnavailableReason: null,
  webhookSubscriptions: false,
  revokesTokens: true,
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
  inboundPaymentsAvailable: true,
  inboundUnavailableReason: null,
  webhookSubscriptions: false,
  revokesTokens: true,
};

export const BUSINESS_CENTRAL_INBOUND_REASON =
  "Business Central's API does not let Trenova read posted payments or which invoices they paid, so payments recorded in Business Central are not brought in. An invoice paid there shows as a balance difference on the drift page.";

export const businessCentralProfile: AccountingProviderProfile = {
  appName: "Microsoft Entra app",
  webhookKeyLabel: "",
  environments: ["Production"],
  ledgerAvailable: false,
  ledgerUnavailableReason:
    "Business Central journal lines cannot post to a customer or vendor or apply to an invoice, so Trenova's journals cannot be sent to Business Central as they are. Send documents instead.",
  callbackCarriesCompany: false,
  callbackPath: "/admin/integrations/business-central/callback",
  lineKind: "Item",
  referenceKinds: ["Account", "Item", "Customer", "Vendor", "Term"],
  inboundPaymentsAvailable: false,
  inboundUnavailableReason: BUSINESS_CENTRAL_INBOUND_REASON,
  webhookSubscriptions: true,
  revokesTokens: false,
};
