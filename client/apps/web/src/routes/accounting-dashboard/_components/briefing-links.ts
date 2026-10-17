import { defineLabels } from "@trenova/shared/i18n/labels";
import type { AccountingBriefingLink } from "@trenova/graphql/generated/graphql";

const OPEN_ITEMS = "/accounting/ar/open-items";
const AGING = "/accounting/ar/aging";
const COLLECTIONS = "/accounting/ar/collections";
const COLLECTIONS_DUE = "/accounting/ar/collections?due=true";

const LINK_PATHS: Record<AccountingBriefingLink, string> = {
  OpenInvoices: OPEN_ITEMS,
  Overdue: OPEN_ITEMS,
  PastNinety: AGING,
  Disputed: COLLECTIONS,
  UnappliedCash: "/accounting/ar/payments",
  TopCustomer: "/accounting/ar/customer-ledger",
  CrossingNinety: AGING,
  DueSoon: OPEN_ITEMS,
  BrokenPromise: COLLECTIONS_DUE,
  PromisesDue: COLLECTIONS,
  Uncontacted: COLLECTIONS_DUE,
  FollowUpsDue: COLLECTIONS_DUE,
  ShortPayRemainders: OPEN_ITEMS,
  Drafts: "/billing/invoices?status=Draft",
  NoCollections: "/accounting/ar/payments",
  DueThisMonth: OPEN_ITEMS,
};

export const BRIEFING_LINK_LABELS = defineLabels({
  OpenInvoices: "Open the open invoices",
  Overdue: "Open the past-due invoices",
  PastNinety: "Open the aging report",
  Disputed: "Open collections",
  UnappliedCash: "Open customer payments",
  TopCustomer: "Open this customer's ledger",
  CrossingNinety: "Open the aging report",
  DueSoon: "Open the invoices falling due",
  BrokenPromise: "Open the missed promise in collections",
  PromisesDue: "Open the promises in collections",
  Uncontacted: "Open the customers due a call",
  FollowUpsDue: "Open the follow-ups due",
  ShortPayRemainders: "Open the short-paid invoices",
  Drafts: "Open the draft invoices",
  NoCollections: "Open customer payments",
  DueThisMonth: "Open the invoices falling due",
} as const satisfies Record<AccountingBriefingLink, string>);

export type BriefingTone = "warning" | "danger";

export const BRIEFING_LINK_TONES: Partial<Record<AccountingBriefingLink, BriefingTone>> = {
  BrokenPromise: "danger",
  CrossingNinety: "danger",
  Uncontacted: "warning",
  FollowUpsDue: "warning",
  Drafts: "warning",
  NoCollections: "danger",
};

/**
 * Where a phrase or action of the accounting brief leads. One about a single
 * customer opens that customer; a customer phrase without one leads nowhere.
 */
export function briefingLinkPath(
  link: AccountingBriefingLink,
  customerId: string | null | undefined,
): string | null {
  if (link === "TopCustomer") {
    return customerId
      ? `${LINK_PATHS.TopCustomer}?customerId=${encodeURIComponent(customerId)}`
      : null;
  }
  if (link === "BrokenPromise" && customerId) {
    return `${COLLECTIONS}?customerId=${encodeURIComponent(customerId)}`;
  }
  return LINK_PATHS[link];
}
