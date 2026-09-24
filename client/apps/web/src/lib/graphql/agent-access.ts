import {
  AgentAccessFieldsFragmentDoc,
  AgentAccessPreviewDocument,
  AgentAccessRoleFieldsFragmentDoc,
  RoleAgentAccessDocument,
  RoleAgentAccessFieldsFragmentDoc,
  SetAgentAccessDocument,
  SetRoleAgentAccessDocument,
  SuggestedAgentAudienceDocument,
  type AgentAccessMode,
  type AgentAccessPreviewQuery,
  type AgentAudienceCoverage,
  type RoleAgentAccessFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type { AgentAccessMode, AgentAudienceCoverage };

/** A role as the access controls draw it. */
export type AgentAccessRole = {
  id: string;
  name: string;
  description: string;
  isSystem: boolean;
};

/** Who may use an agent: everyone who may use the assistant, or the roles granted it. */
export type AgentAccess = {
  mode: AgentAccessMode;
  /** Kept whatever the mode, so an agent opened to everyone and restricted again keeps them. */
  roleIds: string[];
};

/** How much of an agent one role could use, and whether it is granted it now. */
export type AgentAudienceRole = {
  role: AgentAccessRole;
  coverage: AgentAudienceCoverage;
  /** The resources the agent's tools need that the role does not grant. */
  missingResources: string[];
  granted: boolean;
};

export type AgentAudienceSuggestion = {
  agentId: string;
  accessMode: AgentAccessMode;
  roles: AgentAudienceRole[];
  /** While the agent is open to everyone, its tools that reach restricted or confidential data. */
  sensitiveTools: string[];
};

/** Who an agent as the form holds it could be given to, before it is saved. */
export type AgentAccessPreview = Omit<AgentAudienceSuggestion, "agentId">;

/**
 * The agent as the form holds it: the saved agent it edits, if any, the tools
 * chosen and who may use it. Normalized so the same form state is always the
 * same request, and so the same cache entry.
 */
export type AgentAccessPreviewRequest = {
  /** The agent being edited; empty for one not yet saved. */
  agentId: string;
  toolNames: string[];
  accessMode: AgentAccessMode;
};

/** The most tools one preview asks about, as the server allows. */
export const MAX_PREVIEW_TOOLS = 200;

export function agentAccessPreviewRequest({
  agentId,
  toolNames,
  accessMode,
}: {
  agentId: string;
  toolNames: readonly string[];
  accessMode: AgentAccessMode;
}): AgentAccessPreviewRequest {
  const tools = [...new Set(toolNames.map((name) => name.trim()).filter((name) => name !== ""))];
  tools.sort();

  return { agentId, toolNames: tools.slice(0, MAX_PREVIEW_TOOLS), accessMode };
}

function toAudienceRoles(
  roles: AgentAccessPreviewQuery["agentAccessPreview"]["roles"],
): AgentAudienceRole[] {
  return roles.map((entry) => ({
    role: getFragmentData(AgentAccessRoleFieldsFragmentDoc, entry.role),
    coverage: entry.coverage,
    missingResources: [...entry.missingResources],
    granted: entry.granted,
  }));
}

/**
 * Each role's coverage of the tools the form holds and, while the form says
 * everyone may use it, the sensitive ones among them. Worked out from the
 * form, not from the saved agent, and changes nothing.
 */
export async function fetchAgentAccessPreview(
  request: AgentAccessPreviewRequest,
  options?: RequestOptions,
): Promise<AgentAccessPreview> {
  const data = await requestGraphQL({
    document: AgentAccessPreviewDocument,
    operationName: "AgentAccessPreview",
    variables: {
      input: {
        ...(request.agentId ? { agentId: request.agentId } : {}),
        toolNames: request.toolNames,
        accessMode: request.accessMode,
      },
    },
    signal: options?.signal,
  });
  const preview = data.agentAccessPreview;

  return {
    accessMode: preview.accessMode,
    sensitiveTools: [...preview.sensitiveTools],
    roles: toAudienceRoles(preview.roles),
  };
}

export type RoleAgent = RoleAgentAccessFieldsFragment["agents"][number];

export type RoleAgents = {
  roleId: string;
  roleName: string;
  agents: RoleAgent[];
};

/** Each role's coverage of an agent's tools, for choosing who may use it. */
export async function fetchSuggestedAgentAudience(
  agentId: string,
  options?: RequestOptions,
): Promise<AgentAudienceSuggestion> {
  const data = await requestGraphQL({
    document: SuggestedAgentAudienceDocument,
    operationName: "SuggestedAgentAudience",
    variables: { agentId },
    signal: options?.signal,
  });
  const suggestion = data.suggestedAgentAudience;

  return {
    agentId: suggestion.agentId,
    accessMode: suggestion.accessMode,
    sensitiveTools: [...suggestion.sensitiveTools],
    roles: toAudienceRoles(suggestion.roles),
  };
}

/**
 * Sets who may use an agent, replacing the roles granted it. Needs
 * permission to update agents and roles.
 */
export async function setAgentAccess(
  agentId: string,
  access: AgentAccess,
): Promise<{ mode: AgentAccessMode; roles: AgentAccessRole[] }> {
  const data = await requestGraphQL({
    document: SetAgentAccessDocument,
    operationName: "SetAgentAccess",
    variables: {
      agentId,
      input: { accessMode: access.mode, roleIds: [...new Set(access.roleIds)] },
    },
  });
  const saved = getFragmentData(AgentAccessFieldsFragmentDoc, data.setAgentAccess);

  return {
    mode: saved.accessMode,
    roles: saved.accessRoles.map((role) => getFragmentData(AgentAccessRoleFieldsFragmentDoc, role)),
  };
}

function toRoleAgents(fields: RoleAgentAccessFieldsFragment): RoleAgents {
  return { roleId: fields.id, roleName: fields.name, agents: [...fields.agents] };
}

/** The agents a role is granted. Null when the role is not found. */
export async function fetchRoleAgents(
  roleId: string,
  options?: RequestOptions,
): Promise<RoleAgents | null> {
  const data = await requestGraphQL({
    document: RoleAgentAccessDocument,
    operationName: "RoleAgentAccess",
    variables: { id: roleId },
    signal: options?.signal,
  });
  if (!data.role) {
    return null;
  }

  return toRoleAgents(getFragmentData(RoleAgentAccessFieldsFragmentDoc, data.role));
}

/** Replaces the agents a role is granted. */
export async function setRoleAgentAccess(
  roleId: string,
  agentIds: readonly string[],
): Promise<RoleAgents> {
  const data = await requestGraphQL({
    document: SetRoleAgentAccessDocument,
    operationName: "SetRoleAgentAccess",
    variables: { roleId, agentIds: [...new Set(agentIds)] },
  });

  return toRoleAgents(getFragmentData(RoleAgentAccessFieldsFragmentDoc, data.setRoleAgentAccess));
}
