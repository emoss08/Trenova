import type { AgentAccess } from "@/lib/graphql/agent-access";
import type { SaveAgentDefinitionRequest } from "@/types/assistant";
import { describe, expect, it, vi } from "vitest";
import { sameAccess, saveEditedAgent, saveNewAgent } from "../agent-access-save";
import { agentFormDefaults, toSaveRequest } from "../agent-form-schema";

const EVERYONE: AgentAccess = { mode: "Everyone", roleIds: [] };
const DISPATCH: AgentAccess = { mode: "Roles", roleIds: ["role_dispatch"] };

type Agent = { id: string; version: number; enabled: boolean };

/** The order the requests went out in, which is the contract under test. */
function recorder() {
  const calls: string[] = [];
  return {
    calls,
    record<T>(name: string, result: T) {
      return vi.fn(async (..._args: unknown[]): Promise<T> => {
        calls.push(name);
        return result;
      });
    },
    fail(name: string, error: Error) {
      return vi.fn(async (..._args: unknown[]): Promise<never> => {
        calls.push(name);
        throw error;
      });
    },
  };
}

function request(overrides: Partial<SaveAgentDefinitionRequest> = {}): SaveAgentDefinitionRequest {
  return { ...toSaveRequest({ ...agentFormDefaults, name: "Payroll helper" }), ...overrides };
}

describe("sameAccess", () => {
  it("compares the mode and the set of roles, not their order", () => {
    expect(
      sameAccess({ mode: "Roles", roleIds: ["a", "b"] }, { mode: "Roles", roleIds: ["b", "a"] }),
    ).toBe(true);
    expect(
      sameAccess({ mode: "Roles", roleIds: ["a"] }, { mode: "Everyone", roleIds: ["a"] }),
    ).toBe(false);
    expect(
      sameAccess({ mode: "Roles", roleIds: ["a"] }, { mode: "Roles", roleIds: ["a", "b"] }),
    ).toBe(false);
    expect(
      sameAccess({ mode: "Roles", roleIds: ["a", "a"] }, { mode: "Roles", roleIds: ["a"] }),
    ).toBe(true);
  });
});

describe("saveEditedAgent", () => {
  const agent: Agent = { id: "agdef_1", version: 4, enabled: true };

  it("sends access before the agent, and only when it changed", async () => {
    const { calls, record } = recorder();
    const setAccess = record("access", undefined);
    const saveAgent = record("agent", agent);
    const onAccessSaved = vi.fn();

    const result = await saveEditedAgent({
      access: DISPATCH,
      savedAccess: EVERYONE,
      setAccess,
      saveAgent,
      onAccessSaved,
    });

    expect(calls).toEqual(["access", "agent"]);
    expect(setAccess).toHaveBeenCalledWith(DISPATCH);
    expect(onAccessSaved).toHaveBeenCalledWith(DISPATCH);
    expect(result).toEqual({ agent, accessChanged: true });
  });

  it("leaves access alone when it is what was saved", async () => {
    const { calls, record } = recorder();

    const result = await saveEditedAgent({
      access: { mode: "Roles", roleIds: ["role_dispatch"] },
      savedAccess: DISPATCH,
      setAccess: record("access", undefined),
      saveAgent: record("agent", agent),
    });

    expect(calls).toEqual(["agent"]);
    expect(result.accessChanged).toBe(false);
  });

  // Access does not move the agent's version, so a refused access leaves
  // nothing saved and the form can be saved again as it stands.
  it("saves nothing else when access is refused", async () => {
    const { calls, record, fail } = recorder();
    const refusal = new Error("Role not found within your organization");
    const onAccessSaved = vi.fn();

    await expect(
      saveEditedAgent({
        access: DISPATCH,
        savedAccess: EVERYONE,
        setAccess: fail("access", refusal),
        saveAgent: record("agent", agent),
        onAccessSaved,
      }),
    ).rejects.toBe(refusal);
    expect(calls).toEqual(["access"]);
    expect(onAccessSaved).not.toHaveBeenCalled();
  });

  // The agent's own error comes back untouched, so its field errors still
  // land on the fields; the caller has already heard that access was saved.
  it("reports access saved, then passes the agent's own failure through untouched", async () => {
    const { calls, record, fail } = recorder();
    const invalid = new Error("Name is required");
    const onAccessSaved = vi.fn();

    await expect(
      saveEditedAgent({
        access: DISPATCH,
        savedAccess: EVERYONE,
        setAccess: record("access", undefined),
        saveAgent: fail("agent", invalid),
        onAccessSaved,
      }),
    ).rejects.toBe(invalid);
    expect(calls).toEqual(["access", "agent"]);
    expect(onAccessSaved).toHaveBeenCalledWith(DISPATCH);
  });
});

describe("saveNewAgent", () => {
  const created: Agent = { id: "agdef_new", version: 1, enabled: false };

  it("only creates an agent open to everyone with no roles", async () => {
    const { calls, record } = recorder();
    const createAgent = record("create", { ...created, enabled: true });

    const result = await saveNewAgent({
      request: request(),
      access: EVERYONE,
      createAgent,
      updateAgent: record("update", created),
      setAccess: record("access", undefined),
    });

    expect(calls).toEqual(["create"]);
    expect(createAgent).toHaveBeenCalledWith(expect.objectContaining({ enabled: true }));
    expect(result.problem).toBeNull();
  });

  // A new agent is born open to everyone. One meant for some roles is created
  // disabled, restricted, and only then enabled, so it is never open and
  // enabled at once.
  it("creates a restricted agent disabled, restricts it, then enables it", async () => {
    const { calls, record } = recorder();
    const enabled: Agent = { ...created, version: 2, enabled: true };
    const createAgent = record("create", created);
    const setAccess = record("access", undefined);
    const updateAgent = record("update", enabled);

    const result = await saveNewAgent({
      request: request({ enabled: true }),
      access: DISPATCH,
      createAgent,
      updateAgent,
      setAccess,
    });

    expect(calls).toEqual(["create", "access", "update"]);
    expect(createAgent).toHaveBeenCalledWith(expect.objectContaining({ enabled: false }));
    expect(setAccess).toHaveBeenCalledWith("agdef_new", DISPATCH);
    expect(updateAgent).toHaveBeenCalledWith(
      "agdef_new",
      expect.objectContaining({ enabled: true, version: 1, name: "Payroll helper" }),
    );
    expect(result).toEqual({ agent: enabled, problem: null });
  });

  it("does not enable a restricted agent that was asked to be created disabled", async () => {
    const { calls, record } = recorder();

    const result = await saveNewAgent({
      request: request({ enabled: false }),
      access: DISPATCH,
      createAgent: record("create", created),
      updateAgent: record("update", created),
      setAccess: record("access", undefined),
    });

    expect(calls).toEqual(["create", "access"]);
    expect(result.problem).toBeNull();
  });

  it("leaves a restricted agent disabled when its access cannot be saved", async () => {
    const { calls, record, fail } = recorder();
    const refusal = new Error("Role not found within your organization");

    const result = await saveNewAgent({
      request: request({ enabled: true }),
      access: DISPATCH,
      createAgent: record("create", created),
      updateAgent: record("update", created),
      setAccess: fail("access", refusal),
    });

    expect(calls).toEqual(["create", "access"]);
    expect(result).toEqual({
      agent: created,
      problem: { kind: "access-not-saved-left-disabled", error: refusal },
    });
  });

  it("says so when access was saved but the agent could not be enabled", async () => {
    const { record, fail } = recorder();
    const conflict = new Error("Version mismatch");

    const result = await saveNewAgent({
      request: request({ enabled: true }),
      access: DISPATCH,
      createAgent: record("create", created),
      updateAgent: fail("update", conflict),
      setAccess: record("access", undefined),
    });

    expect(result).toEqual({ agent: created, problem: { kind: "not-enabled", error: conflict } });
  });

  // Open to everyone, the roles only matter once it is restricted again, so
  // the agent is created as asked and the roles are recorded after it.
  it("records the roles of an agent open to everyone after creating it as asked", async () => {
    const { calls, record, fail } = recorder();
    const kept: AgentAccess = { mode: "Everyone", roleIds: ["role_dispatch"] };
    const createAgent = record("create", { ...created, enabled: true });
    const setAccess = fail("access", new Error("Role not found within your organization"));

    const result = await saveNewAgent({
      request: request({ enabled: true }),
      access: kept,
      createAgent,
      updateAgent: record("update", created),
      setAccess,
    });

    expect(calls).toEqual(["create", "access"]);
    expect(createAgent).toHaveBeenCalledWith(expect.objectContaining({ enabled: true }));
    expect(result.problem?.kind).toBe("access-not-saved");
  });

  it("creates nothing further when the create itself fails", async () => {
    const { calls, record, fail } = recorder();
    const duplicate = new Error("An agent with this name already exists");

    await expect(
      saveNewAgent({
        request: request(),
        access: DISPATCH,
        createAgent: fail("create", duplicate),
        updateAgent: record("update", created),
        setAccess: record("access", undefined),
      }),
    ).rejects.toBe(duplicate);
    expect(calls).toEqual(["create"]);
  });
});
