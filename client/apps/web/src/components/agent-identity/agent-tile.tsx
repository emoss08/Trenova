import { cn } from "@trenova/shared/lib/utils";
import {
  AGENT_ACCENTS,
  AGENT_ICONS,
  type AgentIdentityInput,
  resolveAgentIdentity,
} from "./agent-identity";

const SIZES = {
  xs: "size-5 rounded-[5px] [&_svg]:size-2.5",
  sm: "size-6 rounded-md [&_svg]:size-3",
  md: "size-7 rounded-md [&_svg]:size-3.5",
  lg: "size-9 rounded-lg [&_svg]:size-4",
  xl: "size-11 rounded-xl [&_svg]:size-5",
} as const;

export type AgentTileSize = keyof typeof SIZES;

type AgentTileProps = {
  agent: AgentIdentityInput | null | undefined;
  size?: AgentTileSize;
  className?: string;
};

/**
 * An agent's mark. One tinted square, the same everywhere the agent appears, so
 * a person recognises it in a list, in the composer and halfway down a thread.
 */
export function AgentTile({ agent, size = "md", className }: AgentTileProps) {
  const { icon, accent } = resolveAgentIdentity(agent ?? {});
  const Icon = AGENT_ICONS[icon];

  return (
    <span
      aria-hidden
      className={cn(
        "agent-tile inline-flex shrink-0 items-center justify-center",
        SIZES[size],
        className,
      )}
      style={{ "--agent-tile-color": AGENT_ACCENTS[accent] } as React.CSSProperties}
    >
      <Icon />
    </span>
  );
}
