import {
  BellIcon,
  BotIcon,
  ClipboardCheckIcon,
  CompassIcon,
  FileInputIcon,
  GaugeIcon,
  HeadsetIcon,
  type LucideIcon,
  PackageIcon,
  RadarIcon,
  ReceiptTextIcon,
  RouteIcon,
  SearchIcon,
  ShieldCheckIcon,
  TruckIcon,
  WalletCardsIcon,
} from "lucide-react";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

/** The icons an organization can give an agent. Mirrors the Go registry. */
export const AGENT_ICONS = {
  bot: BotIcon,
  truck: TruckIcon,
  route: RouteIcon,
  receipt: ReceiptTextIcon,
  wallet: WalletCardsIcon,
  shield: ShieldCheckIcon,
  headset: HeadsetIcon,
  clipboard: ClipboardCheckIcon,
  compass: CompassIcon,
  radar: RadarIcon,
  gauge: GaugeIcon,
  package: PackageIcon,
  file: FileInputIcon,
  search: SearchIcon,
  bell: BellIcon,
  sparkle: AssistMark,
} as const satisfies Record<string, LucideIcon>;

export type AgentIconName = keyof typeof AGENT_ICONS;

export const AGENT_ICON_ORDER = Object.keys(AGENT_ICONS) as AgentIconName[];

/**
 * Accents an agent can carry. Each maps to a token pair defined once in app.css,
 * so a tile reads the same in light and dark and no component writes a colour.
 */
export const AGENT_ACCENTS = {
  indigo: "var(--agent-indigo)",
  teal: "var(--agent-teal)",
  amber: "var(--agent-amber)",
  rose: "var(--agent-rose)",
  emerald: "var(--agent-emerald)",
  sky: "var(--agent-sky)",
  violet: "var(--agent-violet)",
  slate: "var(--agent-slate)",
} as const;

export type AgentAccentName = keyof typeof AGENT_ACCENTS;

export const AGENT_ACCENT_ORDER = Object.keys(AGENT_ACCENTS) as AgentAccentName[];

/** The icon a starter implies, used when an agent has not chosen one. */
export const TEMPLATE_ICON: Partial<Record<string, AgentIconName>> = {
  DispatchAssistant: "truck",
  BillingAssistant: "receipt",
  ComplianceAssistant: "shield",
  CustomerAssistant: "headset",
  GeneralAssistant: "bot",
  BillingException: "receipt",
  DispatchAssignment: "route",
  ImportAssistant: "file",
  LoadMonitor: "radar",
  ShipmentIntake: "package",
  CashApplication: "wallet",
};

export function isAgentIconName(value: string | null | undefined): value is AgentIconName {
  return value != null && value in AGENT_ICONS;
}

export function isAgentAccentName(value: string | null | undefined): value is AgentAccentName {
  return value != null && value in AGENT_ACCENTS;
}

export type AgentIdentityInput = {
  id?: string | null;
  name?: string | null;
  icon?: string | null;
  accent?: string | null;
  template?: string | null;
};

export type AgentIdentity = {
  icon: AgentIconName;
  accent: AgentAccentName;
  /**
   * Whether the icon is the agent's own — chosen, or implied by its starter —
   * rather than the generic fallback. A mark with nothing behind it is better
   * drawn as the agent's initials than as the same robot every other
   * unconfigured agent gets.
   */
  iconChosen: boolean;
  /**
   * Which of the sigil variants this agent wears. See agentSigil.
   */
  sigil: AgentSigil;
};

/** One arc around the mark: where it starts and how far it runs. */
export type AgentSigil = {
  /** Degrees clockwise from the top. */
  rotation: number;
  /** Arc length as a percentage of the circumference. */
  length: number;
};

const SIGIL_ROTATIONS = [0, 45, 90, 135, 180, 225, 270, 315];
const SIGIL_LENGTHS = [14, 22, 30];

/**
 * A short arc around the agent's mark, derived from its id.
 *
 * Sixteen icons and eight accents sounds like plenty until you notice that
 * both are assigned by hashing when an organization does not choose, and that
 * an agent with no starter gets the same robot as every other one. Two desks
 * then wear the identical mark, which is the one thing a mark may not do.
 *
 * The arc adds twenty-four variants that cost nothing to read: it is a line,
 * at one weight, in the accent already there, and a person does not have to
 * decode it — they only have to notice that two marks are not the same. It is
 * derived from the id rather than the name, so renaming an agent leaves its
 * face alone.
 */
export function agentSigil(seed: string): AgentSigil {
  const value = hash(seed);

  return {
    rotation: SIGIL_ROTATIONS[value % SIGIL_ROTATIONS.length],
    length: SIGIL_LENGTHS[Math.floor(value / SIGIL_ROTATIONS.length) % SIGIL_LENGTHS.length],
  };
}

/**
 * The letters an agent's mark falls back to: the initials of the first two
 * words of its name, or the first two letters of a single-word name.
 */
export function agentMonogram(name: string | null | undefined): string {
  const words = (name ?? "")
    .trim()
    .split(/\s+/)
    .filter((word) => word !== "");
  if (words.length === 0) {
    return "";
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }

  return (words[0][0] + words[1][0]).toUpperCase();
}

/**
 * What an agent looks like: its own choice first, then what its starter implies,
 * then a stable derivation.
 *
 * The derivation hashes the id rather than the name so renaming an agent never
 * changes its face. An agent still being typed into the builder has no id yet,
 * so it falls back to the name and the preview moves as you type, which is the
 * honest thing for a value that is not saved.
 */
export function resolveAgentIdentity(agent: AgentIdentityInput): AgentIdentity {
  const templateIcon = TEMPLATE_ICON[agent.template ?? ""];
  const chosen = isAgentIconName(agent.icon) ? agent.icon : templateIcon;
  const icon: AgentIconName = chosen ?? "bot";
  const seed = (agent.id ?? "").trim() || (agent.name ?? "").trim();
  const sigil = agentSigil(seed);

  if (isAgentAccentName(agent.accent)) {
    return { icon, accent: agent.accent, iconChosen: chosen !== undefined, sigil };
  }

  return {
    icon,
    accent: AGENT_ACCENT_ORDER[hash(seed) % AGENT_ACCENT_ORDER.length],
    iconChosen: chosen !== undefined,
    sigil,
  };
}

/** FNV-1a, matching the Go side so a face never changes across the wire. */
function hash(value: string): number {
  let result = 0x811c9dc5;
  for (let index = 0; index < value.length; index++) {
    result ^= value.charCodeAt(index);
    result = Math.imul(result, 0x01000193) >>> 0;
  }

  return result >>> 0;
}
