import { useAccountingMappingLabels } from "@/hooks/use-accounting-mapping-labels";
import { accountingMappingPhase } from "@/lib/accounting-sync";
import type {
  AccountingMappingSource,
  AccountingMappingState,
} from "@trenova/graphql/generated/graphql";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { phaseTone } from "@trenova/shared/lib/status-phase";

export function MappingStateBadge({
  state,
  source,
}: {
  state: AccountingMappingState;
  source?: AccountingMappingSource | null;
}) {
  const labels = useAccountingMappingLabels();
  const suggested = state === "Proposed" && (source === "Suggested" || source === "Model");

  return (
    <Badge variant={phaseTone(accountingMappingPhase(state))}>
      {suggested ? <AssistMark aria-hidden className="size-3" /> : null}
      {labels.state(state)}
    </Badge>
  );
}
