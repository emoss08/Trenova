import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import type { JurisdictionVerificationState } from "@/types/jurisdiction-rule";

export const JURISDICTION_VERIFICATION_LABELS: Record<JurisdictionVerificationState, string> =
  defineLabels({
    Unverified: "Unverified",
    Verified: "Verified",
    Disputed: "Disputed",
  });

/**
 * describeJurisdictionVerification reads a rule's verification as one sentence, so the state
 * and the day it was settled are ordered by each language rather than glued in English order.
 */
export function describeJurisdictionVerification(
  state: JurisdictionVerificationState,
  on: string | null,
): string {
  if (on === null) {
    return JURISDICTION_VERIFICATION_LABELS[state];
  }
  switch (state) {
    case "Verified":
      return translate("Verified on {0}", on);
    case "Disputed":
      return translate("Disputed on {0}", on);
    case "Unverified":
      return translate("Unverified on {0}", on);
  }
}
