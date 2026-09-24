import type { AgentAutonomyAnswer, AgentEgressClass } from "@/lib/graphql/agent-safety";
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

/** Who sees a tool's work, one badge a class, nowhere first and money last. */
export function EgressBadges({ egress }: { egress: readonly AgentEgressClass[] }) {
  const ordered = EGRESS_ORDER.filter((entry) => egress.includes(entry));
  return (
    <span className="flex flex-wrap gap-1">
      {ordered.map((entry) => (
        <EgressBadge key={entry} egress={entry} />
      ))}
    </span>
  );
}

export function AnswerBadge({ answer }: { answer: AgentAutonomyAnswer }) {
  const t = useT();
  const { phase, appearance } = ANSWER_BADGE[answer];
  return (
    <Badge variant={phaseTone(phase)} appearance={appearance}>
      {answerLabel(t, answer)}
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
