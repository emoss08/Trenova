import {
  AgentCapabilitiesDocument,
  AgentCapabilitiesFieldsFragmentDoc,
  AgentCapabilityToolFieldsFragmentDoc,
  UpdateAgentCapabilitiesDocument,
  type AgentCapabilitiesFieldsFragment,
  type AgentCapabilityMode,
  type AgentCapabilityToolFieldsFragment,
  type UpdateAgentCapabilitiesInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type { AgentCapabilityMode, UpdateAgentCapabilitiesInput };

export type AgentCapabilityTool = Omit<AgentCapabilityToolFieldsFragment, " $fragmentName">;

/**
 * What an agent can do, as its capabilities page reads it: the tools it looks
 * things up with and changes things with, who it hands work to and its limits.
 * `canEdit` says whether the reader may change any of it.
 */
export type AgentCapabilities = Omit<
  AgentCapabilitiesFieldsFragment,
  " $fragmentName" | "readTools" | "writeTools"
> & {
  readTools: AgentCapabilityTool[];
  writeTools: AgentCapabilityTool[];
};

type RequestOptions = { signal?: AbortSignal };

function unmask(
  fragment: FragmentType<typeof AgentCapabilitiesFieldsFragmentDoc>,
): AgentCapabilities {
  const caps = getFragmentData(AgentCapabilitiesFieldsFragmentDoc, fragment);
  const tool = (row: (typeof caps.readTools)[number]): AgentCapabilityTool => ({
    ...getFragmentData(AgentCapabilityToolFieldsFragmentDoc, row),
  });

  return {
    ...caps,
    readTools: caps.readTools.map(tool),
    writeTools: caps.writeTools.map(tool),
  };
}

export async function fetchAgentCapabilities(
  agentId: string,
  options?: RequestOptions,
): Promise<AgentCapabilities> {
  const data = await requestGraphQL({
    document: AgentCapabilitiesDocument,
    operationName: "AgentCapabilities",
    variables: { agentId },
    signal: options?.signal,
  });

  return unmask(data.agentCapabilities);
}

export async function updateAgentCapabilities(
  agentId: string,
  input: UpdateAgentCapabilitiesInput,
): Promise<AgentCapabilities> {
  const data = await requestGraphQL({
    document: UpdateAgentCapabilitiesDocument,
    operationName: "UpdateAgentCapabilities",
    variables: { agentId, input },
  });

  return unmask(data.updateAgentCapabilities);
}

/** The modes a row offers, in the page's order. A read never asks first. */
export const WRITE_MODES: readonly AgentCapabilityMode[] = ["Allowed", "AskFirst", "Off"];
export const READ_MODES: readonly AgentCapabilityMode[] = ["Allowed", "Off"];

export type CapabilityOption = {
  mode: AgentCapabilityMode;
  selected: boolean;
  /** The reader cannot pick it: they may only read, or the tool's lock rules it out. */
  disabled: boolean;
};

/**
 * The segmented control for one row. Every mode the row's kind has is drawn,
 * so a locked row still shows where it stands; the ones the server does not
 * offer, or every one for a reader who may not edit, are disabled.
 */
export function capabilityOptions(tool: AgentCapabilityTool, canEdit: boolean): CapabilityOption[] {
  const modes = tool.write ? WRITE_MODES : READ_MODES;

  return modes.map((mode) => ({
    mode,
    selected: tool.mode === mode,
    disabled: !canEdit || !tool.allowedModes.includes(mode),
  }));
}

/** A row is locked when it carries a reason some mode is ruled out. */
export function isLocked(tool: AgentCapabilityTool): boolean {
  return Boolean(tool.lockReason);
}

/**
 * The page as it will read once a mode change lands, so the control moves on
 * the click rather than after the round trip.
 */
export function withToolMode(
  caps: AgentCapabilities,
  key: string,
  mode: AgentCapabilityMode,
): AgentCapabilities {
  const set = (tools: AgentCapabilityTool[]) =>
    tools.map((tool) => (tool.key === key ? { ...tool, mode } : tool));

  return { ...caps, readTools: set(caps.readTools), writeTools: set(caps.writeTools) };
}

/** Past this share of a limit its bar turns amber. */
export const LIMIT_HIGH_SHARE = 0.85;

/** How much of a limit is used, between 0 and 1; 0 when there is no limit. */
export function limitShare(used: number, limit: number): number {
  if (!(limit > 0) || !(used > 0)) {
    return 0;
  }

  return Math.min(1, used / limit);
}

export function limitHigh(used: number, limit: number): boolean {
  return limit > 0 && used / limit > LIMIT_HIGH_SHARE;
}

/** A minute after midnight as a clock time: 420 is "7 AM", 750 is "12:30 PM". */
export function clockLabel(minute: number): string {
  const hour = Math.floor(minute / 60) % 24;
  const mins = minute % 60;
  const suffix = hour >= 12 ? "PM" : "AM";
  const display = hour % 12 === 0 ? 12 : hour % 12;

  return mins === 0
    ? `${display} ${suffix}`
    : `${display}:${String(mins).padStart(2, "0")} ${suffix}`;
}

/**
 * A zone as a person names it: "America/Chicago" is "Central". Falls back to
 * the zone's own name where the browser cannot name it.
 */
export function zoneLabel(timeZone: string, at = new Date()): string {
  try {
    const part = new Intl.DateTimeFormat("en-US", { timeZone, timeZoneName: "longGeneric" })
      .formatToParts(at)
      .find((piece) => piece.type === "timeZoneName")?.value;
    if (part) {
      return part.replace(/ Time$/u, "");
    }
  } catch {
    // An unknown zone reads as its name.
  }

  return timeZone;
}

/** "7 AM – 6 PM Central" */
export function businessHoursLabel(start: number, end: number, timeZone: string): string {
  return `${clockLabel(start)} – ${clockLabel(end)} ${zoneLabel(timeZone)}`;
}
