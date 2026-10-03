import type { DeskMemory, DeskMemoryCount, DeskMemoryScope } from "@/lib/graphql/desk-memories";

/**
 * Which memories the page shows: every one, or one scope's. A role is chosen
 * by its id, since a person can hold more than one.
 */
export type MemoryFilter =
  | { kind: "all" }
  | { kind: "scope"; scope: Exclude<DeskMemoryScope, "Role"> }
  | { kind: "role"; roleId: string };

export const ALL_MEMORIES: MemoryFilter = { kind: "all" };

export function sameFilter(a: MemoryFilter, b: MemoryFilter): boolean {
  if (a.kind !== b.kind) {
    return false;
  }
  if (a.kind === "scope" && b.kind === "scope") {
    return a.scope === b.scope;
  }
  if (a.kind === "role" && b.kind === "role") {
    return a.roleId === b.roleId;
  }
  return true;
}

/** The filter as the server reads it. */
export function filterScope(filter: MemoryFilter): {
  scope: DeskMemoryScope | null;
  roleId: string | null;
} {
  switch (filter.kind) {
    case "scope":
      return { scope: filter.scope, roleId: null };
    case "role":
      return { scope: "Role", roleId: filter.roleId };
    default:
      return { scope: null, roleId: null };
  }
}

/** Whether a memory belongs under the filter, for rows the page holds on to itself. */
export function inFilter(memory: Pick<DeskMemory, "scope" | "roleId">, filter: MemoryFilter) {
  switch (filter.kind) {
    case "scope":
      return memory.scope === filter.scope;
    case "role":
      return memory.scope === "Role" && memory.roleId === filter.roleId;
    default:
      return true;
  }
}

/** One filter chip: its filter and how many memories it would show. */
export type MemoryChip = { key: string; filter: MemoryFilter; label: string; count: number };

/**
 * The chips across the list: All, Just you, one for each of the person's
 * roles (by the role's name), and Organization, each with the count the
 * server gave for it. A role with nothing kept for it still has its chip, so
 * a team's place is there before its first memory.
 */
export function memoryChips(
  counts: readonly DeskMemoryCount[],
  all: number,
  roles: readonly { id: string; name: string }[],
  labels: { all: string; user: string; organization: string },
): MemoryChip[] {
  const countOf = (scope: DeskMemoryScope, roleId: string | null = null) =>
    counts
      .filter((count) => count.scope === scope && (roleId === null || count.roleId === roleId))
      .reduce((sum, count) => sum + count.count, 0);

  return [
    { key: "all", filter: ALL_MEMORIES, label: labels.all, count: all },
    {
      key: "user",
      filter: { kind: "scope", scope: "User" },
      label: labels.user,
      count: countOf("User"),
    },
    ...roles.map((role) => ({
      key: `role:${role.id}`,
      filter: { kind: "role", roleId: role.id } as const,
      label: role.name,
      count: countOf("Role", role.id),
    })),
    {
      key: "organization",
      filter: { kind: "scope", scope: "Organization" },
      label: labels.organization,
      count: countOf("Organization"),
    },
  ];
}

/**
 * What the page holds beside what the server listed: the row being edited and
 * its draft, the memories forgotten in this visit, which stay in place as a
 * "Forgotten · Undo" line until the person leaves, and the memories added in
 * this visit, which arrive with a highlight.
 */
export type MemoryPageState = {
  editing: { id: string; draft: string } | null;
  forgotten: Record<string, DeskMemory>;
  fresh: readonly string[];
};

export const initialMemoryPageState: MemoryPageState = {
  editing: null,
  forgotten: {},
  fresh: [],
};

export type MemoryPageAction =
  | { type: "edit"; memory: DeskMemory }
  | { type: "draft"; text: string }
  | { type: "cancel-edit" }
  | { type: "edited"; id: string }
  | { type: "forgot"; memory: DeskMemory }
  | { type: "restored"; id: string }
  | { type: "added"; id: string };

export function memoryPageReducer(
  state: MemoryPageState,
  action: MemoryPageAction,
): MemoryPageState {
  switch (action.type) {
    case "edit":
      return { ...state, editing: { id: action.memory.id, draft: action.memory.content } };
    case "draft":
      return state.editing
        ? { ...state, editing: { ...state.editing, draft: action.text } }
        : state;
    case "cancel-edit":
      return { ...state, editing: null };
    case "edited":
      return state.editing?.id === action.id ? { ...state, editing: null } : state;
    case "forgot": {
      const editing = state.editing?.id === action.memory.id ? null : state.editing;
      return {
        ...state,
        editing,
        forgotten: { ...state.forgotten, [action.memory.id]: action.memory },
      };
    }
    case "restored": {
      if (!(action.id in state.forgotten)) {
        return state;
      }
      const { [action.id]: _restored, ...forgotten } = state.forgotten;
      return { ...state, forgotten };
    }
    case "added":
      return state.fresh.includes(action.id)
        ? state
        : { ...state, fresh: [action.id, ...state.fresh] };
    default:
      return state;
  }
}

/** A row of the list: a memory, or the line a forgotten one leaves behind. */
export type MemoryRow = { memory: DeskMemory; gone: boolean; fresh: boolean };

function newestFirst(a: DeskMemory, b: DeskMemory): number {
  if (a.createdAt !== b.createdAt) {
    return b.createdAt - a.createdAt;
  }
  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
}

/**
 * The rows the list shows: what the server listed, with each memory forgotten
 * in this visit put back where it stood as a forgotten line. The server no
 * longer lists a forgotten memory, so without this its Undo would vanish with
 * the next refresh. Forgotten ones outside the filter or the search stay out.
 */
export function memoryRows(
  listed: readonly DeskMemory[],
  state: MemoryPageState,
  filter: MemoryFilter,
  query: string,
): MemoryRow[] {
  const needle = query.trim().toLowerCase();
  const byId = new Map<string, DeskMemory>();
  for (const memory of listed) {
    byId.set(memory.id, memory);
  }
  for (const memory of Object.values(state.forgotten)) {
    if (!inFilter(memory, filter)) {
      continue;
    }
    if (needle !== "" && !memory.content.toLowerCase().includes(needle)) {
      continue;
    }
    byId.set(memory.id, memory);
  }

  return [...byId.values()].sort(newestFirst).map((memory) => ({
    memory,
    gone: memory.id in state.forgotten,
    fresh: state.fresh.includes(memory.id),
  }));
}
