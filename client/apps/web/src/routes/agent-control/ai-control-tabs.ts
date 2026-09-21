import { parseAsStringLiteral } from "nuqs";

export const aiControlTabValues = [
  "overview",
  "agents",
  "providers",
  "memory",
  "activity",
] as const;
export type AIControlTab = (typeof aiControlTabValues)[number];

export const AI_CONTROL_TAB_PARAM = "tab";

export const aiControlTabParser = parseAsStringLiteral(aiControlTabValues)
  .withOptions({ history: "push", shallow: true })
  .withDefault("overview");

export const activityViews = ["runs", "proposals", "plans", "evaluations", "exceptions"] as const;

export const ACTIVITY_VIEW_PARAM = "activity";

export const activityViewParser = parseAsStringLiteral(activityViews)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("runs");
