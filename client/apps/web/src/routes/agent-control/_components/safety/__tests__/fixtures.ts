import type {
  AgentSafetyHeader,
  AgentToolAutonomy,
  AgentToolPolicy,
  AgentToolRuleRow,
  AgentToolSafetyRow,
} from "@/lib/graphql/agent-safety";

/** A rule as the server sends it; the id is the tool's name, as the server keys it. */
export function policy(overrides: Partial<AgentToolPolicy>): AgentToolPolicy {
  const name = overrides.name ?? "assign_move";
  return {
    id: name,
    name,
    title: "Assign move",
    kind: "Action",
    needs: { resource: "shipment_move", operation: "update" },
    scope: "Tenant",
    defaultTier: "ActWithApproval",
    maxTier: "AutoExecute",
    promotableTier: "AutoExecute",
    egress: ["Internal"],
    leavesOrganization: false,
    hasClassify: false,
    hasCondition: false,
    conditionDescription: null,
    personalExemption: false,
    effect: "Change",
    artifact: "",
    reversible: true,
    idempotent: true,
    readsExternal: "Never",
    source: null,
    carriesTaint: false,
    rationale: "Assigning a move changes only internal records.",
    explanation: "Runs at the tier the agent sets for it.",
    ...overrides,
  };
}

/** A row of the tool rules table, which also says whether it runs without a person. */
export function rule(
  overrides: Partial<AgentToolPolicy>,
  runsWithoutPerson: boolean | null = null,
): AgentToolRuleRow {
  return { ...policy(overrides), runsWithoutPerson };
}

export function autonomy(overrides: Partial<AgentToolAutonomy>): AgentToolAutonomy {
  return {
    answer: "RUNS_ON_ITS_OWN",
    tier: "AutoExecute",
    heldBy: [],
    earned: false,
    approvalsToNext: null,
    ...overrides,
  };
}

/** "email_customer" reads "Email customer", as the server titles it. */
function titleOf(name: string): string {
  const words = name.replace(/_/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/** One tool an agent holds; the row is keyed by the agent and the tool together. */
export function tool(
  policyName: string,
  clean: Partial<AgentToolAutonomy>,
  tainted: Partial<AgentToolAutonomy> = clean,
  rule: Partial<AgentToolPolicy> = {},
  agent: { id: string; name: string } = { id: "agdef_1", name: "Customer desk" },
): AgentToolSafetyRow {
  return {
    id: `${agent.id}:${policyName}`,
    agentId: agent.id,
    agentName: agent.name,
    policyName,
    policy: policy({ name: policyName, title: titleOf(policyName), ...rule }),
    clean: autonomy(clean),
    tainted: autonomy(tainted),
  };
}

export function safety(
  id: string,
  name: string,
  overrides: Partial<Omit<AgentSafetyHeader, "agent">> & {
    agent?: Partial<AgentSafetyHeader["agent"]>;
  } = {},
): AgentSafetyHeader {
  const { agent, ...rest } = overrides;
  return {
    agentId: id,
    organizationShadow: false,
    reach: { accessMode: "Everyone", roles: [], warnings: [] },
    ...rest,
    agent: {
      id,
      name,
      enabled: true,
      shadowMode: false,
      simulationMode: false,
      autonomyCeiling: "AutoExecute",
      triggerMode: "Chat",
      ...agent,
    },
  };
}
