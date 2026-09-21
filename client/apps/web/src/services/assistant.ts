import { api, withCsrfHeader } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import { readEventStream } from "@trenova/shared/lib/sse";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import {
  agentDefinitionSchema,
  agentEventListSchema,
  agentTemplateListSchema,
  previewPromptResponseSchema,
  toolCatalogSchema,
  assistantMessagePageSchema,
  assistantProposalListSchema,
  assistantProviderListSchema,
  assistantThreadListSchema,
  assistantThreadSchema,
  parseAssistantStreamEvent,
  saveAgentDefinitionRequestSchema,
  sendMessageResultSchema,
  type AgentDefinition,
  type AssistantPageContext,
  type AssistantProposal,
  type AssistantStreamEvent,
  type AssistantThread,
  type ProposalDecision,
  type SaveAgentDefinitionRequest,
} from "@/types/assistant";

/**
 * Where a conversation's transcript is served: the whole thread as a Markdown
 * file, named by the server. Same-origin, so the session cookie authenticates
 * it and a plain link downloads it.
 */
export function assistantTranscriptUrl(threadId: AssistantThread["id"]): string {
  return `${API_BASE_URL}/assistant/threads/${encodeURIComponent(threadId)}/transcript/`;
}

export function downloadAssistantTranscript(threadId: AssistantThread["id"]): void {
  downloadFromUrl(assistantTranscriptUrl(threadId));
}

/** Where a stream that never opened went wrong, for the reader. */
export class AssistantStreamError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "AssistantStreamError";
    this.status = status;
  }
}

export class AssistantService {
  public async listThreads(limit = 50, offset = 0) {
    const response = await api.get(`/assistant/threads/?limit=${limit}&offset=${offset}`);
    return safeParse(assistantThreadListSchema, response, "Assistant Thread");
  }

  public async startThread(agentDefinitionId: string, title = "") {
    const response = await api.post("/assistant/threads/", { agentDefinitionId, title });
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  public async getThread(id: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${id}/`);
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  public async deleteThread(id: AssistantThread["id"]) {
    await api.delete(`/assistant/threads/${id}/`);
  }

  /**
   * One page of a thread: the newest `limit` messages, or with `before` the
   * page above the message carrying that sequence.
   */
  public async listMessages(
    threadId: AssistantThread["id"],
    { limit, before, signal }: { limit?: number; before?: number; signal?: AbortSignal } = {},
  ) {
    const params = new URLSearchParams();
    if (limit !== undefined) {
      params.set("limit", String(limit));
    }
    if (before !== undefined) {
      params.set("before", String(before));
    }
    const query = params.size > 0 ? `?${params.toString()}` : "";
    const response = await api.get(`/assistant/threads/${threadId}/messages/${query}`, { signal });
    return safeParse(assistantMessagePageSchema, response, "Assistant Message");
  }

  /**
   * A refused turn comes back as a normal result with `refused` set, not as an
   * error: the turn was processed, recorded, and explained.
   */
  /** The models this organization has made available to the assistant. */
  public async listProviders() {
    const response = await api.get("/assistant/providers/");
    // safeParse resolves a promise, so the await belongs here rather than on
    // the caller: reading .results off the promise itself yields undefined, the
    // query stores undefined, and the picker quietly renders nothing.
    const parsed = await safeParse(assistantProviderListSchema, response, "Assistant Providers");

    return parsed.results;
  }

  public async sendMessage(
    threadId: AssistantThread["id"],
    content: string,
    context: AssistantPageContext | null = null,
    providerId = "",
  ) {
    const response = await api.post(`/assistant/threads/${threadId}/messages/`, {
      content,
      context,
      providerId,
    });
    return safeParse(sendMessageResultSchema, response, "Assistant Reply");
  }

  /**
   * Sends a message and reports the turn as it happens. Resolves when the
   * stream closes or the signal aborts; a stream that could not be opened at
   * all rejects with the server's message, so the caller can offer a retry.
   */
  public async streamMessage(
    threadId: AssistantThread["id"],
    content: string,
    onEvent: (event: AssistantStreamEvent) => void,
    signal?: AbortSignal,
    context: AssistantPageContext | null = null,
    providerId = "",
  ): Promise<void> {
    const path = `/assistant/threads/${threadId}/messages/stream/`;
    const response = await fetch(`${API_BASE_URL}${path}`, {
      method: "POST",
      headers: await withCsrfHeader(
        "POST",
        { "Content-Type": "application/json", Accept: "text/event-stream" },
        path,
      ),
      body: JSON.stringify({ content, context, providerId }),
      credentials: "include",
      signal,
    });

    if (!response.ok || !response.body) {
      throw new AssistantStreamError(await streamFailureMessage(response), response.status);
    }

    await readEventStream(
      response.body,
      (message) => {
        const event = parseAssistantStreamEvent(message.event, message.data);
        if (event) {
          onEvent(event);
        }
      },
      signal,
    );
  }

  /**
   * Proposals outlive the turn that raised them, so reopening a thread has to
   * fetch them rather than rely on the send response.
   */
  public async listProposals(threadId: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${threadId}/proposals/`);
    return safeParse(assistantProposalListSchema, response, "Assistant Proposal");
  }

  /**
   * Deciding goes through the agent proposal endpoint rather than a chat-specific
   * one: a proposal raised in conversation is the same kind of record as one
   * raised by the billing agent, and it is checked, executed and audited the same
   * way. Accepting runs the tool as the approver, so this can fail on their own
   * permissions even though the message was theirs.
   */
  public async decideProposal(
    proposalId: AssistantProposal["id"],
    decision: ProposalDecision,
    modifications?: Record<string, unknown>,
  ) {
    await api.post(`/agent-proposals/${proposalId}/resolve/`, {
      decision,
      modifications: modifications ?? {},
    });
  }
}

/**
 * The message for a stream the server refused to open. The error body follows
 * the API's usual shape when the failure is one it wrote (permission, a
 * disabled agent, an empty message); anything else gets a plain explanation.
 */
async function streamFailureMessage(response: Response): Promise<string> {
  try {
    const body: unknown = await response.json();
    if (body && typeof body === "object") {
      const record = body as { error?: { message?: string }; message?: string };
      const message = record.error?.message ?? record.message;
      if (typeof message === "string" && message.trim() !== "") {
        return message;
      }
    }
  } catch {
    // The body was not JSON; fall through to the generic wording.
  }

  return response.status === 403
    ? "You do not have permission to use the assistant."
    : "The assistant could not be reached. Try again in a moment.";
}

export class AgentDefinitionService {
  public async getBySystemKey(systemKey: string) {
    const response = await api.get(`/agent-definitions/system/${systemKey}/`);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async tools() {
    const response = await api.get("/agent-definitions/tools/");
    return safeParse(toolCatalogSchema, response, "Agent Tool");
  }

  public async eventKinds() {
    const response = await api.get("/agent-definitions/event-kinds/");
    return safeParse(agentEventListSchema, response, "Agent Event");
  }

  public async previewPrompt(payload: SaveAgentDefinitionRequest) {
    const request = saveAgentDefinitionRequestSchema.parse(payload);
    const response = await api.post("/agent-definitions/preview-prompt/", request);
    return safeParse(previewPromptResponseSchema, response, "Agent Prompt");
  }

  public async get(id: AgentDefinition["id"]) {
    const response = await api.get(`/agent-definitions/${id}/`);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async templates() {
    const response = await api.get("/agent-definitions/templates/");
    return safeParse(agentTemplateListSchema, response, "Agent Template");
  }

  public async create(payload: SaveAgentDefinitionRequest) {
    const request = saveAgentDefinitionRequestSchema.parse(payload);
    const response = await api.post("/agent-definitions/", request);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async update(id: AgentDefinition["id"], payload: SaveAgentDefinitionRequest) {
    const request = saveAgentDefinitionRequestSchema.parse(payload);
    const response = await api.put(`/agent-definitions/${id}/`, request);
    return safeParse(agentDefinitionSchema, response, "Agent");
  }

  public async remove(id: AgentDefinition["id"]) {
    await api.delete(`/agent-definitions/${id}/`);
  }
}
