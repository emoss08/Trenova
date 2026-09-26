import {
  AiTrainingExportHistoryDocument,
  AgentControlFieldsFragmentDoc,
  AgentControlSettingsDocument,
  UpdateAgentControlDocument,
  type AgentControlFieldsFragment,
  type AgentControlInput,
  type AiTrainingExportHistoryQuery,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentControl = AgentControlFieldsFragment;

/** One key for the organization's switches, shared by every reader of them. */
export const AGENT_CONTROL_QUERY_KEY = ["agent-control"] as const;

export type TrainingExportHistoryEntry =
  AiTrainingExportHistoryQuery["aiTrainingExportHistory"][number];

/** The training exports that included this organization's corrections. */
export const TRAINING_EXPORT_HISTORY_QUERY_KEY = [
  ...AGENT_CONTROL_QUERY_KEY,
  "training-exports",
] as const;

export function trainingExportHistoryQueryOptions() {
  return {
    queryKey: TRAINING_EXPORT_HISTORY_QUERY_KEY,
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchTrainingExportHistory({ signal }),
  };
}

export async function fetchTrainingExportHistory(options?: {
  signal?: AbortSignal;
}): Promise<TrainingExportHistoryEntry[]> {
  const data = await requestGraphQL({
    document: AiTrainingExportHistoryDocument,
    operationName: "AITrainingExportHistory",
    signal: options?.signal,
  });

  return data.aiTrainingExportHistory;
}

export function agentControlQueryOptions() {
  return {
    queryKey: AGENT_CONTROL_QUERY_KEY,
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentControl({ signal }),
  };
}

export async function fetchAgentControl(options?: { signal?: AbortSignal }): Promise<AgentControl> {
  const data = await requestGraphQL({
    document: AgentControlSettingsDocument,
    operationName: "AgentControlSettings",
    signal: options?.signal,
  });

  return getFragmentData(AgentControlFieldsFragmentDoc, data.agentControl);
}

export async function updateAgentControl(input: AgentControlInput): Promise<AgentControl> {
  const data = await requestGraphQL({
    document: UpdateAgentControlDocument,
    operationName: "UpdateAgentControl",
    variables: { input },
  });

  return getFragmentData(AgentControlFieldsFragmentDoc, data.updateAgentControl);
}
