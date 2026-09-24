import type {
  AgentSafety,
  AgentToolAutonomy,
  AgentToolPolicy,
  AgentToolSafety,
} from "@/lib/graphql/agent-safety";

export function policy(overrides: Partial<AgentToolPolicy>): AgentToolPolicy {
  return {
    name: "assign_move",
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

export function tool(
  policyName: string,
  clean: Partial<AgentToolAutonomy>,
  tainted: Partial<AgentToolAutonomy> = clean,
  rule: Partial<AgentToolPolicy> = {},
): AgentToolSafety {
  return {
    policyName,
    policy: policy({ name: policyName, title: titleOf(policyName), ...rule }),
    clean: autonomy(clean),
    tainted: autonomy(tainted),
  };
}

export function safety(
  id: string,
  name: string,
  overrides: Partial<Omit<AgentSafety, "agent">> & { agent?: Partial<AgentSafety["agent"]> } = {},
): AgentSafety {
  const { agent, ...rest } = overrides;
  return {
    agentId: id,
    organizationShadow: false,
    tools: [],
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
