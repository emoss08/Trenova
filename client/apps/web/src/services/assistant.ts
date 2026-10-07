import { api, withCsrfHeader } from "@trenova/shared/lib/api";
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
  mentionCandidateListSchema,
  deskSearchResultListSchema,
  threadBudgetSchema,
  type DeskSearchKind,
  type MentionSearchType,
  assistantArtifactListSchema,
  assistantArtifactPageSchema,
  assistantArtifactSchema,
  documentRewriteSchema,
  assistantLiveTurnListSchema,
  agentBudgetStatusSchema,
  assistantPlanListSchema,
  assistantProposalListSchema,
  proposalEditsSchema,
  assistantProviderListSchema,
  assistantThreadListSchema,
  assistantThreadSchema,
  conversationScheduleListSchema,
  conversationScheduleSchema,
  createdScheduleSchema,
  handoffResultSchema,
  parseAssistantStreamEvent,
  saveAgentDefinitionRequestSchema,
  sendMessageResultSchema,
  type AgentDefinition,
  type AssistantArtifact,
  type AssistantEntityRef,
  type AssistantPageContext,
  type ThreadOrigin,
  type AssistantStreamEvent,
  type AssistantThread,
  type ConversationSchedule,
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

/** A table or report preview, read again from its source as a whole CSV. */
export function artifactCsvUrl(threadId: string, artifactId: string): string {
  return `${API_BASE_URL}/assistant/threads/${encodeURIComponent(threadId)}/artifacts/${encodeURIComponent(artifactId)}/export.csv`;
}

/** A document version printed as a PDF or written as a Word file. */
export function artifactDocumentUrl(
  threadId: string,
  artifactId: string,
  format: "pdf" | "docx",
): string {
  return `${API_BASE_URL}/assistant/threads/${encodeURIComponent(threadId)}/artifacts/${encodeURIComponent(artifactId)}/export/?format=${format}`;
}

/** What narrows a page of a conversation's artifacts, on the server. */
export type ArtifactListParams = {
  cursor?: string;
  limit?: number;
  q?: string;
  kind?: string;
  pinned?: boolean;
};

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
  /**
   * What started the turn: the person, the application reporting a decision,
   * a request the person scheduled coming round, or the conversation being
   * compacted.
   */
  origin?: "Person" | "DecisionFollowUp" | "Scheduled" | "Compaction";
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
  /** Replaces what the agents keep in mind for the whole conversation. */
  pinnedFacts?: readonly string[];
  /** Lists a quick question as a conversation. */
  keep?: boolean;
  /** Whether the conversation compacts itself on nearing a full context. */
  autoCompact?: boolean;
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
      pinnedFacts: options.pinnedFacts ?? null,
      keep: options.keep ?? false,
      autoCompact: options.autoCompact ?? null,
    });
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  /**
   * Starts summarizing the older part of a conversation to free its context.
   * It runs as the conversation's turn, followed on its stream and stopped
   * like a reply.
   */
  public async compactThread(threadId: AssistantThread["id"]): Promise<StartedTurn> {
    const response = await api.post(`/assistant/threads/${threadId}/compact/`, {});
    return safeParse(startedTurnSchema, response, "Assistant Turn");
  }

  /**
   * What a conversation produced besides words. Read with the thread rather
   * than taken from the send response: an artifact outlives the turn, and a
   * draft's status follows the decision made on it anywhere.
   */
  public async listArtifacts(
    threadId: AssistantThread["id"],
    options?: { signal?: AbortSignal } & ArtifactListParams,
  ) {
    const params = new URLSearchParams();
    if (options?.cursor) params.set("cursor", options.cursor);
    if (options?.limit) params.set("limit", String(options.limit));
    if (options?.q) params.set("q", options.q);
    if (options?.kind) params.set("kind", options.kind);
    if (options?.pinned) params.set("pinned", "true");
    const query = params.toString();
    const response = await api.get(
      `/assistant/threads/${threadId}/artifacts/${query ? `?${query}` : ""}`,
      { signal: options?.signal },
    );
    return safeParse(assistantArtifactPageSchema, response, "Assistant Artifact");
  }

  /** Every version of the lineage an artifact belongs to, oldest first. */
  public async artifactLineage(
    threadId: AssistantThread["id"],
    artifactId: string,
    options?: { signal?: AbortSignal },
  ) {
    const response = await api.get(`/assistant/threads/${threadId}/artifacts/${artifactId}/`, {
      signal: options?.signal,
    });
    return safeParse(assistantArtifactListSchema, response, "Assistant Artifact");
  }

  /** The lineage a link names by its slug. */
  public async artifactBySlug(
    threadId: AssistantThread["id"],
    slug: string,
    options?: { signal?: AbortSignal },
  ) {
    const response = await api.get(
      `/assistant/threads/${threadId}/artifacts/by-slug/${encodeURIComponent(slug)}/`,
      { signal: options?.signal },
    );
    return safeParse(assistantArtifactListSchema, response, "Assistant Artifact");
  }

  public async saveDocumentVersion(
    threadId: AssistantThread["id"],
    artifactId: string,
    body: { body: string; note: string },
  ) {
    const response = await api.post(
      `/assistant/threads/${threadId}/artifacts/${artifactId}/versions/`,
      body,
    );
    return safeParse(assistantArtifactSchema, response, "Assistant Artifact");
  }

  public async restoreDocumentVersion(threadId: AssistantThread["id"], artifactId: string) {
    const response = await api.post(
      `/assistant/threads/${threadId}/artifacts/${artifactId}/restore/`,
      {},
    );
    return safeParse(assistantArtifactSchema, response, "Assistant Artifact");
  }

  public async rewriteDocument(
    threadId: AssistantThread["id"],
    artifactId: string,
    body: { text: string; mode: "shorter" | "plain" | "ask"; prompt: string },
  ) {
    const response = await api.post(
      `/assistant/threads/${threadId}/artifacts/${artifactId}/rewrite/`,
      body,
    );
    return safeParse(documentRewriteSchema, response, "Document Rewrite");
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

  /**
   * The requests scheduled in one conversation, newest first. A conversation
   * keeps at most 25, so the first page is all of them.
   */
  public async listThreadSchedules(
    threadId: AssistantThread["id"],
    options?: { signal?: AbortSignal; limit?: number; offset?: number },
  ) {
    const response = await api.get(
      `/assistant/threads/${threadId}/schedules/?limit=${options?.limit ?? 25}&offset=${options?.offset ?? 0}`,
      { signal: options?.signal },
    );
    return safeParse(conversationScheduleListSchema, response, "Conversation Schedule");
  }

  /** Every request the person scheduled, across their conversations, newest first. */
  public async listSchedules(options?: { signal?: AbortSignal; limit?: number; offset?: number }) {
    const response = await api.get(
      `/assistant/schedules/?limit=${options?.limit ?? 25}&offset=${options?.offset ?? 0}`,
      { signal: options?.signal },
    );
    return safeParse(conversationScheduleListSchema, response, "Conversation Schedule");
  }

  /**
   * Schedules a message: "every weekday at 7:30am, …" or "/schedule …". The
   * server reads the cadence and answers with the schedule and its card.
   */
  public async createSchedule(threadId: AssistantThread["id"], content: string) {
    const response = await api.post(`/assistant/threads/${threadId}/schedules/`, { content });
    return safeParse(createdScheduleSchema, response, "Conversation Schedule");
  }

  /** Pauses or resumes a schedule. */
  public async setScheduleEnabled(id: ConversationSchedule["id"], enabled: boolean) {
    const response = await api.patch(`/assistant/schedules/${id}/`, { enabled });
    return safeParse(conversationScheduleSchema, response, "Conversation Schedule");
  }

  public async deleteSchedule(id: ConversationSchedule["id"]) {
    await api.delete(`/assistant/schedules/${id}/`);
  }

  /** Asks a schedule's request now; answers with the turn to watch. */
  public async runSchedule(id: ConversationSchedule["id"]) {
    const response = await api.post(`/assistant/schedules/${id}/run/`, {});
    return safeParse(startedTurnSchema, response, "Assistant Turn");
  }

  public async getThread(id: AssistantThread["id"]) {
    const response = await api.get(`/assistant/threads/${id}/`);
    return safeParse(assistantThreadSchema, response, "Assistant Thread");
  }

  /** How close the thread's agent, and the person asking, are to their caps. */
  public async threadBudget(id: AssistantThread["id"], { signal }: { signal?: AbortSignal } = {}) {
    const response = await api.get(`/assistant/threads/${id}/budget/`, { signal });
    return safeParse(threadBudgetSchema, response, "Assistant Thread Budget");
  }

  /** Asks the people who run AI Control for what the person ran out of. */
  public async requestMore(
    id: AssistantThread["id"],
    kind: "access" | "allowance" | "budget" | "daily_runs",
  ) {
    const response = await api.post(`/assistant/threads/${id}/requests/`, { kind });
    return response as { sent: number };
  }

  /**
   * Takes the conversation to another agent: a new conversation with it,
   * opened with a summary, the pinned facts and the pinned artifacts, and a
   * card here saying so.
   */
  public async handoffThread(
    id: AssistantThread["id"],
    agentDefinitionId: string,
    facts: readonly string[] = [],
  ) {
    const response = await api.post(`/assistant/threads/${id}/handoff/`, {
      agentDefinitionId,
      facts,
    });
    return safeParse(handoffResultSchema, response, "Assistant Handoff");
  }

  public async deleteThread(id: AssistantThread["id"]) {
    await api.delete(`/assistant/threads/${id}/`);
  }

  /** Marks everything in the conversation as seen by its owner. */
  public async markThreadRead(id: AssistantThread["id"]) {
    await api.post(`/assistant/threads/${id}/read/`);
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
  /**
   * Records a person can name with @ from the composer, of one kind or of
   * every kind they may read, matched on what the record is called.
   */
  public async searchMentions(
    query: string,
    type: MentionSearchType,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const params = new URLSearchParams({ query, type });
    const response = await api.get(`/assistant/mentions/?${params.toString()}`, { signal });
    const parsed = await safeParse(mentionCandidateListSchema, response, "Assistant Mentions");

    return parsed.results;
  }

  /**
   * The person's own conversations, what was said in them and what they
   * produced. With nothing typed, the conversations and artifacts that are
   * recent.
   */
  public async searchDesk(
    query: string,
    kind: DeskSearchKind,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const params = new URLSearchParams({ query, kind });
    const response = await api.get(`/assistant/search/?${params.toString()}`, { signal });
    const parsed = await safeParse(deskSearchResultListSchema, response, "Desk Search");

    return parsed.results;
  }

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

  /**
   * Every reply this person's conversations are still producing. A reply runs
   * on a worker until it ends or is stopped, whether or not anything is
   * reading it, so this is the one place that knows what is under way when
   * the panel is closed or the conversation was left.
   */
  public async listActiveTurns(options?: { signal?: AbortSignal }) {
    const response = await api.get("/assistant/turns/active/", { signal: options?.signal });
    return safeParse(assistantLiveTurnListSchema, response, "Assistant Turns");
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
   * Keeps the values a person changed on a pending proposal, such as the
   * wording of a drafted message, so the edit survives a reload and goes with
   * the approval. Saving is not deciding; an empty set clears what was saved.
   */
  public async saveProposalEdits(
    threadId: AssistantThread["id"],
    proposalId: string,
    modifications: Record<string, unknown>,
  ) {
    const response = await api.put(
      `/assistant/threads/${threadId}/proposals/${proposalId}/edits/`,
      { modifications },
    );
    return safeParse(proposalEditsSchema, response, "Proposal Edits");
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

  /**
   * Tries a draft once against live records, every write simulated, and hands each
   * server-sent event to `onMessage` as it arrives. Nothing is saved. Resolves when the
   * stream ends; an abort ends it quietly.
   */
  public async dryRun(
    payload: { agentId: string | null; draft: SaveAgentDefinitionRequest; prompt: string },
    onMessage: (event: string, data: string) => void,
    options: { signal?: AbortSignal } = {},
  ): Promise<void> {
    const path = "/agent-definitions/dry-run/";
    const headers = await withCsrfHeader(
      "POST",
      { Accept: "text/event-stream", "Content-Type": "application/json" },
      path,
    );
    const response = await fetch(`${API_BASE_URL}${path}`, {
      method: "POST",
      headers,
      credentials: "include",
      signal: options.signal,
      body: JSON.stringify({
        agentId: payload.agentId ?? "",
        draft: saveAgentDefinitionRequestSchema.parse(payload.draft),
        prompt: payload.prompt,
      }),
    });
    if (!response.ok || !response.body) {
      throw new AssistantStreamError(await streamFailureMessage(response), response.status);
    }

    await readEventStream(
      response.body,
      (message) => onMessage(message.event, message.data),
      options.signal,
    );
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
