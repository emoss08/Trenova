import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentDefinitionCardDocument,
  AgentDraftingAvailableDocument,
  DraftAgentFromDescriptionDocument,
  TightenAgentInstructionsDocument,
  type DraftAgentFromDescriptionMutation,
  type TightenAgentInstructionsMutation,
  AgentDefinitionCardFieldsFragmentDoc,
  AgentDefinitionVersionDraftDocument,
  AgentDefinitionVersionsDocument,
  AgentInstructionLintDocument,
  AgentShadowReportDocument,
  type AgentDefinitionVersionsQuery,
  type AgentInstructionLintQuery,
  type AgentShadowReportQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AgentDefinitionRow } from "./agent-definition";

type RequestOptions = { signal?: AbortSignal };

export type InstructionFinding = AgentInstructionLintQuery["agentInstructionLint"][number];
export type AgentShadowReport = AgentShadowReportQuery["agentShadowReport"];
export type AgentVersion = AgentDefinitionVersionsQuery["agentDefinitionVersions"][number];

/** How many saves the version history lists. */
export const AGENT_VERSIONS_SHOWN = 20;

export type InstructionLintRequest = {
  instructions: string;
  toolNames: readonly string[];
};

/** What a draft's instructions ask for that none of its tools can do. Saves nothing. */
export async function fetchInstructionLint(
  request: InstructionLintRequest,
  options?: RequestOptions,
): Promise<InstructionFinding[]> {
  const data = await requestGraphQL({
    document: AgentInstructionLintDocument,
    operationName: "AgentInstructionLint",
    variables: {
      input: { instructions: request.instructions, toolNames: [...request.toolNames] },
    },
    signal: options?.signal,
  });
  return data.agentInstructionLint;
}

/** How an agent in shadow compares with the people doing the same work. */
export async function fetchAgentShadowReport(
  agentId: string,
  days: number,
  options?: RequestOptions,
): Promise<AgentShadowReport> {
  const data = await requestGraphQL({
    document: AgentShadowReportDocument,
    operationName: "AgentShadowReport",
    variables: { agentId, days },
    signal: options?.signal,
  });
  return data.agentShadowReport;
}

/** An agent's saves, newest first. */
export async function fetchAgentVersions(
  agentId: string,
  options?: RequestOptions,
): Promise<AgentVersion[]> {
  const data = await requestGraphQL({
    document: AgentDefinitionVersionsDocument,
    operationName: "AgentDefinitionVersions",
    variables: { agentId, limit: AGENT_VERSIONS_SHOWN },
    signal: options?.signal,
  });
  return data.agentDefinitionVersions;
}

/** An earlier version as a draft of the agent as it is now. Saves nothing. */
export async function fetchAgentVersionDraft(
  agentId: string,
  version: number,
  options?: RequestOptions,
): Promise<AgentDefinitionRow> {
  const data = await requestGraphQL({
    document: AgentDefinitionVersionDraftDocument,
    operationName: "AgentDefinitionVersionDraft",
    variables: { agentId, version },
    signal: options?.signal,
  });
  return getFragmentData(AgentDefinitionCardFieldsFragmentDoc, data.agentDefinitionVersionDraft);
}

/** One agent as the roster reads it, or null when it is gone. */
export async function fetchAgentDefinition(
  id: string,
  options?: RequestOptions,
): Promise<AgentDefinitionRow | null> {
  const data = await requestGraphQL({
    document: AgentDefinitionCardDocument,
    operationName: "AgentDefinitionCard",
    variables: { id },
    signal: options?.signal,
  });
  return data.agentDefinition
    ? getFragmentData(AgentDefinitionCardFieldsFragmentDoc, data.agentDefinition)
    : null;
}

export type AgentDraft = DraftAgentFromDescriptionMutation["draftAgentFromDescription"];
export type TightenedInstructions = TightenAgentInstructionsMutation["tightenAgentInstructions"];

/** Whether Nova can draft agents and tighten instructions: a provider takes the assistant's work. */
export async function fetchAgentDraftingAvailable(options?: RequestOptions): Promise<boolean> {
  const data = await requestGraphQL({
    document: AgentDraftingAvailableDocument,
    operationName: "AgentDraftingAvailable",
    signal: options?.signal,
  });
  return data.agentDraftingAvailable;
}

/** An agent drafted from a description of its job. Saves nothing. */
export async function draftAgentFromDescription(description: string): Promise<AgentDraft> {
  const data = await requestGraphQL({
    document: DraftAgentFromDescriptionDocument,
    operationName: "DraftAgentFromDescription",
    variables: { description },
  });
  return data.draftAgentFromDescription;
}

/** Instructions with the same meaning, shorter. Saves nothing. */
export async function tightenAgentInstructions(instructions: string): Promise<TightenedInstructions> {
  const data = await requestGraphQL({
    document: TightenAgentInstructionsDocument,
    operationName: "TightenAgentInstructions",
    variables: { instructions },
  });
  return data.tightenAgentInstructions;
}
