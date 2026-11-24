import type { AgentScopeInput } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { ToolNamespace } from "@trenova/shared/types/generated/tool-namespaces";
import { resourceLabel } from "./resource-label";

/**
 * What an agent may do, as the server's `agentdefinition.Scope` holds it. An area allows every
 * capability in it; a capability is a tool name or an entity read (`invoice.list`). Denials win.
 * Areas and capabilities both empty is everything the person may do: the universal agent.
 */
export type AgentScope = {
  areas: string[];
  capabilities: string[];
  denyAreas: string[];
  denyCapabilities: string[];
};

/** A scope as GraphQL returns it, with lists that may be null. */
export type AgentScopeLike = {
  areas?: readonly string[] | null;
  capabilities?: readonly string[] | null;
  denyAreas?: readonly string[] | null;
  denyCapabilities?: readonly string[] | null;
};

/** The most capabilities an agent loads on every turn, as the server allows. */
export const MAX_PINNED_TOOLS = 12;

/** The area that stands for all of them: `denyAreas: ["*"]` switches an agent off. */
export const ALL_AREAS = "*";

/** Everything the person may do. */
export const UNIVERSAL_SCOPE: AgentScope = Object.freeze({
  areas: [],
  capabilities: [],
  denyAreas: [],
  denyCapabilities: [],
}) as AgentScope;

export function toAgentScope(scope: AgentScopeLike | null | undefined): AgentScope {
  return {
    areas: [...(scope?.areas ?? [])],
    capabilities: [...(scope?.capabilities ?? [])],
    denyAreas: [...(scope?.denyAreas ?? [])],
    denyCapabilities: [...(scope?.denyCapabilities ?? [])],
  };
}

/** The scope as the server takes it: every list sent, so the same scope is the same request. */
export function toAgentScopeInput(scope: AgentScope): Required<AgentScopeInput> {
  return {
    areas: [...new Set(scope.areas)].sort(),
    capabilities: [...new Set(scope.capabilities)].sort(),
    denyAreas: [...new Set(scope.denyAreas)].sort(),
    denyCapabilities: [...new Set(scope.denyCapabilities)].sort(),
  };
}

/** Whether the scope reaches everything the person may do, before its denials. */
export function coversEverything(scope: AgentScopeLike): boolean {
  return (scope.areas?.length ?? 0) === 0 && (scope.capabilities?.length ?? 0) === 0;
}

/**
 * The scope a form holds as it reads: one being narrowed that names nothing yet holds nothing,
 * rather than everything its empty lists would otherwise stand for.
 */
export function formScope(scope: AgentScope, limited: boolean): AgentScope {
  return limited && coversEverything(scope)
    ? { ...scope, denyAreas: [...scope.denyAreas, ALL_AREAS] }
    : scope;
}

/** Whether every area is denied, which is how an agent is switched off. */
export function isSwitchedOff(scope: AgentScopeLike): boolean {
  return (scope.denyAreas ?? []).includes(ALL_AREAS);
}

/**
 * The area a tool belongs to, as the server's `agentcapability.AreaOf` reads it: the word before
 * its first underscore, with the agent's own verbs under "agent" and the generic reads under
 * "records".
 */
export function toolArea(name: string): string {
  const cut = name.indexOf("_");
  if (cut < 0) {
    return "agent";
  }
  const area = name.slice(0, cut);
  switch (area) {
    case "find":
    case "search":
    case "ask":
    case "show":
    case "publish":
    case "request":
    case "withdraw":
    case "delegate":
    case "name":
      return "agent";
    case "records":
    case "record":
    case "describe":
      return "records";
    default:
      return area;
  }
}

/** Whether a capability is an entity read (`invoice.list`) rather than a tool name. */
export function isEntityCapability(capability: string): boolean {
  return /^[a-z_]+\.(list|get)$/.test(capability);
}

/**
 * Whether the scope allows a tool, as the server's `Scope.Allows` decides it. Tools every agent
 * holds are the caller's to add; this reads only the scope.
 */
export function scopeAllows(scope: AgentScopeLike, name: string): boolean {
  if (name === "") {
    return false;
  }
  const area = toolArea(name);
  const denyAreas = scope.denyAreas ?? [];
  if ((scope.denyCapabilities ?? []).includes(name) || denyAreas.includes(area)) {
    return false;
  }
  if (denyAreas.includes(ALL_AREAS)) {
    return false;
  }
  if (coversEverything(scope)) {
    return true;
  }

  return (scope.areas ?? []).includes(area) || (scope.capabilities ?? []).includes(name);
}

/**
 * The scope with a tool allowed, as the server's `Scope.Allow` does it: a denial of it is lifted,
 * and it is named among the capabilities only when nothing else allows it. A `limited` scope is
 * one being narrowed that names nothing yet, so its empty lists do not stand for everything.
 */
export function allowTool(scope: AgentScope, name: string, limited = false): AgentScope {
  const next = {
    ...scope,
    denyCapabilities: scope.denyCapabilities.filter((entry) => entry !== name),
  };
  if (scopeAllows(next, name) && !(limited && coversEverything(next))) {
    return next;
  }

  return { ...next, capabilities: [...next.capabilities, name] };
}

/**
 * The scope with a tool taken away, as the server's `Scope.Deny` does it: it leaves the
 * capabilities, and while an area or everything still allows it, it is switched off by name.
 */
export function denyTool(scope: AgentScope, name: string, limited = false): AgentScope {
  const next = { ...scope, capabilities: scope.capabilities.filter((entry) => entry !== name) };
  if (!scopeAllows(next, name) || (limited && coversEverything(next))) {
    return next;
  }

  return { ...next, denyCapabilities: [...next.denyCapabilities, name] };
}

/** The scope with every area allowed by name, dropping a denial of each. */
export function withAreas(scope: AgentScope, areas: readonly string[]): AgentScope {
  const chosen = [...new Set(areas)];
  return {
    ...scope,
    areas: chosen,
    denyAreas: scope.denyAreas.filter((area) => !chosen.includes(area)),
  };
}

/** The words a scope says about what an agent is for, for matching it to work. */
export function scopeWords(scope: AgentScopeLike): string[] {
  return [...(scope.areas ?? []), ...(scope.capabilities ?? [])];
}

/** Each area of Trenova in words, as the builder offers it. */
const AREA_LABEL: Record<ToolNamespace | "records", string> = defineLabels({
  accounting: "Accounting",
  agent: "The agent itself",
  analysis: "Analysis",
  billing: "Billing",
  carrier: "Carriers",
  customer: "Customers",
  detention: "Detention",
  dispatch: "Dispatch",
  document: "Documents",
  edi: "EDI",
  equipment: "Equipment",
  exception: "Exceptions",
  factoring: "Factoring",
  formula: "Formulas",
  fuel: "Fuel",
  home: "Home page",
  inbox: "Inbox",
  insight: "Insights",
  intake: "Intake",
  invoice: "Invoices",
  ledger: "Ledger",
  location: "Locations",
  masterdata: "Master data",
  order: "Orders",
  rate: "Rates",
  receivable: "Receivables",
  records: "Any record",
  report: "Reports",
  settlement: "Settlements",
  shipment: "Shipments",
  web: "The web",
  worker: "Workers",
});

/** An area in words; one the client does not know reads as its resource would. */
export function areaLabel(area: string): string {
  return (AREA_LABEL as Record<string, string>)[area] ?? resourceLabel(area);
}
