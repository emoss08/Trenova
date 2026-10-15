import {
  FACTORING_REMITTANCE_MATCH_STATUS_LABELS,
  FACTORING_REMITTANCE_STATUS_LABELS,
  factoringRemittanceMatchPhase,
  factoringRemittancePhase,
} from "@/lib/factoring-remittance";
import type {
  FactoringRemittanceMatchStatus,
  FactoringRemittanceStatus,
} from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { phaseTone } from "@trenova/shared/lib/status-phase";

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
