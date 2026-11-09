import { parseAsString, parseAsStringLiteral } from "nuqs";

export const FACTORING_COMPANIES_PATH = "/billing/configuration-files/factoring-companies";

export const FACTORING_COMPANY_TABS = [
  "overview",
  "details",
  "agreement",
  "customers",
  "delivery",
  "ledger",
  "activity",
] as const;

export type FactoringCompanyTab = (typeof FACTORING_COMPANY_TABS)[number];

/**
 * The factor page's tab, and the agreement or notice a record link opened on it, so the tab
 * picks that record out.
 */
export const factoringCompanyPageParsers = {
  tab: parseAsStringLiteral(FACTORING_COMPANY_TABS).withDefault("overview"),
  agreement: parseAsString,
  noa: parseAsString,
};

export function factoringCompanyPath(
  factoringCompanyId: string,
  open?: { tab: FactoringCompanyTab; agreement?: string; noa?: string },
): string {
  const path = `${FACTORING_COMPANIES_PATH}/${encodeURIComponent(factoringCompanyId)}`;
  if (!open) return path;
  const params = new URLSearchParams({ tab: open.tab });
  if (open.agreement) params.set("agreement", open.agreement);
  if (open.noa) params.set("noa", open.noa);
  return `${path}?${params.toString()}`;
}
