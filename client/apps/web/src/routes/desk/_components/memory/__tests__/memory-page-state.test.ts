import type { DeskMemory } from "@/lib/graphql/desk-memories";
import { describe, expect, it } from "vitest";
import {
  ALL_MEMORIES,
  filterScope,
  initialMemoryPageState,
  inFilter,
  memoryChips,
  memoryPageReducer,
  memoryRows,
  sameFilter,
  type MemoryFilter,
} from "../memory-page-state";

function memory(overrides: Partial<DeskMemory> & { id: string }): DeskMemory {
  return {
    __typename: "DeskMemory",
    content: "Acme is billed net-45.",
    scope: "User",
    roleId: null,
    roleName: "",
    status: "Active",
    source: "User",
    sourceTitle: "",
    useCount: 0,
    lastUsedAt: null,
    createdAt: 100,
    version: 1,
    editable: true,
    ...overrides,
  } as DeskMemory;
}

const billing: MemoryFilter = { kind: "role", roleId: "rol_billing" };

describe("memory filters", () => {
  it("asks the server for one scope, and for a role by its id", () => {
    expect(filterScope(ALL_MEMORIES)).toEqual({ scope: null, roleId: null });
    expect(filterScope({ kind: "scope", scope: "User" })).toEqual({ scope: "User", roleId: null });
    expect(filterScope(billing)).toEqual({ scope: "Role", roleId: "rol_billing" });
  });

  it("places a memory under its own scope and role only", () => {
    const team = memory({ id: "m1", scope: "Role", roleId: "rol_billing" });
    expect(inFilter(team, billing)).toBe(true);
    expect(inFilter(team, { kind: "role", roleId: "rol_dispatch" })).toBe(false);
    expect(inFilter(team, { kind: "scope", scope: "User" })).toBe(false);
    expect(inFilter(team, ALL_MEMORIES)).toBe(true);
    expect(sameFilter(billing, { kind: "role", roleId: "rol_billing" })).toBe(true);
    expect(sameFilter(billing, ALL_MEMORIES)).toBe(false);
  });

  it("gives every role the person holds a chip, counted from the server", () => {
    const chips = memoryChips(
      [
        { scope: "User", roleId: null, count: 2 },
        { scope: "Role", roleId: "rol_billing", count: 3 },
        { scope: "Organization", roleId: null, count: 4 },
      ],
      9,
      [
        { id: "rol_billing", name: "Billing" },
        { id: "rol_dispatch", name: "Dispatch" },
      ],
      { all: "All", user: "Just you", organization: "Organization" },
    );

    expect(chips.map((chip) => [chip.label, chip.count])).toEqual([
      ["All", 9],
      ["Just you", 2],
      ["Billing", 3],
      ["Dispatch", 0],
      ["Organization", 4],
    ]);
  });
});

describe("memory page state", () => {
  it("edits one row at a time and lets go of it once saved", () => {
    const first = memory({ id: "m1", content: "First." });
    let state = memoryPageReducer(initialMemoryPageState, { type: "edit", memory: first });
    expect(state.editing).toEqual({ id: "m1", draft: "First." });

    state = memoryPageReducer(state, { type: "draft", text: "First, edited." });
    expect(state.editing?.draft).toBe("First, edited.");

    expect(memoryPageReducer(state, { type: "edited", id: "m2" }).editing).not.toBeNull();
    expect(memoryPageReducer(state, { type: "edited", id: "m1" }).editing).toBeNull();
    expect(memoryPageReducer(state, { type: "cancel-edit" }).editing).toBeNull();
  });

  it("keeps a forgotten memory in place with its Undo until it is brought back", () => {
    const newer = memory({ id: "m2", createdAt: 200, content: "Newer." });
    const older = memory({ id: "m1", createdAt: 100, content: "Older." });
    const forgotten = { ...newer, status: "Retired" as const };

    let state = memoryPageReducer(initialMemoryPageState, { type: "edit", memory: newer });
    state = memoryPageReducer(state, { type: "forgot", memory: forgotten });
    expect(state.editing).toBeNull();

    // The server no longer lists it; the row stays where it stood.
    const rows = memoryRows([older], state, ALL_MEMORIES, "");
    expect(rows.map((row) => [row.memory.id, row.gone])).toEqual([
      ["m2", true],
      ["m1", false],
    ]);

    state = memoryPageReducer(state, { type: "restored", id: "m2" });
    expect(memoryRows([newer, older], state, ALL_MEMORIES, "").every((row) => !row.gone)).toBe(
      true,
    );
  });

  it("leaves a forgotten memory out of a filter or search it does not match", () => {
    const team = memory({ id: "m1", scope: "Role", roleId: "rol_billing", content: "Net-45." });
    const state = memoryPageReducer(initialMemoryPageState, {
      type: "forgot",
      memory: { ...team, status: "Retired" },
    });

    expect(memoryRows([], state, { kind: "scope", scope: "User" }, "")).toEqual([]);
    expect(memoryRows([], state, billing, "lumper")).toEqual([]);
    expect(memoryRows([], state, billing, "net")).toHaveLength(1);
  });

  it("marks a memory added in this visit as fresh, once", () => {
    let state = memoryPageReducer(initialMemoryPageState, { type: "added", id: "m3" });
    state = memoryPageReducer(state, { type: "added", id: "m3" });
    expect(state.fresh).toEqual(["m3"]);

    const rows = memoryRows([memory({ id: "m3" }), memory({ id: "m1" })], state, ALL_MEMORIES, "");
    expect(rows.find((row) => row.memory.id === "m3")?.fresh).toBe(true);
    expect(rows.find((row) => row.memory.id === "m1")?.fresh).toBe(false);
  });
});
