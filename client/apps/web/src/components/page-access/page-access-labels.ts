import type { PageAccessDecision, PageAccessUse } from "@/lib/graphql/page-access";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** How a screen depends on a kind of record, as the panel and the role editor name it. */
export const PAGE_ACCESS_USE_LABELS = defineLabels({
  REQUIRED: "Required",
  READ: "Reads",
  DECLARED: "Asks for more",
  REFERENCE: "Reference view",
}) satisfies Record<PageAccessUse, string>;

export const PAGE_ACCESS_USE_HELP = defineLabels({
  REQUIRED: "The screen's route, or an endpoint it calls, checks this.",
  READ: "The screen lists or opens these records itself, which needs their read.",
  DECLARED: "The screen asks for fields beyond the reference view, which need their read.",
  REFERENCE: "The screen shows these only through their reference view.",
}) satisfies Record<PageAccessUse, string>;

/** Why a linked field comes back the way it does. */
export const PAGE_ACCESS_DECISION_LABELS = defineLabels({
  VISIBLE: "Shown",
  WITHHELD_KIND: "Needs read",
  WITHHELD_SENSITIVITY: "Above ceiling",
}) satisfies Record<PageAccessDecision, string>;
