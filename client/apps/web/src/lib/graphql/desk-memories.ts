import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  ConfirmDeskMemoryDocument,
  CreateDeskMemoryDocument,
  DeskMemoriesByIdsDocument,
  DeskMemoriesDocument,
  DeskMemoryFieldsFragmentDoc,
  DeskMemorySettingsDocument,
  DeskMemorySettingsFieldsFragmentDoc,
  DismissDeskMemoryDocument,
  ReviseDeskMemoryDocument,
  SetDeskMemoryStatusDocument,
  SetMemorySavingModeDocument,
  type AgentMemorySavingMode,
  type AgentMemoryScope,
  type AgentMemoryStatus,
  type DeskMemoryFieldsFragment,
  type DeskMemorySettingsFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

/** Every Desk memory read lives under this key, so one invalidation refreshes them all. */
export const DESK_MEMORIES_KEY = "desk-memories";

export type DeskMemory = DeskMemoryFieldsFragment;
export type DeskMemorySettings = DeskMemorySettingsFieldsFragment;
export type DeskMemoryScope = Extract<AgentMemoryScope, "User" | "Role" | "Organization">;
export type { AgentMemorySavingMode };

/**
 * Who a memory is kept for, as the page names it: one scope, and for Role one
 * role. Agent is a lesson an agent learned in the person's conversation for
 * everyone who uses it; it stays where it is when they save or edit it.
 */
export type DeskMemoryAudience = { scope: DeskMemoryScope | "Agent"; roleId: string | null };

export type DeskMemoryCount = { scope: AgentMemoryScope; roleId: string | null; count: number };

export type DeskMemoryPage = {
  items: DeskMemory[];
  next: string | null;
  all: number;
  counts: DeskMemoryCount[];
};

export type DeskMemoryQuery = {
  scope: DeskMemoryScope | null;
  roleId: string | null;
  query: string;
};

export async function fetchDeskMemories(
  filter: DeskMemoryQuery,
  after: string | null,
  first: number,
  signal?: AbortSignal,
): Promise<DeskMemoryPage> {
  const data = await requestGraphQL({
    document: DeskMemoriesDocument,
    operationName: "DeskMemories",
    variables: {
      input: {
        first,
        after,
        scope: filter.scope,
        roleId: filter.scope === "Role" ? filter.roleId : null,
        query: filter.query.trim() || null,
      },
    },
    signal,
  });
  const page = data.deskMemories;

  return {
    items: page.items.map((item) => getFragmentData(DeskMemoryFieldsFragmentDoc, item)),
    next: page.next ?? null,
    all: page.all,
    counts: page.counts.map((count) => ({ ...count, roleId: count.roleId ?? null })),
  };
}

export async function fetchDeskMemoriesByIds(
  ids: readonly string[],
  signal?: AbortSignal,
): Promise<DeskMemory[]> {
  const data = await requestGraphQL({
    document: DeskMemoriesByIdsDocument,
    operationName: "DeskMemoriesByIds",
    variables: { ids: [...ids] },
    signal,
  });

  return data.deskMemoriesByIds.map((item) => getFragmentData(DeskMemoryFieldsFragmentDoc, item));
}

export async function fetchDeskMemorySettings(signal?: AbortSignal): Promise<DeskMemorySettings> {
  const data = await requestGraphQL({
    document: DeskMemorySettingsDocument,
    operationName: "DeskMemorySettings",
    variables: {},
    signal,
  });

  return getFragmentData(DeskMemorySettingsFieldsFragmentDoc, data.deskMemorySettings);
}

export async function createDeskMemory(
  content: string,
  audience: DeskMemoryAudience,
): Promise<DeskMemory> {
  const data = await requestGraphQL({
    document: CreateDeskMemoryDocument,
    operationName: "CreateDeskMemory",
    variables: { input: { content, scope: audience.scope, roleId: audience.roleId } },
  });

  return getFragmentData(DeskMemoryFieldsFragmentDoc, data.createDeskMemory);
}

/** Changes what a memory says, who it is kept for, or both; what is left out stays. */
export async function reviseDeskMemory(
  id: string,
  change: { content?: string; audience?: DeskMemoryAudience; version: number },
): Promise<DeskMemory> {
  const data = await requestGraphQL({
    document: ReviseDeskMemoryDocument,
    operationName: "ReviseDeskMemory",
    variables: {
      id,
      input: {
        content: change.content ?? null,
        scope: change.audience?.scope ?? null,
        roleId: change.audience?.roleId ?? null,
        version: change.version,
      },
    },
  });

  return getFragmentData(DeskMemoryFieldsFragmentDoc, data.reviseDeskMemory);
}

/** Active resumes or brings a forgotten memory back, Paused sets it aside, Retired forgets it. */
export async function setDeskMemoryStatus(
  id: string,
  status: Extract<AgentMemoryStatus, "Active" | "Paused" | "Retired">,
): Promise<DeskMemory> {
  const data = await requestGraphQL({
    document: SetDeskMemoryStatusDocument,
    operationName: "SetDeskMemoryStatus",
    variables: { id, status },
  });

  return getFragmentData(DeskMemoryFieldsFragmentDoc, data.setDeskMemoryStatus);
}

export async function confirmDeskMemory(
  id: string,
  content: string,
  audience: DeskMemoryAudience,
  version: number,
): Promise<DeskMemory> {
  const data = await requestGraphQL({
    document: ConfirmDeskMemoryDocument,
    operationName: "ConfirmDeskMemory",
    variables: {
      id,
      input: { content, scope: audience.scope, roleId: audience.roleId, version },
    },
  });

  return getFragmentData(DeskMemoryFieldsFragmentDoc, data.confirmDeskMemory);
}

export async function dismissDeskMemory(id: string): Promise<DeskMemory> {
  const data = await requestGraphQL({
    document: DismissDeskMemoryDocument,
    operationName: "DismissDeskMemory",
    variables: { id },
  });

  return getFragmentData(DeskMemoryFieldsFragmentDoc, data.dismissDeskMemory);
}

export async function setMemorySavingMode(
  mode: AgentMemorySavingMode,
): Promise<DeskMemorySettings> {
  const data = await requestGraphQL({
    document: SetMemorySavingModeDocument,
    operationName: "SetMemorySavingMode",
    variables: { mode },
  });

  return getFragmentData(DeskMemorySettingsFieldsFragmentDoc, data.setMemorySavingMode);
}
