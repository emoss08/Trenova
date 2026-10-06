import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
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
    const response = await api.post("/agent-runs/", request);
    return safeParse(agentRunSchema, response, "Agent Run");
  }

  public async get(id: AgentRun["id"]) {
    const response = await api.get(`/agent-runs/${id}/`);
    return safeParse(agentRunSchema, response, "Agent Run");
  }
}
