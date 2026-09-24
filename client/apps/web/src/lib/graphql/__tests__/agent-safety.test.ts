import {
  AgentSafetyDocument,
  AgentSafetySummaryDocument,
  AgentToolRuleTableDocument,
  AgentToolSafetyTableDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  agentToolRuleTableGraphQLConfig,
  createAgentToolSafetyTableGraphQLConfig,
  fetchAgentSafetyHeaders,
  fetchAgentSafetySummary,
} from "../agent-safety";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables?: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("table configs", () => {
  // The data table reads rows from the connection the config names; a name
  // that drifts from the operation reads nothing.
  it("reads the tool rules from the rule connection", () => {
    expect(agentToolRuleTableGraphQLConfig.document).toBe(AgentToolRuleTableDocument);
    expect(agentToolRuleTableGraphQLConfig.operationName).toBe("AgentToolRuleTable");
    expect(agentToolRuleTableGraphQLConfig.connectionKey).toBe("agentToolRuleConnection");
    expect(agentToolRuleTableGraphQLConfig.extraVariables).toBeUndefined();
  });

  // The agents are an argument of the query, not a filter the table owns, so
  // clearing the table's filters never widens it to every agent.
  it("names the compared agents as a variable of their own", () => {
    const agentIds = ["agdef_1", "agdef_2"];
    const config = createAgentToolSafetyTableGraphQLConfig(agentIds);

    expect(config.document).toBe(AgentToolSafetyTableDocument);
    expect(config.connectionKey).toBe("agentToolSafetyConnection");
    expect(config.extraVariables).toEqual({ agentIds: ["agdef_1", "agdef_2"] });

    agentIds.push("agdef_3");
    expect(config.extraVariables).toEqual({ agentIds: ["agdef_1", "agdef_2"] });
  });
});

describe("fetchAgentSafetySummary", () => {
  it("reads the figures and forwards the signal", async () => {
    const summary = {
      toolCount: 4,
      runWithoutPerson: 1,
      leaveOrganization: 1,
      openWithSensitive: 0,
      resources: ["general"],
    };
    requestGraphQLMock.mockResolvedValue({ agentSafetySummary: summary });
    const controller = new AbortController();

    await expect(fetchAgentSafetySummary({ signal: controller.signal })).resolves.toEqual(summary);
    expect(lastCall().document).toBe(AgentSafetySummaryDocument);
    expect(lastCall().signal).toBe(controller.signal);
  });
});

describe("fetchAgentSafetyHeaders", () => {
  // An empty list is "no agents", never the null that reads every agent.
  it("reads nothing for no agents", async () => {
    await expect(fetchAgentSafetyHeaders([])).resolves.toEqual([]);
    expect(requestGraphQLMock).not.toHaveBeenCalled();
  });

  it("reads only the named agents, without their tools", async () => {
    const header = {
      agentId: "agdef_1",
      organizationShadow: false,
      agent: { id: "agdef_1", name: "Customer desk", enabled: true },
      reach: { accessMode: "Everyone", roles: [], warnings: [] },
    };
    requestGraphQLMock.mockResolvedValue({ agentSafety: [header] });

    const headers = await fetchAgentSafetyHeaders(["agdef_1"]);

    expect(lastCall().document).toBe(AgentSafetyDocument);
    expect(lastCall().variables).toEqual({ agentIds: ["agdef_1"] });
    expect(headers).toEqual([header]);
    expect(headers[0]).not.toHaveProperty("tools");
  });
});
