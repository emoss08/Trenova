import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { phaseTone, type BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type {
  AiCorrectionOutcome,
  ExtractionEvalCaseStatus,
  ExtractionEvalResultStatus,
  ExtractionEvalRunStatus,
  ExtractionShadowResultStatus,
  ExtractionShadowVerdict,
} from "@trenova/graphql/generated/graphql";
import { CASE_STATUS, OUTCOME, RESULT_STATUS, RUN_STATUS } from "./extraction-model";
import { SHADOW_STATUS, SHADOW_VERDICT } from "./shadow-model";

function PhaseBadge({ attrs, t }: { attrs: BadgeAttrProps; t: TranslateFn }) {
  return (
    <Badge
      variant={phaseTone(attrs.phase)}
      title={attrs.description ? t(attrs.description) : undefined}
    >
      {t(attrs.text)}
    </Badge>
  );
}

export function CaseStatusBadge({ value, t }: { value: ExtractionEvalCaseStatus; t: TranslateFn }) {
  return <PhaseBadge attrs={CASE_STATUS[value]} t={t} />;
}

export function RunStatusBadge({ value, t }: { value: ExtractionEvalRunStatus; t: TranslateFn }) {
  return <PhaseBadge attrs={RUN_STATUS[value]} t={t} />;
}

export function ResultStatusBadge({
  value,
  t,
}: {
  value: ExtractionEvalResultStatus;
  t: TranslateFn;
}) {
  return <PhaseBadge attrs={RESULT_STATUS[value]} t={t} />;
}

export function OutcomeBadge({ value, t }: { value: AiCorrectionOutcome; t: TranslateFn }) {
  return <PhaseBadge attrs={OUTCOME[value]} t={t} />;
}

export function ShadowStatusBadge({
  value,
  t,
}: {
  value: ExtractionShadowResultStatus;
  t: TranslateFn;
}) {
  return <PhaseBadge attrs={SHADOW_STATUS[value]} t={t} />;
}

export function ShadowVerdictBadge({
  value,
  t,
}: {
  value: ExtractionShadowVerdict;
  t: TranslateFn;
}) {
  return <PhaseBadge attrs={SHADOW_VERDICT[value]} t={t} />;
}
