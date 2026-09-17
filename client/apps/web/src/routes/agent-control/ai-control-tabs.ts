import { parseAsStringLiteral } from "nuqs";

export const aiControlTabValues = ["overview", "agents", "providers", "activity"] as const;

export type AIControlTab = (typeof aiControlTabValues)[number];

export const AI_CONTROL_TAB_PARAM = "tab";

export const aiControlTabParser = parseAsStringLiteral(aiControlTabValues)
  .withOptions({ history: "push", shallow: true })
  .withDefault("overview");
