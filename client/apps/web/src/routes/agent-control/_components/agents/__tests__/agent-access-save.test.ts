import type { AgentAccess } from "@/lib/graphql/agent-access";
import { saveAgentDefinitionRequestSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { accessToSave, NEW_AGENT_ACCESS, sameAccess } from "../agent-access-save";
import { accessOf, agentFormDefaults, toSaveRequest } from "../agent-form-schema";

const DISPATCH: AgentAccess = { mode: "Roles", roleIds: ["role_dispatch"] };

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

/**
 * Who may use the agent rides with its save, in one request the server applies
 * in one transaction. It is sent only when it changed: changing it needs
 * permission to update roles, and re-sending it unchanged would put back roles
 * someone else changed since the form was loaded.
 */
describe("accessToSave", () => {
  it("sends access that changed", () => {
    expect(accessToSave(DISPATCH, NEW_AGENT_ACCESS)).toEqual(DISPATCH);
    expect(accessToSave({ mode: "Everyone", roleIds: ["role_dispatch"] }, DISPATCH)).toEqual({
      mode: "Everyone",
      roleIds: ["role_dispatch"],
    });
  });

  it("leaves out access that did not", () => {
    expect(accessToSave(DISPATCH, { mode: "Roles", roleIds: ["role_dispatch"] })).toBeUndefined();
    expect(accessToSave(NEW_AGENT_ACCESS, NEW_AGENT_ACCESS)).toBeUndefined();
  });
});

/**
 * A new agent meant for some roles used to be created disabled, restricted,
 * then enabled, three requests with a failure between any two. It is now one
 * create that carries its access, enabled as asked.
 */
describe("creating an agent with who may use it", () => {
  it("creates a restricted agent enabled, in one request that carries its roles", () => {
    const values = {
      ...agentFormDefaults,
      name: "Payroll helper",
      enabled: true,
      accessMode: "Roles" as const,
      accessRoleIds: ["role_payroll", "role_payroll", "role_finance"],
    };

    const request = toSaveRequest(values, accessToSave(accessOf(values), NEW_AGENT_ACCESS));

    expect(request.enabled).toBe(true);
    expect(request.accessMode).toBe("Roles");
    expect(request.accessRoleIds).toEqual(["role_payroll", "role_finance"]);
    expect(saveAgentDefinitionRequestSchema.parse(request)).toMatchObject({
      accessMode: "Roles",
      accessRoleIds: ["role_payroll", "role_finance"],
    });
  });

  it("creates an agent open to everyone without saying who may use it", () => {
    const values = { ...agentFormDefaults, name: "Dispatch" };

    const request = toSaveRequest(values, accessToSave(accessOf(values), NEW_AGENT_ACCESS));

    expect(request).not.toHaveProperty("accessMode");
    expect(request).not.toHaveProperty("accessRoleIds");
  });

  // Roles chosen while open to everyone are kept for when it is restricted
  // again, so they are access too, and are sent.
  it("sends roles chosen for an agent open to everyone", () => {
    const values = {
      ...agentFormDefaults,
      name: "Dispatch",
      accessMode: "Everyone" as const,
      accessRoleIds: ["role_dispatch"],
    };

    const request = toSaveRequest(values, accessToSave(accessOf(values), NEW_AGENT_ACCESS));

    expect(request.accessMode).toBe("Everyone");
    expect(request.accessRoleIds).toEqual(["role_dispatch"]);
  });
});
