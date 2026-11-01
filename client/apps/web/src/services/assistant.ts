import {
  AssistantActiveTurnDocument,
  AssistantArtifactBySlugDocument,
  AssistantArtifactLineageDocument,
  AssistantArtifactRowsDocument,
  AssistantArtifactsDocument,
  AssistantCaseDocument,
  AssistantChaptersDocument,
  AssistantCommandAvailabilityDocument,
  AssistantLiveTurnsDocument,
  AssistantMentionPageDocument,
  AssistantMentionsDocument,
  AssistantMessagesDocument,
  AssistantProvidersDocument,
  AssistantQueueDocument,
  AssistantSchedulesDocument,
  AssistantThreadBudgetDocument,
  AssistantThreadDocument,
  AssistantThreadPlansDocument,
  AssistantThreadProposalsDocument,
  AssistantThreadSchedulesDocument,
  AssistantThreadShelvesDocument,
  AssistantThreadsDocument,
  AssistantWaitsDocument,
  AwaitAssistantCaseReplyDocument,
  BindAssistantCaseDocument,
  CancelAssistantWaitDocument,
  CompactAssistantThreadDocument,
  CreateAssistantScheduleDocument,
  DeleteAssistantScheduleDocument,
  DeleteAssistantThreadDocument,
  DeskSearchDocument,
  EditAssistantQueuedMessageDocument,
  EnqueueAssistantMessageDocument,
  HandOffAssistantThreadDocument,
  MarkAssistantThreadReadDocument,
  OpenAssistantChapterDocument,
  PinAssistantArtifactDocument,
  RemoveAssistantChapterDocument,
  RemoveAssistantQueuedMessageDocument,
  RenameAssistantChapterDocument,
  ReorderAssistantQueueDocument,
  RequestAssistantAllowanceDocument,
  RestoreAssistantDocumentVersionDocument,
  RewriteAssistantDocumentDocument,
  RunAssistantArtifactReportDocument,
  RunAssistantScheduleDocument,
  SaveAssistantDocumentVersionDocument,
  SaveAssistantProposalEditsDocument,
  SendAssistantQueuedMessageDocument,
  SnoozeAssistantCaseDocument,
  StartAssistantThreadDocument,
  StartAssistantTurnDocument,
  StopAssistantTurnDocument,
  TickAssistantCaseItemDocument,
  UnbindAssistantCaseDocument,
  UpdateAssistantScheduleDocument,
  UpdateAssistantThreadDocument,
  WakeAssistantCaseDocument,
  type AssistantMessageInput,
  type AssistantPageContextInput,
  type UpdateAssistantScheduleInput,
} from "@trenova/graphql/generated/graphql";
import { api, withSessionHeaders } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { safeParse } from "@trenova/shared/lib/parse";
import { readEventStream } from "@trenova/shared/lib/sse";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
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
  mentionPageSchema,
  deskSearchResultListSchema,
  assistantChapterListSchema,
  assistantChapterSchema,
  threadBudgetSchema,
  commandAvailabilitySchema,
  type DeskSearchKind,
  type MentionPageKind,
  type MentionSearchType,
  assistantArtifactListSchema,
  assistantArtifactPageSchema,
  artifactRowsSchema,
  assistantArtifactSchema,
  assistantLiveTurnListSchema,
  agentBudgetStatusSchema,
  assistantPlanListSchema,
  assistantProposalListSchema,
  proposalEditsSchema,
  assistantProviderListSchema,
  assistantThreadPageSchema,
  assistantThreadShelfCountsSchema,
  assistantThreadSchema,
  conversationScheduleListSchema,
  queuedMessageListSchema,
  agentWaitListSchema,
  agentWaitSchema,
  caseBindingSchema,
  caseViewSchema,
  queuedMessageSchema,
  conversationScheduleSchema,
  createdScheduleSchema,
  handoffResultSchema,
  parseAskStreamEvent,
  parseAssistantStreamEvent,
  saveAgentDefinitionRequestSchema,
  type AgentDefinition,
  type AssistantArtifact,
  type AssistantEntityRef,
  type AssistantPageContext,
  type AssistantSurface,
  type ThreadOrigin,
  type AssistantStreamEvent,
  type AssistantThread,
  type CaseParty,
  type CaseSubjectType,
  type ConversationSchedule,
  type QueuedMessage,
  type SaveAgentDefinitionRequest,
  type SnoozeAnchor,
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

/**
 * What narrows the person's conversations to one shelf, on the server: pinned
 * or not, and the instants, in Unix seconds, between which each was last
 * touched (its last message, or its creation when it has none). Either end of
 * the range may be left open.
 */
export type ThreadListFilter = {
  pinned?: boolean;
  touchedFrom?: number;
  touchedBefore?: number;
};

/** What narrows a page of a conversation's artifacts, on the server. */
export type ArtifactListParams = {
  cursor?: string;
  limit?: number;
  q?: string;
  kind?: string;
  pinned?: boolean;
  /** Each payload cut to the keys that name and draw an artifact in a list. */
  summary?: boolean;
};

/** Which page of a table's list to read, and how it is searched and sorted. */
export type ArtifactRowsRequest = {
  limit: number;
  after?: string;
  offset?: number;
  q?: string;
  /** A column of the table, which the server maps to the field the list sorts by. */
  sortBy?: string;
  sortDirection?: "asc" | "desc";
  /** Ask for how many rows the list holds in all, alongside the page. */
  total?: boolean;
};

/**
 * What names one reading of a table's list: the version of the table it
 * continues, the page size, the search and the sort, and whether it reads the
 * list from its start rather than from where the agent stopped.
 */
export type ArtifactRowsKey = {
  stamp: number;
  size: number;
  q: string;
  sortBy: string;
  sortDirection: "asc" | "desc";
  fromStart: boolean;
  /** When the person asked to read the list again; zero for the table as it was kept. */
  readAt: number;
};

export function downloadAssistantTranscript(threadId: AssistantThread["id"]): void {
  downloadFromUrl(assistantTranscriptUrl(threadId));
}

/** A turn handed to a worker, and where to watch it. */
const startedTurnSchema = z.object({
  turnId: z.string(),
  threadId: z.string(),
  streamUrl: z.string(),
  status: z.string(),
});

/** A palette question handed to a worker, and where its answer streams. */
const startedAskSchema = z.object({
  askId: z.string(),
  streamUrl: z.string(),
});

export type StartedAsk = z.infer<typeof startedAskSchema>;

/**
 * The Desk conversation a palette question was carried into. `unanswered` is
 * the question when there was no answer to carry, for the Desk to ask.
 */
const keptAskSchema = z.object({
  thread: assistantThreadSchema,
  unanswered: z.string().optional().default(""),
});

export type KeptAsk = z.infer<typeof keptAskSchema>;

export type StartedTurn = z.infer<typeof startedTurnSchema>;

/**
 * What became of a message handed to the queue: still waiting (with why, when
 * the conversation was free but could not take it), read into the reply under
 * way, or sent as a reply of its own.
 */
const queueOutcomeSchema = z.object({
  item: queuedMessageSchema.nullish(),
  steering: z.boolean().optional().default(false),
  held: z.string().optional().default(""),
  turn: startedTurnSchema.nullish(),
});

export type QueueOutcome = z.infer<typeof queueOutcomeSchema>;

/** A message for the queue: the words, and what rides with them. */
export type EnqueueOptions = SendMessageOptions & {
  /** Read it into the reply under way at its next step, rather than after it. */
  steer?: boolean;
};

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
  origin?: "Person" | "DecisionFollowUp" | "Scheduled" | "Compaction" | "WaitResolved";
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
  surface?: AssistantSurface;
  providerId?: string;
  /** Documents uploaded to this thread for this message. */
  attachmentDocumentIds?: readonly string[];
  /** Records named from the composer. */
  mentions?: readonly AssistantEntityRef[];
  /**
   * Hands the message to another agent the person may use, run from this
   * conversation in place of its own agent's reply.
   */
  directedAgentId?: string;
};

/**
 * A quick question from the command palette: no thread yet, the answer makes
 * one. The route is the palette's, so the server names the surface itself.
 */
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
  /** Whether the conversation compacts itself on nearing a full context. */
  autoCompact?: boolean;
};

export class AssistantService {
  /**
   * One page of the person's conversations; the cursor is the previous page's
   * `nextCursor`. Given `until` (a page's own `nextCursor`), it reads that
   * page's range again instead: everything from the cursor down to the place
   * `until` marks, however many conversations sit there now.
   */
  public async listThreads({
    limit,
    cursor,
    until,
    filter,
    signal,
  }: {
    limit?: number;
    cursor?: string;
    until?: string;
    filter?: ThreadListFilter;
    signal?: AbortSignal;
  } = {}) {
    const response = await requestGraphQL({
      document: AssistantThreadsDocument,
      operationName: "AssistantThreads",
      variables: {
        limit: limit ?? null,
        cursor: cursor || null,
        until: until || null,
        pinned: filter?.pinned ?? null,
        touchedFrom: filter?.touchedFrom ?? null,
        touchedBefore: filter?.touchedBefore ?? null,
      },
      signal,
    });
    return safeParse(assistantThreadPageSchema, response.assistantThreads, "Assistant Thread");
  }

  /** How many conversations each shelf holds, banded by the edges given, newest first. */
  public async countThreadShelves(
    edges: readonly number[],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantThreadShelvesDocument,
      operationName: "AssistantThreadShelves",
      variables: { edges: [...edges] },
      signal,
    });
    return safeParse(
      assistantThreadShelfCountsSchema,
      response.assistantThreadShelves,
      "Assistant Thread Shelves",
    );
  }

  public async startThread(agentDefinitionId: string, options: StartThreadOptions = {}) {
    const response = await requestGraphQL({
      document: StartAssistantThreadDocument,
      operationName: "StartAssistantThread",
      variables: {
        input: {
          agentDefinitionId,
          title: options.title ?? "",
          origin: options.origin ?? "Panel",
          subjectType: options.subjectType ?? "",
          subjectId: options.subjectId || null,
        },
      },
    });
    return safeParse(assistantThreadSchema, response.startAssistantThread, "Assistant Thread");
  }

  public async updateThread(id: AssistantThread["id"], options: UpdateThreadOptions) {
    const response = await requestGraphQL({
      document: UpdateAssistantThreadDocument,
      operationName: "UpdateAssistantThread",
      variables: {
        id,
        input: {
          title: options.title ?? null,
          pinned: options.pinned ?? null,
          pinnedFacts: options.pinnedFacts ? [...options.pinnedFacts] : null,
          autoCompact: options.autoCompact ?? null,
        },
      },
    });
    return safeParse(assistantThreadSchema, response.updateAssistantThread, "Assistant Thread");
  }

  /**
   * Starts summarizing the older part of a conversation to free its context.
   * It runs as the conversation's turn, followed on its stream and stopped
   * like a reply.
   */
  public async compactThread(threadId: AssistantThread["id"]): Promise<StartedTurn> {
    const response = await requestGraphQL({
      document: CompactAssistantThreadDocument,
      operationName: "CompactAssistantThread",
      variables: { id: threadId },
    });
    return response.compactAssistantThread;
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
    const response = await requestGraphQL({
      document: AssistantArtifactsDocument,
      operationName: "AssistantArtifacts",
      variables: {
        threadId,
        limit: options?.limit || null,
        cursor: options?.cursor || null,
        q: options?.q || null,
        kind: options?.kind || null,
        pinned: options?.pinned ? true : null,
        summary: options?.summary ? true : null,
      },
      signal: options?.signal,
    });
    return safeParse(
      assistantArtifactPageSchema,
      response.assistantArtifacts,
      "Assistant Artifact",
    );
  }

  /**
   * One page of a table's list, read again from its source with the person's
   * own access, after a cursor or at an offset, searched or sorted.
   */
  public async artifactRows(
    threadId: AssistantThread["id"],
    artifactId: string,
    read: ArtifactRowsRequest,
    options?: { signal?: AbortSignal },
  ) {
    const response = await requestGraphQL({
      document: AssistantArtifactRowsDocument,
      operationName: "AssistantArtifactRows",
      variables: {
        threadId,
        artifactId,
        input: {
          limit: read.limit,
          after: read.after || null,
          offset: read.offset || null,
          q: read.q || null,
          sortBy: read.sortBy || null,
          sortDirection: read.sortBy ? (read.sortDirection ?? "asc") : null,
          total: read.total ? true : null,
        },
      },
      signal: options?.signal,
    });
    return safeParse(artifactRowsSchema, response.assistantArtifactRows, "Artifact Rows");
  }

  /** Every version of the lineage an artifact belongs to, oldest first. */
  public async artifactLineage(
    threadId: AssistantThread["id"],
    artifactId: string,
    options?: { signal?: AbortSignal },
  ) {
    const response = await requestGraphQL({
      document: AssistantArtifactLineageDocument,
      operationName: "AssistantArtifactLineage",
      variables: { threadId, artifactId },
      signal: options?.signal,
    });
    return safeParse(
      assistantArtifactListSchema,
      response.assistantArtifactLineage,
      "Assistant Artifact",
    );
  }

  /** The lineage a link names by its slug. */
  public async artifactBySlug(
    threadId: AssistantThread["id"],
    slug: string,
    options?: { signal?: AbortSignal },
  ) {
    const response = await requestGraphQL({
      document: AssistantArtifactBySlugDocument,
      operationName: "AssistantArtifactBySlug",
      variables: { threadId, slug },
      signal: options?.signal,
    });
    return safeParse(
      assistantArtifactListSchema,
      response.assistantArtifactBySlug,
      "Assistant Artifact",
    );
  }

  public async saveDocumentVersion(
    threadId: AssistantThread["id"],
    artifactId: string,
    body: { body: string; note: string },
  ) {
    const response = await requestGraphQL({
      document: SaveAssistantDocumentVersionDocument,
      operationName: "SaveAssistantDocumentVersion",
      variables: { threadId, artifactId, body: body.body, note: body.note },
    });
    return safeParse(
      assistantArtifactSchema,
      response.saveAssistantDocumentVersion,
      "Assistant Artifact",
    );
  }

  public async restoreDocumentVersion(threadId: AssistantThread["id"], artifactId: string) {
    const response = await requestGraphQL({
      document: RestoreAssistantDocumentVersionDocument,
      operationName: "RestoreAssistantDocumentVersion",
      variables: { threadId, artifactId },
    });
    return safeParse(
      assistantArtifactSchema,
      response.restoreAssistantDocumentVersion,
      "Assistant Artifact",
    );
  }

  /** Runs the report a report run or preview shows, kept as the artifact's next version. */
  public async runArtifactReport(threadId: AssistantThread["id"], artifactId: string) {
    const response = await requestGraphQL({
      document: RunAssistantArtifactReportDocument,
      operationName: "RunAssistantArtifactReport",
      variables: { threadId, artifactId },
    });
    return safeParse(
      assistantArtifactSchema,
      response.runAssistantArtifactReport,
      "Assistant Artifact",
    );
  }

  public async rewriteDocument(
    threadId: AssistantThread["id"],
    artifactId: string,
    body: { text: string; mode: "shorter" | "plain" | "ask"; prompt: string },
  ) {
    const response = await requestGraphQL({
      document: RewriteAssistantDocumentDocument,
      operationName: "RewriteAssistantDocument",
      variables: { threadId, artifactId, ...body },
    });
    return { text: response.rewriteAssistantDocument };
  }

  public async pinArtifact(
    threadId: AssistantThread["id"],
    artifactId: AssistantArtifact["id"],
    pinned: boolean,
  ) {
    const response = await requestGraphQL({
      document: PinAssistantArtifactDocument,
      operationName: "PinAssistantArtifact",
      variables: { threadId, artifactId, pinned },
    });
    return safeParse(assistantArtifactSchema, response.pinAssistantArtifact, "Assistant Artifact");
  }

  /**
   * The requests scheduled in one conversation, newest first. A conversation
   * keeps at most 25, so the first page is all of them.
   */
  public async listThreadSchedules(
    threadId: AssistantThread["id"],
    options?: { signal?: AbortSignal; limit?: number; offset?: number },
  ) {
    const response = await requestGraphQL({
      document: AssistantThreadSchedulesDocument,
      operationName: "AssistantThreadSchedules",
      variables: { threadId, limit: options?.limit ?? 25, offset: options?.offset ?? 0 },
      signal: options?.signal,
    });
    return safeParse(
      conversationScheduleListSchema,
      response.assistantThreadSchedules,
      "Conversation Schedule",
    );
  }

  /**
   * Every request the person scheduled, across their conversations, with each
   * one's conversation title: the ones that run soonest first, paused ones after.
   */
  public async listSchedules(options?: { signal?: AbortSignal; limit?: number; offset?: number }) {
    const response = await requestGraphQL({
      document: AssistantSchedulesDocument,
      operationName: "AssistantSchedules",
      variables: { limit: options?.limit ?? 25, offset: options?.offset ?? 0 },
      signal: options?.signal,
    });
    return safeParse(
      conversationScheduleListSchema,
      response.assistantSchedules,
      "Conversation Schedule",
    );
  }

  /**
   * Schedules a message: "every weekday at 7:30am, …" or "/schedule …". The
   * server reads the cadence and answers with the schedule and its card.
   */
  public async createSchedule(threadId: AssistantThread["id"], content: string) {
    const response = await requestGraphQL({
      document: CreateAssistantScheduleDocument,
      operationName: "CreateAssistantSchedule",
      variables: { threadId, content },
    });
    return safeParse(
      createdScheduleSchema,
      response.createAssistantSchedule,
      "Conversation Schedule",
    );
  }

  /**
   * Changes what a schedule asks, how often and when: `repeat` is "day",
   * "weekday" or a day of the week, `time` a 24-hour "HH:MM" on the
   * schedule's own clock. The server answers with the schedule as saved.
   */
  public async editSchedule(
    id: ConversationSchedule["id"],
    edit: { prompt: string; repeat: string; time: string },
  ) {
    return this.updateSchedule(id, edit);
  }

  /** Pauses or resumes a schedule. */
  public async setScheduleEnabled(id: ConversationSchedule["id"], enabled: boolean) {
    return this.updateSchedule(id, { enabled });
  }

  public async deleteSchedule(id: ConversationSchedule["id"]) {
    await requestGraphQL({
      document: DeleteAssistantScheduleDocument,
      operationName: "DeleteAssistantSchedule",
      variables: { id },
    });
  }

  /** Asks a schedule's request now; answers with the turn to watch. */
  public async runSchedule(id: ConversationSchedule["id"]): Promise<StartedTurn> {
    const response = await requestGraphQL({
      document: RunAssistantScheduleDocument,
      operationName: "RunAssistantSchedule",
      variables: { id },
    });
    return response.runAssistantSchedule;
  }

  /** What the person left for the conversation, in the order it will be sent. */
  public async listQueue(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantQueueDocument,
      operationName: "AssistantQueue",
      variables: { threadId },
      signal,
    });
    return safeParse(queuedMessageListSchema, response.assistantQueue, "Queued Message");
  }

  /**
   * Leaves a message for the conversation. With steer, the reply under way
   * reads it at its next step; otherwise it is sent once the reply ends. A
   * conversation with nothing under way sends it at once, and the outcome
   * carries the turn to follow.
   */
  public async enqueue(
    threadId: AssistantThread["id"],
    content: string,
    options: EnqueueOptions = {},
  ): Promise<QueueOutcome> {
    const response = await requestGraphQL({
      document: EnqueueAssistantMessageDocument,
      operationName: "EnqueueAssistantMessage",
      variables: {
        threadId,
        input: { ...messageInput(content, options), steer: options.steer ?? false },
      },
    });
    return safeParse(queueOutcomeSchema, response.enqueueAssistantMessage, "Queued Message");
  }

  public async editQueued(
    threadId: AssistantThread["id"],
    item: Pick<QueuedMessage, "id" | "version">,
    content: string,
  ) {
    const response = await requestGraphQL({
      document: EditAssistantQueuedMessageDocument,
      operationName: "EditAssistantQueuedMessage",
      variables: { threadId, itemId: item.id, content, version: item.version },
    });
    return safeParse(queuedMessageSchema, response.editAssistantQueuedMessage, "Queued Message");
  }

  public async removeQueued(threadId: AssistantThread["id"], id: QueuedMessage["id"]) {
    await requestGraphQL({
      document: RemoveAssistantQueuedMessageDocument,
      operationName: "RemoveAssistantQueuedMessage",
      variables: { threadId, itemId: id },
    });
  }

  /** Puts the waiting messages in this order; every one of them is named once. */
  public async reorderQueue(threadId: AssistantThread["id"], ids: readonly string[]) {
    const response = await requestGraphQL({
      document: ReorderAssistantQueueDocument,
      operationName: "ReorderAssistantQueue",
      variables: { threadId, ids: [...ids] },
    });
    return safeParse(queuedMessageListSchema, response.reorderAssistantQueue, "Queued Message");
  }

  /**
   * Sends a waiting message now: into the reply under way when there is one,
   * or as a reply of its own ahead of the rest of the queue.
   */
  public async sendQueued(threadId: AssistantThread["id"], id: QueuedMessage["id"]) {
    const response = await requestGraphQL({
      document: SendAssistantQueuedMessageDocument,
      operationName: "SendAssistantQueuedMessage",
      variables: { threadId, itemId: id },
    });
    return safeParse(queueOutcomeSchema, response.sendAssistantQueuedMessage, "Queued Message");
  }

  /** What the conversation's agent is waiting on, newest first. */
  public async listWaits(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantWaitsDocument,
      operationName: "AssistantWaits",
      variables: { threadId },
      signal,
    });
    return safeParse(agentWaitListSchema, response.assistantWaits, "Agent Wait");
  }

  /** Ends a wait without the agent picking the work up. */
  public async cancelWait(threadId: AssistantThread["id"], waitId: string) {
    const response = await requestGraphQL({
      document: CancelAssistantWaitDocument,
      operationName: "CancelAssistantWait",
      variables: { threadId, waitId },
    });
    return safeParse(agentWaitSchema, response.cancelAssistantWait, "Agent Wait");
  }

  /** The conversation's chapters, oldest first. */
  public async listChapters(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantChaptersDocument,
      operationName: "AssistantChapters",
      variables: { threadId },
      signal,
    });
    const list = await safeParse(
      assistantChapterListSchema,
      response.assistantChapters,
      "Assistant Chapter",
    );
    return list.items;
  }

  /** Starts a chapter at the turn the message belongs to. */
  public async openChapter(threadId: AssistantThread["id"], messageId: string) {
    const response = await requestGraphQL({
      document: OpenAssistantChapterDocument,
      operationName: "OpenAssistantChapter",
      variables: { threadId, messageId },
    });
    return safeParse(assistantChapterSchema, response.openAssistantChapter, "Assistant Chapter");
  }

  /** Names a chapter; the title is the person's from then on. */
  public async renameChapter(
    threadId: AssistantThread["id"],
    chapterId: string,
    title: string,
    version: number,
  ) {
    const response = await requestGraphQL({
      document: RenameAssistantChapterDocument,
      operationName: "RenameAssistantChapter",
      variables: { threadId, chapterId, title, version },
    });
    return safeParse(assistantChapterSchema, response.renameAssistantChapter, "Assistant Chapter");
  }

  /** Ends a chapter where it starts, so what it held reads as part of the one before. */
  public async removeChapter(threadId: AssistantThread["id"], chapterId: string) {
    await requestGraphQL({
      document: RemoveAssistantChapterDocument,
      operationName: "RemoveAssistantChapter",
      variables: { threadId, chapterId },
    });
  }

  /** Where the conversation's case stands, what stands between its record and what comes next. */
  public async getCase(threadId: AssistantThread["id"], { signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: AssistantCaseDocument,
      operationName: "AssistantCase",
      variables: { threadId },
      signal,
    });
    return safeParse(caseViewSchema, response.assistantCase, "Case");
  }

  /** Makes the conversation a case about a shipment, invoice or dispute. */
  public async bindCase(
    threadId: AssistantThread["id"],
    subject: { subjectType: CaseSubjectType; subjectId: string },
  ) {
    const response = await requestGraphQL({
      document: BindAssistantCaseDocument,
      operationName: "BindAssistantCase",
      variables: { threadId, subjectType: subject.subjectType, subjectId: subject.subjectId },
    });
    return safeParse(caseBindingSchema, response.bindAssistantCase, "Case");
  }

  /** Makes the case an ordinary conversation again. */
  public async unbindCase(threadId: AssistantThread["id"]) {
    const response = await requestGraphQL({
      document: UnbindAssistantCaseDocument,
      operationName: "UnbindAssistantCase",
      variables: { threadId },
    });
    return safeParse(caseBindingSchema, response.unbindAssistantCase, "Case");
  }

  /**
   * Puts the case out of the way until a time, the shipment's next
   * appointment or its ETA. `until` is the time, or the fallback for an ETA
   * the shipment does not have yet.
   */
  public async snoozeCase(
    threadId: AssistantThread["id"],
    snooze: { anchor: SnoozeAnchor; until?: number },
  ) {
    const response = await requestGraphQL({
      document: SnoozeAssistantCaseDocument,
      operationName: "SnoozeAssistantCase",
      variables: { threadId, anchor: snooze.anchor, until: snooze.until ?? null },
    });
    return safeParse(caseBindingSchema, response.snoozeAssistantCase, "Case");
  }

  public async wakeCase(threadId: AssistantThread["id"]) {
    const response = await requestGraphQL({
      document: WakeAssistantCaseDocument,
      operationName: "WakeAssistantCase",
      variables: { threadId },
    });
    return safeParse(caseBindingSchema, response.wakeAssistantCase, "Case");
  }

  /** Ticks, or unticks, a step the organization added for a person to tick. */
  public async tickCaseItem(threadId: AssistantThread["id"], itemKey: string, ticked: boolean) {
    const response = await requestGraphQL({
      document: TickAssistantCaseItemDocument,
      operationName: "TickAssistantCaseItem",
      variables: { threadId, itemKey, ticked },
    });
    return safeParse(caseViewSchema, response.tickAssistantCaseItem, "Case");
  }

  /** Parks the case until the customer or one of the carriers replies. */
  public async awaitCaseReply(
    threadId: AssistantThread["id"],
    party: { party: CaseParty["kind"]; partyId: string; giveUpAfterHours?: number },
  ) {
    const response = await requestGraphQL({
      document: AwaitAssistantCaseReplyDocument,
      operationName: "AwaitAssistantCaseReply",
      variables: {
        threadId,
        party: party.party,
        partyId: party.partyId,
        giveUpAfterHours: party.giveUpAfterHours ?? null,
      },
    });
    return safeParse(agentWaitSchema, response.awaitAssistantCaseReply, "Agent Wait");
  }

  public async getThread(id: AssistantThread["id"], { signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: AssistantThreadDocument,
      operationName: "AssistantThread",
      variables: { id },
      signal,
    });
    return safeParse(assistantThreadSchema, response.assistantThread, "Assistant Thread");
  }

  /** How close the thread's agent, and the person asking, are to their caps. */
  public async threadBudget(id: AssistantThread["id"], { signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: AssistantThreadBudgetDocument,
      operationName: "AssistantThreadBudget",
      variables: { threadId: id },
      signal,
    });
    return safeParse(threadBudgetSchema, response.assistantThreadBudget, "Assistant Thread Budget");
  }

  /** The extension commands (/browser, /search) the person can send this agent. */
  public async commandAvailability(agentId: string, { signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: AssistantCommandAvailabilityDocument,
      operationName: "AssistantCommandAvailability",
      variables: { agentId },
      signal,
    });
    return safeParse(
      commandAvailabilitySchema,
      response.assistantCommandAvailability,
      "Command Availability",
    );
  }

  /** Asks the people who run AI Control for what the person ran out of. */
  public async requestMore(
    id: AssistantThread["id"],
    kind: "access" | "allowance" | "budget" | "daily_runs",
  ) {
    const response = await requestGraphQL({
      document: RequestAssistantAllowanceDocument,
      operationName: "RequestAssistantAllowance",
      variables: { threadId: id, kind },
    });
    return { sent: response.requestAssistantAllowance };
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
    chapterId?: string,
  ) {
    const response = await requestGraphQL({
      document: HandOffAssistantThreadDocument,
      operationName: "HandOffAssistantThread",
      variables: {
        id,
        input: { agentDefinitionId, facts: [...facts], chapterId: chapterId || null },
      },
    });
    return safeParse(handoffResultSchema, response.handOffAssistantThread, "Assistant Handoff");
  }

  public async deleteThread(id: AssistantThread["id"]) {
    await requestGraphQL({
      document: DeleteAssistantThreadDocument,
      operationName: "DeleteAssistantThread",
      variables: { id },
    });
  }

  /** Marks everything in the conversation as seen by its owner. */
  public async markThreadRead(id: AssistantThread["id"]) {
    await requestGraphQL({
      document: MarkAssistantThreadReadDocument,
      operationName: "MarkAssistantThreadRead",
      variables: { id },
    });
  }

  /**
   * One page of a thread: the newest `limit` messages, or with `before` the
   * page above the message carrying that sequence.
   */
  public async listMessages(
    threadId: AssistantThread["id"],
    { limit, before, signal }: { limit?: number; before?: number; signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantMessagesDocument,
      operationName: "AssistantMessages",
      variables: { threadId, limit: limit ?? null, before: before ?? null },
      signal,
    });
    return safeParse(assistantMessagePageSchema, response.assistantMessages, "Assistant Message");
  }

  /**
   * Records a person can name with @ from the composer, of one kind or of
   * every kind they may read, matched on what the record is called.
   */
  public async searchMentions(
    query: string,
    type: MentionSearchType,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantMentionsDocument,
      operationName: "AssistantMentions",
      variables: { query, type },
      signal,
    });
    const parsed = await safeParse(
      mentionCandidateListSchema,
      response.assistantMentions,
      "Assistant Mentions",
    );

    return parsed.results;
  }

  /** One page of the records of one kind, for a list that scrolls through them all. */
  public async searchMentionPage(
    query: string,
    kind: MentionPageKind,
    { offset, limit, signal }: { offset: number; limit: number; signal?: AbortSignal },
  ) {
    const response = await requestGraphQL({
      document: AssistantMentionPageDocument,
      operationName: "AssistantMentionPage",
      variables: { query, kind, offset, limit },
      signal,
    });

    return safeParse(mentionPageSchema, response.assistantMentionPage, "Assistant Mentions");
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
    const response = await requestGraphQL({
      document: DeskSearchDocument,
      operationName: "DeskSearch",
      variables: { query, kind },
      signal,
    });
    const parsed = await safeParse(deskSearchResultListSchema, response.deskSearch, "Desk Search");

    return parsed.results;
  }

  /** The models this organization has made available to the assistant. */
  public async listProviders({ signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: AssistantProvidersDocument,
      operationName: "AssistantProviders",
      signal,
    });
    // safeParse resolves a promise, so the await belongs here rather than on
    // the caller: reading .results off the promise itself yields undefined, the
    // query stores undefined, and the picker quietly renders nothing.
    const parsed = await safeParse(
      assistantProviderListSchema,
      response.assistantProviders,
      "Assistant Providers",
    );

    return parsed.results;
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
    const response = await requestGraphQL({
      document: StartAssistantTurnDocument,
      operationName: "StartAssistantTurn",
      variables: { threadId, input: messageInput(content, options) },
      signal,
    });

    return response.startAssistantTurn;
  }

  /**
   * A quick question from the palette. Nova answers it on a worker without
   * making a conversation; its answer streams from `attachAsk`.
   */
  public async startAsk(
    content: string,
    options: AskOptions = {},
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<StartedAsk> {
    const response = await api.post(
      "/assistant/ask/",
      {
        content,
        context: options.context ?? null,
        mentions: options.mentions ?? [],
      },
      { signal },
    );

    return safeParse(startedAskSchema, response, "Palette Question");
  }

  /** Follows a palette question's answer from where the reader left off. */
  public async attachAsk(
    askId: string,
    onEvent: (event: AssistantStreamEvent, cursor: string) => void,
    options: { cursor?: string; signal?: AbortSignal } = {},
  ): Promise<void> {
    await this.attachStream(
      `/assistant/ask/${askId}/stream/`,
      parseAskStreamEvent,
      onEvent,
      options,
    );
  }

  /** Takes back a palette question nobody is waiting for. */
  public async stopAsk(askId: string): Promise<void> {
    await api.delete(`/assistant/ask/${askId}/`);
  }

  /** Carries a palette question and its answer into a new Desk conversation. */
  public async keepAsk(askId: string): Promise<KeptAsk> {
    const response = await api.post(`/assistant/ask/${askId}/keep/`, {});
    return safeParse(keptAskSchema, response, "Kept Question");
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
    await this.attachStream(
      `/assistant/turns/${turnId}/stream/`,
      parseAssistantStreamEvent,
      onEvent,
      options,
    );
  }

  private async attachStream(
    path: string,
    parse: (event: string, raw: string) => AssistantStreamEvent | null,
    onEvent: (event: AssistantStreamEvent, cursor: string) => void,
    options: { cursor?: string; signal?: AbortSignal },
  ): Promise<void> {
    const headers = await withSessionHeaders("GET", { Accept: "text/event-stream" }, path);
    if (options.cursor) {
      headers.set("Last-Event-ID", options.cursor);
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
        const event = parse(message.event, message.data);
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
    await requestGraphQL({
      document: StopAssistantTurnDocument,
      operationName: "StopAssistantTurn",
      variables: { turnId },
    });
  }

  /**
   * Every reply this person's conversations are still producing. A reply runs
   * on a worker until it ends or is stopped, whether or not anything is
   * reading it, so this is the one place that knows what is under way when
   * the panel is closed or the conversation was left.
   */
  public async listActiveTurns(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: AssistantLiveTurnsDocument,
      operationName: "AssistantLiveTurns",
      signal: options?.signal,
    });
    return safeParse(
      assistantLiveTurnListSchema,
      { items: response.assistantLiveTurns },
      "Assistant Turns",
    );
  }

  /** The reply a conversation is still producing, if it is producing one. */
  public async activeTurn(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<ActiveTurn | null> {
    const response = await requestGraphQL({
      document: AssistantActiveTurnDocument,
      operationName: "AssistantActiveTurn",
      variables: { threadId },
      signal,
    });
    const turn = response.assistantActiveTurn;
    if (typeof turn !== "object" || turn === null) {
      return null;
    }

    return turn as ActiveTurn;
  }

  /**
   * Proposals outlive the turn that raised them, so reopening a thread has to
   * fetch them rather than rely on the send response.
   */
  public async listProposals(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantThreadProposalsDocument,
      operationName: "AssistantThreadProposals",
      variables: { threadId },
      signal,
    });
    return safeParse(
      assistantProposalListSchema,
      response.assistantThreadProposals,
      "Assistant Proposal",
    );
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
    const response = await requestGraphQL({
      document: SaveAssistantProposalEditsDocument,
      operationName: "SaveAssistantProposalEdits",
      variables: { threadId, proposalId, modifications },
    });
    return safeParse(proposalEditsSchema, response.saveAssistantProposalEdits, "Proposal Edits");
  }

  /**
   * The plans a thread's runs formed: several writes decided as one. Steps are
   * the proposals carrying the plan's id, so both lists are read together.
   */
  public async listPlans(
    threadId: AssistantThread["id"],
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await requestGraphQL({
      document: AssistantThreadPlansDocument,
      operationName: "AssistantThreadPlans",
      variables: { threadId },
      signal,
    });
    return safeParse(assistantPlanListSchema, response.assistantThreadPlans, "Assistant Plan");
  }

  private async updateSchedule(
    id: ConversationSchedule["id"],
    input: UpdateAssistantScheduleInput,
  ) {
    const response = await requestGraphQL({
      document: UpdateAssistantScheduleDocument,
      operationName: "UpdateAssistantSchedule",
      variables: { id, input },
    });
    return safeParse(
      conversationScheduleSchema,
      response.updateAssistantSchedule,
      "Conversation Schedule",
    );
  }
}

function messageInput(content: string, options: SendMessageOptions): AssistantMessageInput {
  return {
    content,
    context: pageContextInput(options.context),
    surface: options.surface ?? "",
    providerId: options.providerId ?? "",
    attachmentDocumentIds: [...(options.attachmentDocumentIds ?? [])],
    mentions: (options.mentions ?? []).map((mention) => ({
      type: mention.type,
      id: mention.id,
      label: mention.label ?? "",
    })),
    directedAgentId: options.directedAgentId || null,
  };
}

/** The page under a message, and the unsaved work of a page-bound conversation's page. */
function pageContextInput(
  context: AssistantPageContext | null | undefined,
): AssistantPageContextInput | null {
  if (!context) {
    return null;
  }

  return {
    path: context.path,
    entityType: context.entityType ?? "",
    entityId: context.entityId ?? "",
    title: context.title ?? "",
    view: context.view ?? null,
    draft: context.draft ?? null,
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
    const headers = await withSessionHeaders(
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
