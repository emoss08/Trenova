import type { FactoringVerdict } from "@trenova/graphql/generated/graphql";
import { parseAsStringLiteral } from "nuqs";

export const FACTORING_TABS = ["ready", "not-ready", "submitted", "funded", "schedules"] as const;
export type FactoringTab = (typeof FACTORING_TABS)[number];
export type FactoringQueueTab = Exclude<FactoringTab, "schedules">;

export const factoringSearchParamsParser = {
  view: parseAsStringLiteral(FACTORING_TABS).withDefault("ready"),
};

const TAB_VERDICTS: Record<FactoringQueueTab, FactoringVerdict[]> = {
  ready: ["Ready"],
  "not-ready": ["NotReady"],
  submitted: ["Submitted"],
  funded: ["Funded"],
};

export function factoringTabVerdicts(tab: FactoringQueueTab): FactoringVerdict[] {
  return TAB_VERDICTS[tab];
}
