import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import {
  AGENT_ACCENTS,
  AGENT_ICONS,
  agentMonogram,
  resolveAgentIdentity,
  type AgentIdentityInput,
} from "./agent-identity";

const SIZES = {
  xs: "size-5 rounded-[5px] text-3xs [&_svg.agent-glyph]:size-2.5",
  sm: "size-6 rounded-md text-3xs [&_svg.agent-glyph]:size-3",
  md: "size-7 rounded-md text-2xs [&_svg.agent-glyph]:size-3.5",
  lg: "size-9 rounded-lg text-xs [&_svg.agent-glyph]:size-4",
  xl: "size-11 rounded-xl text-sm [&_svg.agent-glyph]:size-5",
} as const;

/** Below this the ring has no room to read as an arc; it reads as grit. */
const SIGIL_SIZES = new Set(["md", "lg", "xl"]);

export type AgentTileSize = keyof typeof SIZES;

type AgentTileProps = {
  agent: AgentIdentityInput | null | undefined;
  size?: AgentTileSize;
  className?: string;
};

/**
 * An agent's mark. The same everywhere the agent appears, so a person
 * recognises it in a list, in the composer and halfway down a thread.
 *
 * It is three things layered, and each earns its place:
 *
 * The glyph is the agent's chosen icon — or its initials, when nothing has
 * been chosen. That fallback matters more than it sounds: an organization
 * that has not picked icons has every agent wearing the same robot, and a
 * mark that is identical across four desks is not a mark.
 *
 * The arc is the sigil, derived from the id. It is what tells two agents
 * apart when the hash has handed them the same accent and neither has an
 * icon. One line, one weight, no meaning to decode.
 *
 * The fill and the hairline are the surface, and they are lines and tints
 * rather than depth — a mark that casts a shadow is a mark that cannot sit
 * in a table cell.
 */
export function AgentTile({ agent, size = "md", className }: AgentTileProps) {
  const { icon, accent, iconChosen, sigil } = resolveAgentIdentity(agent ?? {});
  const Icon = AGENT_ICONS[icon];
  const monogram = iconChosen ? "" : agentMonogram(agent?.name);

  return (
    <span
      aria-hidden
      className={cn(
        "agent-tile relative inline-flex shrink-0 items-center justify-center",
        SIZES[size],
        className,
      )}
      style={{ "--agent-tile-color": AGENT_ACCENTS[accent] } as CSSProperties}
    >
      {SIGIL_SIZES.has(size) && (
        <svg
          className="agent-sigil pointer-events-none absolute inset-0 size-full"
          viewBox="0 0 32 32"
          fill="none"
          focusable="false"
        >
          <circle
            cx="16"
            cy="16"
            r="14.25"
            pathLength="100"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeDasharray={`${sigil.length} ${100 - sigil.length}`}
            transform={`rotate(${sigil.rotation - 90} 16 16)`}
          />
        </svg>
      )}

      {monogram === "" ? (
        <Icon className="agent-glyph" />
      ) : (
        <span className="font-semibold leading-none tracking-tight">{monogram}</span>
      )}
    </span>
  );
}
