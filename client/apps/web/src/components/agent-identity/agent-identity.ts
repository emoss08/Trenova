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
  SparklesIcon,
  TruckIcon,
  WalletCardsIcon,
} from "lucide-react";

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
  sparkle: SparklesIcon,
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
};

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
  const icon: AgentIconName = isAgentIconName(agent.icon)
    ? agent.icon
    : (TEMPLATE_ICON[agent.template ?? ""] ?? "bot");

  if (isAgentAccentName(agent.accent)) {
    return { icon, accent: agent.accent };
  }

  const seed = (agent.id ?? "").trim() || (agent.name ?? "").trim();

  return { icon, accent: AGENT_ACCENT_ORDER[hash(seed) % AGENT_ACCENT_ORDER.length] };
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
