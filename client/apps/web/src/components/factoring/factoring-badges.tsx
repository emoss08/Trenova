import {
  FACTORING_AGREEMENT_STATUS_LABELS,
  FACTORING_NOA_STATUS_LABELS,
  FACTORING_SUBMISSION_ITEM_STATUS_LABELS,
  FACTORING_SUBMISSION_STATUS_LABELS,
  FACTORING_VERDICT_LABELS,
  factoringAgreementPhase,
  factoringNoaDisplayStatus,
  factoringNoaPhase,
  factoringSubmissionItemPhase,
  factoringSubmissionPhase,
  factoringVerdictPhase,
} from "@/lib/factoring";
import {
  FACTORING_REMITTANCE_MATCH_STATUS_LABELS,
  FACTORING_REMITTANCE_STATUS_LABELS,
  factoringRemittanceMatchPhase,
  factoringRemittancePhase,
} from "@/lib/factoring-remittance";
import type {
  FactoringAgreementStatus,
  FactoringNoaStatus,
  FactoringRemittanceMatchStatus,
  FactoringRemittanceStatus,
  FactoringSubmissionItemStatus,
  FactoringSubmissionStatus,
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

export function FactoringSubmissionStatusBadge({ status }: { status: FactoringSubmissionStatus }) {
  return (
    <Badge variant={phaseTone(factoringSubmissionPhase(status))}>
      {FACTORING_SUBMISSION_STATUS_LABELS[status]}
    </Badge>
  );
}

export function FactoringSubmissionItemStatusBadge({
  status,
}: {
  status: FactoringSubmissionItemStatus;
}) {
  return (
    <Badge variant={phaseTone(factoringSubmissionItemPhase(status))}>
      {FACTORING_SUBMISSION_ITEM_STATUS_LABELS[status]}
    </Badge>
  );
}

export function FactoringRemittanceStatusBadge({ status }: { status: FactoringRemittanceStatus }) {
  return (
    <Badge variant={phaseTone(factoringRemittancePhase(status))}>
      {FACTORING_REMITTANCE_STATUS_LABELS[status]}
    </Badge>
  );
}

export function FactoringRemittanceMatchBadge({
  status,
}: {
  status: FactoringRemittanceMatchStatus;
}) {
  return (
    <Badge variant={phaseTone(factoringRemittanceMatchPhase(status))}>
      {FACTORING_REMITTANCE_MATCH_STATUS_LABELS[status]}
    </Badge>
  );
}
