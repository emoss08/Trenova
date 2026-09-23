import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import { readEventStream } from "@trenova/shared/lib/sse";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import { z } from "zod";
import {
  agentDefinitionSchema,
  agentEventListSchema,
  agentTemplateListSchema,
  previewPromptResponseSchema,
  toolCatalogSchema,
  toolTrustListSchema,
  assistantMessagePageSchema,
  assistantArtifactListSchema,
  assistantArtifactSchema,
  agentBudgetStatusSchema,
  assistantPlanListSchema,
  assistantProposalListSchema,
  assistantProviderListSchema,
  assistantThreadListSchema,
  assistantThreadSchema,
  parseAssistantStreamEvent,
  saveAgentDefinitionRequestSchema,
  sendMessageResultSchema,
  type AgentDefinition,
  type AssistantArtifact,
  type AssistantEntityRef,
  type AssistantPageContext,
  type ThreadOrigin,
  type AssistantPlan,
  type AssistantProposal,
  type AssistantStreamEvent,
  type AssistantThread,
  type PlanDecision,
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

/**
 * A turn handed to a worker, and where to watch it. A quick question also
 * carries the thread the server made for it.
 */
const startedTurnSchema = z.object({
  turnId: z.string(),
  threadId: z.string(),
  streamUrl: z.string(),
  status: z.string(),
  thread: assistantThreadSchema.optional(),
});

export type StartedTurn = z.infer<typeof startedTurnSchema>;

/** A reply a conversation is still producing. */
export type ActiveTurn = {
  id: string;
  threadId: string;
  status: string;
  workflowId?: string;
  /** What started the turn: the person, or the application reporting a decision. */
  origin?: "Person" | "DecisionFollowUp";
  /** The question the turn answers, when a person asked one. */
  input?: string;
};

/** Where a stream that never opened went wrong, for the reader. */
export class AssistantStreamError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "AssistantStreamError";
    this.status = status;
  }
}

/** Where a conversation begins and, when it was opened from a record, which. */
export type StartThreadOptions = {
  title?: string;
  origin?: ThreadOrigin;
  subjectType?: string;
  subjectId?: string;
};

/** What rides with a message besides the words. */
export type SendMessageOptions = {
  context?: AssistantPageContext | null;
  providerId?: string;
  /** Documents uploaded to this thread for this message. */
  attachmentDocumentIds?: readonly string[];
  /** Records named from the composer. */
  mentions?: readonly AssistantEntityRef[];
};

/** A quick question from anywhere: no thread yet, the answer makes one. */
export type AskOptions = {
  context?: AssistantPageContext | null;
  mentions?: readonly AssistantEntityRef[];
};

/** What a person may change about their own conversation. */
export type UpdateThreadOptions = {
  title?: string;
  pinned?: boolean;
  /** Lists a quick question as a conversation. */
  keep?: boolean;
};

export class AssistantService {
  public async listThreads(limit = 50, offset = 0) {
    const response = await api.get(`/assistant/threads/?limit=${limit}&offset=${offset}`);
    return safeParse(assistantThreadListSchema, response, "Assistant Thread");
  }

  public async startThread(agentDefinitionId: string, options: StartThreadOptions = {}) {
    const response = await api.post("/assistant/threads/", {
      agentDefinitionId,
      title: options.title ?? "",
      origin: options.origin ?? "Panel",
      subjectType: options.subjectType ?? "",
      subjectId: options.subjectId || null,
    });
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  public async updateThread(id: AssistantThread["id"], options: UpdateThreadOptions) {
    const response = await api.patch(`/assistant/threads/${id}/`, {
      title: options.title ?? null,
      pinned: options.pinned ?? null,
      keep: options.keep ?? false,
    });
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  /**
   * What a conversation produced besides words. Read with the thread rather
   * than taken from the send response: an artifact outlives the turn, and a
   * draft's status follows the decision made on it anywhere.
   */
  public async listArtifacts(threadId: AssistantThread["id"], options?: { signal?: AbortSignal }) {
    const response = await api.get(`/assistant/threads/${threadId}/artifacts/`, {
      signal: options?.signal,
    });
    return safeParse(assistantArtifactListSchema, response, "Assistant Artifact");
  }

  public async pinArtifact(
    threadId: AssistantThread["id"],
    artifactId: AssistantArtifact["id"],
    pinned: boolean,
  ) {
    const response = await api.post(`/assistant/threads/${threadId}/artifacts/${artifactId}/pin/`, {
      pinned,
    });
    return safeParse(assistantArtifactSchema, response, "Assistant Artifact");
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
    options: SendMessageOptions = {},
  ) {
    const response = await api.post(
      `/assistant/threads/${threadId}/messages/`,
      messageBody(content, options),
    );
    return safeParse(sendMessageResultSchema, response, "Assistant Reply");
  }

  /**
   * Hands a question to a worker and returns the turn to watch.
   *
   * The reply is not on this response. It arrives on the turn's stream, which
   * means it survives this request ending — a deploy, a dropped connection, or
   * somebody closing the tab.
   *
   * Aborting `signal` withdraws the question while it is on its way; once the
   * turn is returned, only stopping it ends the reply.
   */
  public async startTurn(
    threadId: AssistantThread["id"],
    content: string,
    options: SendMessageOptions = {},
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<StartedTurn> {
    const response = await api.post(
      `/assistant/threads/${threadId}/turns/`,
      messageBody(content, options),
      { signal },
    );

    return safeParse(startedTurnSchema, response, "Assistant Turn");
  }

  /**
   * A quick question from the palette. It is answered on a hidden thread the
   * server makes for it, returned with the turn so the reader can keep the
   * conversation even when the answer fails partway.
   */
  public async startAsk(
    content: string,
    options: AskOptions = {},
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<StartedTurn> {
    const response = await api.post(
      "/assistant/ask/",
      {
        content,
        context: options.context ?? null,
        mentions: options.mentions ?? [],
      },
      { signal },
    );

    return safeParse(startedTurnSchema, response, "Assistant Turn");
  }

  /**
   * Follows a turn from where the reader left off.
   *
   * `cursor` is the id of the last event the caller *applied*, not the last it
   * received: those differ when a connection dies mid-frame, and resuming from
   * the received one loses an event. An empty cursor replays the turn from its
   * beginning, which is what a fresh tab attaching to a reply in progress
   * wants.
   */
  public async attachTurn(
    turnId: string,
    onEvent: (event: AssistantStreamEvent, cursor: string) => void,
    options: { cursor?: string; signal?: AbortSignal } = {},
  ): Promise<void> {
    const path = `/assistant/turns/${turnId}/stream/`;
    const headers: Record<string, string> = { Accept: "text/event-stream" };
    if (options.cursor) {
      headers["Last-Event-ID"] = options.cursor;
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
      method: "GET",
      headers,
      credentials: "include",
      signal: options.signal,
    });

    if (!response.ok || !response.body) {
      throw new AssistantStreamError(await streamFailureMessage(response), response.status);
    }

    await readEventStream(
      response.body,
      (message) => {
        const event = parseAssistantStreamEvent(message.event, message.data);
        if (event) {
          onEvent(event, message.id);
        }
      },
      options.signal,
    );
  }

  /**
   * Stops a reply nobody is waiting for.
   *
   * This has to reach the server. Aborting the reader used to stop the model,
   * because the model was running on the request being aborted; with the work
   * on a worker it stops nothing and the turn keeps billing.
   */
  public async stopTurn(turnId: string): Promise<void> {
    await api.post(`/assistant/turns/${turnId}/stop/`, {});
  }

  /** The reply a conversation is still producing, if it is producing one. */
  public async activeTurn(threadId: AssistantThread["id"]): Promise<ActiveTurn | null> {
    const response = await api.get(`/assistant/threads/${threadId}/turns/active/`);
    if (typeof response !== "object" || response === null || !("turn" in response)) {
      return null;
    }

    return ((response as { turn: ActiveTurn | null }).turn as ActiveTurn) ?? null;
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
  /**
   * The plans a thread's runs formed: several writes decided as one. Steps are
   * the proposals carrying the plan's id, so both lists are read together.
   */
  public async listPlans(threadId: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${threadId}/plans/`);
    return safeParse(assistantPlanListSchema, response, "Assistant Plan");
  }

  /**
   * Approving a plan runs every step in the order the agent asked, stopping at
   * the first that fails; rejecting it rejects them all. Like a single proposal
   * it is checked and audited as the approver, so it can fail on their
   * permissions rather than the asker's.
   */
  public async decidePlan(planId: AssistantPlan["id"], decision: PlanDecision) {
    await api.post(`/agent-plans/${planId}/resolve/`, { decision, reasonCode: "" });
  }

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

function messageBody(content: string, options: SendMessageOptions): Record<string, unknown> {
  return {
    content,
    context: options.context ?? null,
    providerId: options.providerId ?? "",
    attachmentDocumentIds: options.attachmentDocumentIds ?? [],
    mentions: options.mentions ?? [],
  };
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

  /** Where the agent stands against its caps, for the form that sets them. */
  public async budget(id: AgentDefinition["id"], options?: { signal?: AbortSignal }) {
    const response = await api.get(`/agent-definitions/${id}/budget/`, { signal: options?.signal });
    return safeParse(agentBudgetStatusSchema, response, "Agent Budget");
  }

  public async trust(id: AgentDefinition["id"], options?: { signal?: AbortSignal }) {
    const response = await api.get(`/agent-definitions/${id}/trust/`, { signal: options?.signal });
    return safeParse(toolTrustListSchema, response, "Agent Track Record");
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
