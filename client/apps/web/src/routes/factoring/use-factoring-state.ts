import type { FactoringVerdict } from "@trenova/graphql/generated/graphql";
import { parseAsStringLiteral } from "nuqs";

export const FACTORING_TABS = ["ready", "not-ready"] as const;
export type FactoringTab = (typeof FACTORING_TABS)[number];

export const factoringSearchParamsParser = {
  view: parseAsStringLiteral(FACTORING_TABS).withDefault("ready"),
};

const TAB_VERDICTS: Record<FactoringTab, FactoringVerdict[]> = {
  ready: ["Ready"],
  "not-ready": ["NotReady"],
};

export function factoringTabVerdicts(tab: FactoringTab): FactoringVerdict[] {
  return TAB_VERDICTS[tab];
}
