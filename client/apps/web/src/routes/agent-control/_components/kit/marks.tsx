import {
  AGENT_ICONS,
  agentMonogram,
  resolveAgentIdentity,
  type AgentAccentName,
  type AgentIdentityInput,
} from "@/components/agent-identity/agent-identity";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

/** The hue token each accent draws its tile from; slate is the neutral tile. */
export const ACCENT_HUE: Record<AgentAccentName, string | null> = {
  indigo: "var(--hue-brand)",
  teal: "var(--hue-teal)",
  amber: "var(--hue-amber)",
  rose: "var(--hue-rose)",
  emerald: "var(--hue-emerald)",
  sky: "var(--hue-sky)",
  violet: "var(--hue-violet)",
  slate: null,
};

type TileProps = {
  agent: AgentIdentityInput | null | undefined;
  /** Size in pixels. */
  s?: number;
  className?: string;
};

/** An agent's tile: its icon on a tint of its accent, or a neutral tile for slate. */
export function Tile({ agent, s = 30, className }: TileProps) {
  const { icon, accent, iconChosen } = resolveAgentIdentity(agent ?? {});
  const Icon = AGENT_ICONS[icon];
  const hue = ACCENT_HUE[accent];
  const style = {
    "--h": hue ?? 0,
    width: s,
    height: s,
    borderRadius: Math.round(s * 0.28),
  } as CSSProperties;

  return (
    <span aria-hidden className={cn("tile", !hue && "n", className)} style={style}>
      {iconChosen ? (
        <Icon size={Math.round(s * 0.5)} strokeWidth={1.7} />
      ) : (
        <span style={{ fontSize: Math.round(s * 0.36), fontWeight: 600 }}>
          {agentMonogram(agent?.name)}
        </span>
      )}
    </span>
  );
}

type MarkTone = { hue: string; chroma: number } | null;

/** Known vendors keep their colour; anything else hashes to one of the accent hues. */
const VENDOR_TONES: { match: RegExp; abbr: string | null; tone: MarkTone }[] = [
  { match: /anthropic|claude/i, abbr: "A", tone: { hue: "var(--hue-amber)", chroma: 0.09 } },
  { match: /azure/i, abbr: "Az", tone: { hue: "var(--hue-sky)", chroma: 0.1 } },
  { match: /openai|gpt/i, abbr: "O", tone: { hue: "var(--hue-emerald)", chroma: 0.06 } },
  { match: /ollama/i, abbr: "Ol", tone: null },
  { match: /gemini|google/i, abbr: "G", tone: { hue: "var(--hue-brand)", chroma: 0.12 } },
  { match: /groq/i, abbr: "Gq", tone: { hue: "var(--hue-rose)", chroma: 0.14 } },
  { match: /mistral/i, abbr: "Mi", tone: { hue: "var(--hue-amber)", chroma: 0.15 } },
  { match: /vllm/i, abbr: null, tone: { hue: "var(--hue-brand)", chroma: 0.1 } },
];

const HASHED_TONES: MarkTone[] = [
  { hue: "var(--hue-brand)", chroma: 0.1 },
  { hue: "var(--hue-teal)", chroma: 0.09 },
  { hue: "var(--hue-violet)", chroma: 0.12 },
  { hue: "var(--hue-amber)", chroma: 0.1 },
  { hue: "var(--hue-rose)", chroma: 0.12 },
];

function vendorOf(name: string) {
  return VENDOR_TONES.find((entry) => entry.match.test(name));
}

function toneOf(name: string): MarkTone {
  const vendor = vendorOf(name);
  if (vendor) {
    return vendor.tone;
  }
  let hash = 0;
  for (const char of name) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  }
  return HASHED_TONES[hash % HASHED_TONES.length];
}

/** The initials a provider mark shows: one letter for one word, two for more. */
export function markInitials(name: string): string {
  const words = name
    .replace(/[^\p{L}\p{N}\s]/gu, " ")
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) {
    return "?";
  }
  if (words.length === 1) {
    return words[0][0].toUpperCase();
  }
  return (words[0][0] + words[1][0]).toUpperCase();
}

type MarkProps = {
  provider: { name: string };
  s?: number;
  className?: string;
};

/** A provider's mark: its initials on a tint of its vendor's hue. */
export function Mark({ provider, s = 28, className }: MarkProps) {
  const initials = vendorOf(provider.name)?.abbr ?? markInitials(provider.name);
  const tone = toneOf(provider.name);
  const style = {
    "--h": tone?.hue ?? 0,
    "--c": tone?.chroma ?? 0,
    width: s,
    height: s,
    fontSize: Math.max(9, Math.round(s * (initials.length > 1 ? 0.36 : 0.46))),
    borderRadius: Math.round(s * 0.26),
  } as CSSProperties;

  return (
    <span aria-hidden className={cn("mk", !tone && "n", className)} style={style}>
      {initials}
    </span>
  );
}
