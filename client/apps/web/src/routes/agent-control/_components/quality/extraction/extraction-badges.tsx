import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { AiCorrectionOutcome } from "@trenova/graphql/generated/graphql";
import { OUTCOME } from "./extraction-model";

export function OutcomeBadge({ value, t }: { value: AiCorrectionOutcome; t: TranslateFn }) {
  const attrs = OUTCOME[value];
  return (
    <Badge
      variant={phaseTone(attrs.phase)}
      title={attrs.description ? t(attrs.description) : undefined}
    >
      {t(attrs.text)}
    </Badge>
  );
}
