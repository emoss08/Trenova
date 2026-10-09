import type { AgentAutonomyAnswer, AgentEgressClass } from "@/lib/graphql/agent-safety";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import {
  EGRESS_HUE,
  EGRESS_ORDER,
  ANSWER_CLASS,
  answerLabel,
  egressLabel,
  heldByLabel,
} from "./safety-model";

/** Who sees a tool's work, as a chip in that audience's own accent. */
export function EgressBadge({ egress }: { egress: AgentEgressClass }) {
  const t = useT();
  const { hue, chroma } = EGRESS_HUE[egress];

  return (
    <span className="eg" style={{ "--h": hue, "--c": chroma } as CSSProperties}>
      {egressLabel(t, egress)}
    </span>
  );
}

/** Who sees a tool's work, one chip a class, nowhere first and money last. */
export function EgressBadges({ egress }: { egress: readonly AgentEgressClass[] }) {
  const ordered = EGRESS_ORDER.filter((entry) => egress.includes(entry));

  return (
    <span className="hb">
      {ordered.map((entry) => (
        <EgressBadge key={entry} egress={entry} />
      ))}
    </span>
  );
}

/** How much a person stands between the agent and the change. */
export function AnswerBadge({ answer }: { answer: AgentAutonomyAnswer }) {
  const t = useT();

  return <span className={cn("an", ANSWER_CLASS[answer])}>{answerLabel(t, answer)}</span>;
}

/** Why a call goes no further, one tag per reason. */
export function HeldByChips({ heldBy }: { heldBy: readonly string[] }) {
  const t = useT();
  if (heldBy.length === 0) {
    return <span className="dim">—</span>;
  }

  return (
    <span className="hb">
      {heldBy.map((key) => (
        <span key={key} className="tg">
          {heldByLabel(t, key)}
        </span>
      ))}
    </span>
  );
}
