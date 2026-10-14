import {
  FACTORING_AGREEMENT_STATUS_LABELS,
  FACTORING_NOA_STATUS_LABELS,
  FACTORING_VERDICT_LABELS,
  factoringAgreementPhase,
  factoringNoaDisplayStatus,
  factoringNoaPhase,
  factoringVerdictPhase,
} from "@/lib/factoring";
import type {
  FactoringAgreementStatus,
  FactoringNoaStatus,
  FactoringVerdict,
} from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { phaseTone } from "@trenova/shared/lib/status-phase";

export function FactoringVerdictBadge({ verdict }: { verdict: FactoringVerdict }) {
  return (
    <Badge variant={phaseTone(factoringVerdictPhase(verdict))}>
      {FACTORING_VERDICT_LABELS[verdict]}
    </Badge>
  );
}

export function FactoringNoaStatusBadge({
  noa,
  nowUnix,
}: {
  noa: { status: FactoringNoaStatus; expiresAt?: number | null };
  nowUnix: number;
}) {
  const status = factoringNoaDisplayStatus(noa, nowUnix);
  return (
    <Badge variant={phaseTone(factoringNoaPhase(status))}>
      {FACTORING_NOA_STATUS_LABELS[status]}
    </Badge>
  );
}

export function FactoringAgreementStatusBadge({ status }: { status: FactoringAgreementStatus }) {
  return (
    <Badge variant={phaseTone(factoringAgreementPhase(status))}>
      {FACTORING_AGREEMENT_STATUS_LABELS[status]}
    </Badge>
  );
}
