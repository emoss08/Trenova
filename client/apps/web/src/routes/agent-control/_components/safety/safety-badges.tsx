import type {
  AgentEgressClass,
  AgentToolAutonomy,
  AgentToolPolicy,
} from "@/lib/graphql/agent-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import {
  ANSWER_BADGE,
  EGRESS_ACCENT,
  EGRESS_ORDER,
  answerLabel,
  egressLabel,
  heldByLabel,
} from "./safety-model";

export function EgressBadge({ egress }: { egress: AgentEgressClass }) {
  const t = useT();
  return <Badge variant={EGRESS_ACCENT[egress]}>{egressLabel(t, egress)}</Badge>;
}

export function EgressBadges({ policy }: { policy: AgentToolPolicy }) {
  const ordered = EGRESS_ORDER.filter((egress) => policy.egress.includes(egress));
  return (
    <span className="flex flex-wrap gap-1">
      {ordered.map((egress) => (
        <EgressBadge key={egress} egress={egress} />
      ))}
    </span>
  );
}

export function AnswerBadge({ autonomy }: { autonomy: AgentToolAutonomy }) {
  const t = useT();
  const { phase, appearance } = ANSWER_BADGE[autonomy.answer];
  return (
    <Badge variant={phaseTone(phase)} appearance={appearance}>
      {answerLabel(t, autonomy.answer)}
    </Badge>
  );
}

/** Why a call goes no further, one chip per reason. */
export function HeldByChips({ heldBy }: { heldBy: readonly string[] }) {
  const t = useT();
  if (heldBy.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }
  return (
    <span className="flex flex-wrap gap-1">
      {heldBy.map((key) => (
        <Badge key={key} variant="neutral" appearance="outline">
          {heldByLabel(t, key)}
        </Badge>
      ))}
    </span>
  );
}
