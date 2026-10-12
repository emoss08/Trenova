import type { AgentEgressClass } from "@/lib/graphql/agent-safety";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { EGRESS_HUE, egressLabel, egressSegments } from "./safety-model";

type EgressMapProps = {
  counts: readonly { egress: AgentEgressClass; count: number }[];
  active: AgentEgressClass | null;
  onPick: (egress: AgentEgressClass | null) => void;
};

/**
 * The tools that change something, laid out by the widest audience their work reaches.
 * Picking an audience narrows the tool rules to it; picking it again clears that.
 */
export function EgressMap({ counts, active, onPick }: EgressMapProps) {
  const t = useT();
  const segments = egressSegments(counts);

  return (
    <div className={cn("egm", active && "foc")} role="group" aria-label={t("Who sees the work")}>
      {segments.map((segment) => {
        const { hue, chroma } = EGRESS_HUE[segment.egress];
        const on = active === segment.egress;
        return (
          <button
            key={segment.egress}
            type="button"
            aria-pressed={on}
            className={cn("egm-s", on && "on")}
            style={{ flexGrow: segment.grow, "--h": hue, "--c": chroma } as CSSProperties}
            onClick={() => onPick(on ? null : segment.egress)}
          >
            <span className="egm-l">
              <b className="mono">{segment.count}</b>
              <span>{egressLabel(t, segment.egress)}</span>
            </span>
            <span className="egm-b" />
          </button>
        );
      })}
    </div>
  );
}
