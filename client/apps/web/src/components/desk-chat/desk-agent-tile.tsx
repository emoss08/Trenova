import {
  AGENT_ICONS,
  agentMonogram,
  resolveAgentIdentity,
  type AgentIdentityInput,
} from "@/components/agent-identity/agent-identity";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

export type DeskAgentTileSize = "xs" | "sm" | "md" | "lg";

const PIXELS: Record<DeskAgentTileSize, number> = { xs: 20, sm: 24, md: 28, lg: 36 };
const GLYPH: Record<DeskAgentTileSize, number> = { xs: 11, sm: 13, md: 15, lg: 18 };

/**
 * An agent's mark as the Desk draws it: the agent's icon on a wash of its
 * accent, or its initials when it has no icon of its own, and at the two
 * larger sizes the short arc that tells two similar agents apart.
 */
export function DeskAgentTile({
  agent,
  size = "md",
  className,
}: {
  agent: AgentIdentityInput | null | undefined;
  size?: DeskAgentTileSize;
  className?: string;
}) {
  const identity = resolveAgentIdentity(agent ?? {});
  const Icon = AGENT_ICONS[identity.icon];
  const px = PIXELS[size];
  const glyph = GLYPH[size];
  const { rotation, length } = identity.sigil;

  return (
    <span
      aria-hidden
      className={cn(
        "dk-at",
        `dk-at-${size}`,
        identity.accent === "slate" && "dk-slate",
        className,
      )}
      style={
        {
          "--dk-ah": `var(--hue-${identity.accent})`,
          width: px,
          height: px,
        } as CSSProperties
      }
    >
      {(size === "md" || size === "lg") && (
        <svg className="dk-at-sig" viewBox="0 0 32 32" fill="none">
          <circle
            cx="16"
            cy="16"
            r="14.25"
            pathLength="100"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeDasharray={`${length} ${100 - length}`}
            transform={`rotate(${rotation - 90} 16 16)`}
          />
        </svg>
      )}
      {identity.iconChosen ? (
        <Icon width={glyph} height={glyph} strokeWidth={2} />
      ) : (
        <b>{agentMonogram(agent?.name)}</b>
      )}
    </span>
  );
}
