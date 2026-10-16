import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { WatchtowerAgent, WatchtowerAgentWork } from "@/lib/graphql/watchtower";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

/** How far along the agent is, from 0 to 1; a finished agent is all the way round. */
export function agentProgress(
  work: Pick<WatchtowerAgentWork, "status" | "step" | "steps">,
): number {
  if (work.status === "Done" || work.steps.length === 0) {
    return 1;
  }
  return Math.min(1, work.step / work.steps.length);
}

/**
 * An agent's mark with a ring round it that fills as its work goes on: blue
 * while it works, amber when it waits on a person, green once it is done.
 */
export function AgentRing({
  work,
  agent,
  size = 22,
}: {
  work: Pick<WatchtowerAgentWork, "status" | "step" | "steps">;
  agent: WatchtowerAgent | null;
  size?: number;
}) {
  const box = size + 8;
  const center = box / 2;
  const radius = size / 2 + 2.5;
  const circumference = 2 * Math.PI * radius;
  const style: CSSProperties = { width: size, height: size };

  return (
    <span className={cn("dk-wt-ar", `dk-st-${work.status}`)} style={style}>
      <DeskAgentTile agent={agent} size="xs" />
      <svg width={box} height={box} viewBox={`0 0 ${box} ${box}`} aria-hidden fill="none">
        <circle
          cx={center}
          cy={center}
          r={radius}
          stroke="currentColor"
          strokeOpacity={0.18}
          strokeWidth={1.6}
        />
        <circle
          cx={center}
          cy={center}
          r={radius}
          stroke="currentColor"
          strokeWidth={1.6}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={circumference * (1 - agentProgress(work))}
          transform={`rotate(-90 ${center} ${center})`}
        />
      </svg>
    </span>
  );
}
