import type { FactoringVerdict } from "@trenova/graphql/generated/graphql";
import { parseAsString, parseAsStringLiteral } from "nuqs";

export const FACTORING_TABS = [
  "ready",
  "not-ready",
  "submitted",
  "funded",
  "schedules",
  "remittances",
] as const;
export type FactoringTab = (typeof FACTORING_TABS)[number];
export type FactoringQueueTab = Exclude<FactoringTab, "schedules" | "remittances">;

/** The workspace's tab, and the one factor it is narrowed to when a factor's page opened it. */
export const factoringSearchParamsParser = {
  view: parseAsStringLiteral(FACTORING_TABS).withDefault("ready"),
  factor: parseAsString,
};

export function factoringWorkspacePath(view: FactoringTab, factoringCompanyId?: string): string {
  const params = new URLSearchParams({ view });
  if (factoringCompanyId) params.set("factor", factoringCompanyId);
  return `/billing/factoring?${params.toString()}`;
}

const TAB_VERDICTS: Record<FactoringQueueTab, FactoringVerdict[]> = {
  ready: ["Ready"],
  "not-ready": ["NotReady"],
  submitted: ["Submitted"],
  funded: ["Funded"],
};

export function factoringTabVerdicts(tab: FactoringQueueTab): FactoringVerdict[] {
  return TAB_VERDICTS[tab];
}
