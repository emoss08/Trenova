import { StartAgentRunDocument, type AgentSubjectType } from "@trenova/graphql/generated/graphql";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import {
  agentRunSchema,
  startAgentRunRequestSchema,
  type AgentRun,
  type StartAgentRunRequest,
} from "@/types/agent-run";

/**
 * Where a background run's transcript is served: the run as a Markdown file,
 * named by the server, the same document a conversation downloads as.
 */
export function agentRunTranscriptUrl(runId: AgentRun["id"]): string {
  return `${API_BASE_URL}/agent-runs/${encodeURIComponent(runId)}/transcript/`;
}

export function downloadAgentRunTranscript(runId: AgentRun["id"]): void {
  downloadFromUrl(agentRunTranscriptUrl(runId));
}

export class AgentRunService {
  public async start(payload: StartAgentRunRequest) {
    const request = startAgentRunRequestSchema.parse(payload);
    const data = await requestGraphQL({
      document: StartAgentRunDocument,
      operationName: "StartAgentRun",
      variables: {
        input: {
          agentDefinitionId: request.agentDefinitionId ?? null,
          systemKey: request.systemKey ?? null,
          subjectType: (request.subjectType as AgentSubjectType | undefined) ?? null,
          subjectId: request.subjectId ?? null,
        },
      },
    });
    return safeParse(agentRunSchema, data.startAgentRun, "Agent Run");
  }
}
