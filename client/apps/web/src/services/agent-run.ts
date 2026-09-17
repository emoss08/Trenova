import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  agentRunSchema,
  startAgentRunRequestSchema,
  type AgentRun,
  type StartAgentRunRequest,
} from "@/types/agent-run";

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
