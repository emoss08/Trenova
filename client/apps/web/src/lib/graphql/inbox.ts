import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import {
  AssignInboundMessagesDocument,
  BlockInboundSenderDocument,
  CorrectInboundClassificationDocument,
  CreateInboundLoadDocument,
  CreateInboundMailboxDocument,
  CreateMyTriageViewDocument,
  CreateTriageReplyTemplateDocument,
  CreateTriageRuleDocument,
  DecideInboundStopsDocument,
  DeleteMyTriageViewDocument,
  DeleteTriageReplyTemplateDocument,
  DeleteTriageRuleDocument,
  DiscardInboundLoadDocument,
  DiscardInboundReplyDraftDocument,
  DraftInboundLoadDocument,
  DraftInboundManyDocument,
  DraftInboundReplyDocument,
  FileInboundAttachmentDocument,
  FileInboundManyDocument,
  FlagInboundFactDocument,
  InboundReplyAttachablesDocument,
  HandOffTriageDocument,
  InboundMailboxCredentialsFieldsFragmentDoc,
  InboundMailboxesDocument,
  InboundMailboxFieldsFragmentDoc,
  InboundMessageCountsDocument,
  InboundMessageDocument,
  InboundMessagesDocument,
  InboundSenderProfileDocument,
  InboundThreadDocument,
  LinkInboundMessageDocument,
  LogInboundExceptionDocument,
  MyTriageViewsDocument,
  ReadUnreadInboundMessagesDocument,
  RenameMyTriageViewDocument,
  RenderTriageReplyTemplateDocument,
  ReopenInboundConflictDocument,
  ReopenInboundMessagesDocument,
  ResolveInboundConflictDocument,
  ReviewInboundMessageDocument,
  ReviewInboundMessagesDocument,
  RotateInboundMailboxTokenDocument,
  SaveInboundReplyDraftDocument,
  SendInboundMailboxTestDocument,
  SendInboundReplyDocument,
  SetInboundMailboxApiKeyDocument,
  SetInboundMailboxSigningSecretDocument,
  SetTriageRuleEnabledDocument,
  SnoozeInboundMessagesDocument,
  StopWaitingForInboundReplyDocument,
  TriageAssigneesDocument,
  TriageBriefingDocument,
  TriageCustomerPeekDocument,
  TriageReplyTemplatesDocument,
  TriageResponseTargetsDocument,
  TriageRuleMatchesDocument,
  TriageRulePreviewDocument,
  TriageRulesDocument,
  TriageSenderDomainsDocument,
  TriageSettingsDocument,
  TriageShipmentPeekDocument,
  TriageStopTimesDocument,
  UnblockInboundSenderDocument,
  UnfileInboundAttachmentDocument,
  UnlinkInboundMessageDocument,
  UpdateInboundMailboxDocument,
  UpdateTriageReplyTemplateDocument,
  UpdateTriageResponseTargetsDocument,
  UpdateTriageRuleDocument,
  UpdateTriageSettingsDocument,
  WaitForInboundReplyDocument,
  WakeInboundMessageDocument,
  type BulkReviewInboundMessagesInput,
  type FileInboundAttachmentInput,
  type InboundActivityKind,
  type InboundClassification,
  type InboundLinkedRecord,
  type InboundMailboxCredentialsFieldsFragment,
  type InboundMailboxFieldsFragment,
  type InboundMailboxInput,
  type InboundMailboxStatus,
  type InboundMessageCountsQuery,
  type InboundMessageDetailFieldsFragment,
  type InboundMessageListFieldsFragment,
  type InboundMessagesInput,
  type InboundMessageStatus,
  type InboundProvider,
  type InboundReplyAttachableFieldsFragment,
  type InboundReplyAttachableSource,
  type InboundReplyFieldsFragment,
  type InboundReviewPolicy,
  type InboundSenderProfileQuery,
  type InboundSnoozeEvent,
  type InboundThreadQuery,
  type InboundSourceHealth,
  type LinkInboundMessageInput,
  type LogInboundExceptionInput,
  type ReviewInboundMessageInput,
  type SendInboundReplyInput,
  type SnoozeInboundMessagesInput,
  type TriageActionFieldsFragment,
  type TriageActionKind,
  type TriageAssigneesQuery,
  type TriageBriefingQuery,
  type TriageBulkResultFieldsFragment,
  type TriageConflictResolution,
  type TriageCustomerPeekQuery,
  type TriageExceptionKind,
  type TriageLane,
  type TriageLoadFieldInput,
  type TriageReplyPurpose,
  type TriageReplyTemplateFieldsFragment,
  type TriageReplyTemplateInput,
  type TriageResponseTargetInput,
  type TriageResponseTargetsQuery,
  type TriageRuleAction,
  type TriageRuleFieldsFragment,
  type TriageRuleInput,
  type TriageRulePreviewFieldsFragment,
  type TriageSelectionInput,
  type TriageSenderScope,
  type TriageSettingsInput,
  type TriageSettingsQuery,
  type TriageShipmentPeekFieldsFragment,
  type TriageSort,
  type TriageStopsDecision,
  type TriageStopTimesQuery,
  type TriageUserFieldsFragment,
  type TriageViewFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";

/*
 * getFragmentData does the runtime unmasking — it hands back the same object
 * with the mask marker gone — but its return type is still the masked
 * fragment, and the masked type says nothing about the fields the nested
 * fragments contribute. The cast is the type catching up with what the call
 * already did; it is written once, here, rather than at every call site.
 */
function unmask<T>(value: T): UnmaskFragments<T> {
  return value as UnmaskFragments<T>;
}

export type InboundMessageRow = UnmaskFragments<InboundMessageListFieldsFragment>;
export type InboundMessageDetail = UnmaskFragments<InboundMessageDetailFieldsFragment>;
export type InboundMailbox = UnmaskFragments<InboundMailboxFieldsFragment>;
export type InboundMailboxCredentials = UnmaskFragments<InboundMailboxCredentialsFieldsFragment>;
export type InboundShipmentRef = NonNullable<InboundMessageRow["matchedShipment"]>;
export type InboundAttachment = InboundMessageDetail["attachments"][number];
export type InboundSnooze = NonNullable<InboundMessageRow["snooze"]>;
export type InboundReply = UnmaskFragments<InboundReplyFieldsFragment>;
export type InboundReplyAttachment = InboundReply["attachments"][number];
export type InboundReplyAttachable = UnmaskFragments<InboundReplyAttachableFieldsFragment>;
export type { InboundReplyAttachableSource };

/** What a reply to a message can carry, and how much its files may add up to. */
export type InboundReplyAttachables = {
  documents: InboundReplyAttachable[];
  limitBytes: number;
};
export type InboundActivity = InboundMessageDetail["activity"][number];
export type TriageAction = UnmaskFragments<TriageActionFieldsFragment>;
export type TriageFact = InboundMessageDetail["facts"][number];
export type TriageConflict = InboundMessageDetail["conflicts"][number];
export type TriageLoadDraft = NonNullable<InboundMessageDetail["loadDraft"]>;
export type InboundSenderRole = InboundSenderProfile["role"];
export type TriageUser = UnmaskFragments<TriageUserFieldsFragment>;
export type TriageView = UnmaskFragments<TriageViewFieldsFragment>;
export type TriageRule = UnmaskFragments<TriageRuleFieldsFragment>;
export type TriageRulePreview = UnmaskFragments<TriageRulePreviewFieldsFragment>;
export type TriageReplyTemplate = UnmaskFragments<TriageReplyTemplateFieldsFragment>;
export type TriageShipmentPeek = UnmaskFragments<TriageShipmentPeekFieldsFragment>;
export type TriageBulkResult = UnmaskFragments<TriageBulkResultFieldsFragment>;
export type TriageBriefing = UnmaskFragments<TriageBriefingQuery["triageBriefing"]>;
export type TriageAssignee = UnmaskFragments<TriageAssigneesQuery["triageAssignees"][number]>;
export type TriageSettings = UnmaskFragments<TriageSettingsQuery["triageSettings"]>;
export type TriageResponseTarget = TriageResponseTargetsQuery["triageResponseTargets"][number];
export type TriageStopTimes = NonNullable<UnmaskFragments<TriageStopTimesQuery["triageStopTimes"]>>;
export type TriageCustomerPeek = UnmaskFragments<TriageCustomerPeekQuery["triageCustomerPeek"]>;
export type InboundSenderProfile = UnmaskFragments<
  InboundSenderProfileQuery["inboundSenderProfile"]
>;
export type InboundMessageCounts = InboundMessageCountsQuery["inboundMessageCounts"];

export type {
  FileInboundAttachmentInput,
  InboundActivityKind,
  InboundClassification,
  InboundLinkedRecord,
  InboundMailboxInput,
  InboundMailboxStatus,
  InboundMessageStatus,
  InboundProvider,
  InboundReviewPolicy,
  InboundSnoozeEvent,
  InboundSourceHealth,
  LinkInboundMessageInput,
  LogInboundExceptionInput,
  SendInboundReplyInput,
  SnoozeInboundMessagesInput,
  TriageActionKind,
  TriageConflictResolution,
  TriageExceptionKind,
  TriageLane,
  TriageLoadFieldInput,
  TriageReplyPurpose,
  TriageReplyTemplateInput,
  TriageResponseTargetInput,
  TriageRuleAction,
  TriageRuleInput,
  TriageSenderScope,
  TriageSettingsInput,
  TriageSort,
  TriageStopsDecision,
};

/** Every query wrapper takes the TanStack signal so an abandoned read is cancelled. */
export type RequestOptions = { signal?: AbortSignal };

/**
 * A write that must happen once even when it is retried: the reply that goes
 * to a partner, a whole folder settled at once, a load booked from a tender.
 * The key travels as the Idempotency-Key header (docs/engineering/idempotency.md).
 */
export type IdempotentOptions = { idempotencyKey?: string };

/** How much of a folder one request asks for. */
export const INBOX_PAGE_SIZE = 40;

/** One folder's position in the queue: every filter but the cursor. */
export type InboundMessageFilter = Omit<InboundMessagesInput, "after" | "first">;

export type InboundMessagePage = {
  messages: InboundMessageRow[];
  endCursor: string | null;
  hasNextPage: boolean;
};

export async function fetchInboundMessages(
  filter: InboundMessageFilter,
  after: string | null,
  options?: RequestOptions,
): Promise<InboundMessagePage> {
  const data = await requestGraphQL({
    document: InboundMessagesDocument,
    operationName: "InboundMessages",
    variables: { input: { ...filter, after, first: INBOX_PAGE_SIZE } },
    signal: options?.signal,
  });
  const connection = unmask(data.inboundMessages);

  return {
    messages: connection.edges.map((edge) => edge.node),
    endCursor: connection.pageInfo.endCursor ?? null,
    hasNextPage: connection.pageInfo.hasNextPage,
  };
}

export async function fetchInboundMessage(
  id: string,
  options?: RequestOptions,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: InboundMessageDocument,
    operationName: "InboundMessage",
    variables: { id },
    signal: options?.signal,
  });

  return unmask(data.inboundMessage);
}

export async function fetchInboundMessageCounts(
  options?: RequestOptions,
): Promise<InboundMessageCounts> {
  const data = await requestGraphQL({
    document: InboundMessageCountsDocument,
    operationName: "InboundMessageCounts",
    variables: {},
    signal: options?.signal,
  });

  return data.inboundMessageCounts;
}

export async function fetchInboundMailboxes(options?: RequestOptions): Promise<InboundMailbox[]> {
  const data = await requestGraphQL({
    document: InboundMailboxesDocument,
    operationName: "InboundMailboxes",
    variables: {},
    signal: options?.signal,
  });

  return data.inboundMailboxes.map((row) => mailbox(row));
}

export type InboundThreadMessage = UnmaskFragments<InboundThreadQuery["inboundThread"][number]>;

export async function fetchInboundThread(
  threadKey: string,
  options?: RequestOptions,
): Promise<InboundThreadMessage[]> {
  const data = await requestGraphQL({
    document: InboundThreadDocument,
    operationName: "InboundThread",
    variables: { threadKey },
    signal: options?.signal,
  });

  return unmask(data.inboundThread);
}

export async function fetchTriageBriefing(options?: RequestOptions): Promise<TriageBriefing> {
  const data = await requestGraphQL({
    document: TriageBriefingDocument,
    operationName: "TriageBriefing",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.triageBriefing);
}

export async function fetchMyTriageViews(options?: RequestOptions): Promise<TriageView[]> {
  const data = await requestGraphQL({
    document: MyTriageViewsDocument,
    operationName: "MyTriageViews",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.myTriageViews);
}

export async function fetchTriageResponseTargets(
  options?: RequestOptions,
): Promise<TriageResponseTarget[]> {
  const data = await requestGraphQL({
    document: TriageResponseTargetsDocument,
    operationName: "TriageResponseTargets",
    variables: {},
    signal: options?.signal,
  });

  return data.triageResponseTargets;
}

export async function fetchTriageSettings(options?: RequestOptions): Promise<TriageSettings> {
  const data = await requestGraphQL({
    document: TriageSettingsDocument,
    operationName: "TriageSettings",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.triageSettings);
}

export async function fetchTriageAssignees(options?: RequestOptions): Promise<TriageAssignee[]> {
  const data = await requestGraphQL({
    document: TriageAssigneesDocument,
    operationName: "TriageAssignees",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.triageAssignees);
}

export async function fetchTriageRules(options?: RequestOptions): Promise<TriageRule[]> {
  const data = await requestGraphQL({
    document: TriageRulesDocument,
    operationName: "TriageRules",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.triageRules);
}

export async function fetchTriageRulePreview(
  input: TriageRuleInput,
  options?: RequestOptions,
): Promise<TriageRulePreview> {
  const data = await requestGraphQL({
    document: TriageRulePreviewDocument,
    operationName: "TriageRulePreview",
    variables: { input },
    signal: options?.signal,
  });

  return unmask(data.triageRulePreview);
}

export async function fetchTriageRuleMatches(
  id: string,
  options?: RequestOptions,
): Promise<TriageRulePreview> {
  const data = await requestGraphQL({
    document: TriageRuleMatchesDocument,
    operationName: "TriageRuleMatches",
    variables: { id },
    signal: options?.signal,
  });

  return unmask(data.triageRuleMatches);
}

export async function fetchTriageSenderDomains(
  query: string,
  options?: RequestOptions,
): Promise<string[]> {
  const data = await requestGraphQL({
    document: TriageSenderDomainsDocument,
    operationName: "TriageSenderDomains",
    variables: { query: query.trim() === "" ? null : query.trim() },
    signal: options?.signal,
  });

  return data.triageSenderDomains;
}

export async function fetchInboundSenderProfile(
  messageId: string,
  options?: RequestOptions,
): Promise<InboundSenderProfile> {
  const data = await requestGraphQL({
    document: InboundSenderProfileDocument,
    operationName: "InboundSenderProfile",
    variables: { messageId },
    signal: options?.signal,
  });

  return unmask(data.inboundSenderProfile);
}

export async function fetchTriageReplyTemplates(
  options?: RequestOptions,
): Promise<TriageReplyTemplate[]> {
  const data = await requestGraphQL({
    document: TriageReplyTemplatesDocument,
    operationName: "TriageReplyTemplates",
    variables: {},
    signal: options?.signal,
  });

  return unmask(data.triageReplyTemplates);
}

export async function renderTriageReplyTemplate(
  templateId: string,
  messageId: string,
  options?: RequestOptions,
): Promise<string> {
  const data = await requestGraphQL({
    document: RenderTriageReplyTemplateDocument,
    operationName: "RenderTriageReplyTemplate",
    variables: { templateId, messageId },
    signal: options?.signal,
  });

  return data.renderTriageReplyTemplate;
}

export async function fetchTriageStopTimes(
  messageId: string,
  options?: RequestOptions,
): Promise<TriageStopTimes | null> {
  const data = await requestGraphQL({
    document: TriageStopTimesDocument,
    operationName: "TriageStopTimes",
    variables: { messageId },
    signal: options?.signal,
  });

  return data.triageStopTimes === null ? null : unmask(data.triageStopTimes);
}

export async function fetchInboundReplyAttachables(
  messageId: string,
  options?: RequestOptions,
): Promise<InboundReplyAttachables> {
  const data = await requestGraphQL({
    document: InboundReplyAttachablesDocument,
    operationName: "InboundReplyAttachables",
    variables: { messageId },
    signal: options?.signal,
  });

  return {
    documents: unmask(data.inboundReplyAttachables),
    limitBytes: data.inboundReplyAttachmentLimit,
  };
}

export async function fetchTriageShipmentPeek(
  shipmentId: string,
  options?: RequestOptions,
): Promise<TriageShipmentPeek> {
  const data = await requestGraphQL({
    document: TriageShipmentPeekDocument,
    operationName: "TriageShipmentPeek",
    variables: { shipmentId },
    signal: options?.signal,
  });

  return unmask(data.triageShipmentPeek);
}

export async function fetchTriageCustomerPeek(
  customerId: string,
  options?: RequestOptions,
): Promise<TriageCustomerPeek> {
  const data = await requestGraphQL({
    document: TriageCustomerPeekDocument,
    operationName: "TriageCustomerPeek",
    variables: { customerId },
    signal: options?.signal,
  });

  return unmask(data.triageCustomerPeek);
}

/** Which messages a bulk write touches: named ones, or a whole folder up to the server's bound. */
export type TriageSelection = { ids: readonly string[] } | { folder: InboundMessageFilter };

function selectionInput(selection: TriageSelection): TriageSelectionInput {
  return "ids" in selection ? { ids: [...selection.ids] } : { folder: selection.folder };
}

export async function reviewInboundMessage(
  id: string,
  input: ReviewInboundMessageInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: ReviewInboundMessageDocument,
    operationName: "ReviewInboundMessage",
    variables: { id, input },
  });

  return unmask(data.reviewInboundMessage);
}

export async function reviewInboundMessages(
  selection: TriageSelection,
  input: BulkReviewInboundMessagesInput,
  options?: IdempotentOptions,
): Promise<TriageBulkResult> {
  const data = await requestGraphQL({
    document: ReviewInboundMessagesDocument,
    operationName: "ReviewInboundMessages",
    variables: { selection: selectionInput(selection), input },
    idempotencyKey: options?.idempotencyKey,
  });

  return unmask(data.reviewInboundMessages);
}

export async function reopenInboundMessages(selection: TriageSelection): Promise<TriageBulkResult> {
  const data = await requestGraphQL({
    document: ReopenInboundMessagesDocument,
    operationName: "ReopenInboundMessages",
    variables: { selection: selectionInput(selection) },
  });

  return unmask(data.reopenInboundMessages);
}

export async function assignInboundMessages(
  selection: TriageSelection,
  ownerId: string | null,
): Promise<TriageBulkResult> {
  const data = await requestGraphQL({
    document: AssignInboundMessagesDocument,
    operationName: "AssignInboundMessages",
    variables: { selection: selectionInput(selection), ownerId },
  });

  return unmask(data.assignInboundMessages);
}

export async function snoozeInboundMessages(
  selection: TriageSelection,
  input: SnoozeInboundMessagesInput,
): Promise<TriageBulkResult> {
  const data = await requestGraphQL({
    document: SnoozeInboundMessagesDocument,
    operationName: "SnoozeInboundMessages",
    variables: { selection: selectionInput(selection), input },
  });

  return unmask(data.snoozeInboundMessages);
}

export async function wakeInboundMessage(id: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: WakeInboundMessageDocument,
    operationName: "WakeInboundMessage",
    variables: { id },
  });

  return unmask(data.wakeInboundMessage);
}

export async function waitForInboundReply(
  id: string,
  until: number,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: WaitForInboundReplyDocument,
    operationName: "WaitForInboundReply",
    variables: { id, until },
  });

  return unmask(data.waitForInboundReply);
}

export async function stopWaitingForInboundReply(id: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: StopWaitingForInboundReplyDocument,
    operationName: "StopWaitingForInboundReply",
    variables: { id },
  });

  return unmask(data.stopWaitingForInboundReply);
}

export async function linkInboundMessage(
  id: string,
  input: LinkInboundMessageInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: LinkInboundMessageDocument,
    operationName: "LinkInboundMessage",
    variables: { id, input },
  });

  return unmask(data.linkInboundMessage);
}

export async function unlinkInboundMessage(
  id: string,
  record: InboundLinkedRecord,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: UnlinkInboundMessageDocument,
    operationName: "UnlinkInboundMessage",
    variables: { id, record },
  });

  return unmask(data.unlinkInboundMessage);
}

export async function sendInboundMailboxTest(id: string): Promise<InboundMailbox> {
  const data = await requestGraphQL({
    document: SendInboundMailboxTestDocument,
    operationName: "SendInboundMailboxTest",
    variables: { id },
  });

  return mailbox(data.sendInboundMailboxTest);
}

export type TriageHandOff = { affected: number; to: TriageUser };

export async function handOffTriage(toUserId: string): Promise<TriageHandOff> {
  const data = await requestGraphQL({
    document: HandOffTriageDocument,
    operationName: "HandOffTriage",
    variables: { toUserId },
  });

  return unmask(data.handOffTriage);
}

export async function readUnreadInboundMessages(): Promise<number> {
  const data = await requestGraphQL({
    document: ReadUnreadInboundMessagesDocument,
    operationName: "ReadUnreadInboundMessages",
    variables: {},
  });

  return data.readUnreadInboundMessages;
}

export async function createMyTriageView(name: string, query: string): Promise<TriageView> {
  const data = await requestGraphQL({
    document: CreateMyTriageViewDocument,
    operationName: "CreateMyTriageView",
    variables: { name, query },
  });

  return unmask(data.createMyTriageView);
}

export async function renameMyTriageView(id: string, name: string): Promise<TriageView> {
  const data = await requestGraphQL({
    document: RenameMyTriageViewDocument,
    operationName: "RenameMyTriageView",
    variables: { id, name },
  });

  return unmask(data.renameMyTriageView);
}

export async function deleteMyTriageView(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteMyTriageViewDocument,
    operationName: "DeleteMyTriageView",
    variables: { id },
  });

  return data.deleteMyTriageView;
}

export async function updateTriageResponseTargets(
  input: TriageResponseTargetInput[],
): Promise<TriageResponseTarget[]> {
  const data = await requestGraphQL({
    document: UpdateTriageResponseTargetsDocument,
    operationName: "UpdateTriageResponseTargets",
    variables: { input },
  });

  return data.updateTriageResponseTargets;
}

export async function updateTriageSettings(input: TriageSettingsInput): Promise<TriageSettings> {
  const data = await requestGraphQL({
    document: UpdateTriageSettingsDocument,
    operationName: "UpdateTriageSettings",
    variables: { input },
  });

  return unmask(data.updateTriageSettings);
}

export async function createTriageRule(input: TriageRuleInput): Promise<TriageRule> {
  const data = await requestGraphQL({
    document: CreateTriageRuleDocument,
    operationName: "CreateTriageRule",
    variables: { input },
  });

  return unmask(data.createTriageRule);
}

export async function updateTriageRule(
  id: string,
  version: number,
  input: TriageRuleInput,
): Promise<TriageRule> {
  const data = await requestGraphQL({
    document: UpdateTriageRuleDocument,
    operationName: "UpdateTriageRule",
    variables: { id, version, input },
  });

  return unmask(data.updateTriageRule);
}

export async function setTriageRuleEnabled(id: string, enabled: boolean): Promise<TriageRule> {
  const data = await requestGraphQL({
    document: SetTriageRuleEnabledDocument,
    operationName: "SetTriageRuleEnabled",
    variables: { id, enabled },
  });

  return unmask(data.setTriageRuleEnabled);
}

export async function deleteTriageRule(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteTriageRuleDocument,
    operationName: "DeleteTriageRule",
    variables: { id },
  });

  return data.deleteTriageRule;
}

export async function correctInboundClassification(
  id: string,
  classification: InboundClassification,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: CorrectInboundClassificationDocument,
    operationName: "CorrectInboundClassification",
    variables: { id, classification },
  });

  return unmask(data.correctInboundClassification);
}

export async function flagInboundFact(id: string, key: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: FlagInboundFactDocument,
    operationName: "FlagInboundFact",
    variables: { id, key },
  });

  return unmask(data.flagInboundFact);
}

export async function resolveInboundConflict(
  id: string,
  conflictId: string,
  resolution: TriageConflictResolution,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: ResolveInboundConflictDocument,
    operationName: "ResolveInboundConflict",
    variables: { id, conflictId, resolution },
  });

  return unmask(data.resolveInboundConflict);
}

export async function reopenInboundConflict(
  id: string,
  conflictId: string,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: ReopenInboundConflictDocument,
    operationName: "ReopenInboundConflict",
    variables: { id, conflictId },
  });

  return unmask(data.reopenInboundConflict);
}

export async function draftInboundReply(messageId: string): Promise<InboundReply> {
  const data = await requestGraphQL({
    document: DraftInboundReplyDocument,
    operationName: "DraftInboundReply",
    variables: { messageId },
  });

  return unmask(data.draftInboundReply);
}

/** documentIds replaces the draft's files; leaving it out keeps the ones it has. */
export async function saveInboundReplyDraft(
  messageId: string,
  subject: string,
  body: string,
  documentIds?: readonly string[],
): Promise<InboundReply> {
  const data = await requestGraphQL({
    document: SaveInboundReplyDraftDocument,
    operationName: "SaveInboundReplyDraft",
    variables: {
      messageId,
      subject,
      body,
      ...(documentIds === undefined ? {} : { documentIds: [...documentIds] }),
    },
  });

  return unmask(data.saveInboundReplyDraft);
}

export async function discardInboundReplyDraft(messageId: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: DiscardInboundReplyDraftDocument,
    operationName: "DiscardInboundReplyDraft",
    variables: { messageId },
  });

  return unmask(data.discardInboundReplyDraft);
}

/** A refusal comes back as a Failed reply with failureText, not as an error. */
export async function sendInboundReply(
  input: SendInboundReplyInput,
  options?: IdempotentOptions,
): Promise<InboundReply> {
  const data = await requestGraphQL({
    document: SendInboundReplyDocument,
    operationName: "SendInboundReply",
    variables: { input },
    idempotencyKey: options?.idempotencyKey,
  });

  return unmask(data.sendInboundReply);
}

export async function draftInboundMany(messageIds: readonly string[]): Promise<number> {
  const data = await requestGraphQL({
    document: DraftInboundManyDocument,
    operationName: "DraftInboundMany",
    variables: { messageIds: [...messageIds] },
  });

  return data.draftInboundMany;
}

export async function createTriageReplyTemplate(
  input: TriageReplyTemplateInput,
): Promise<TriageReplyTemplate> {
  const data = await requestGraphQL({
    document: CreateTriageReplyTemplateDocument,
    operationName: "CreateTriageReplyTemplate",
    variables: { input },
  });

  return unmask(data.createTriageReplyTemplate);
}

export async function updateTriageReplyTemplate(
  id: string,
  version: number,
  input: TriageReplyTemplateInput,
): Promise<TriageReplyTemplate> {
  const data = await requestGraphQL({
    document: UpdateTriageReplyTemplateDocument,
    operationName: "UpdateTriageReplyTemplate",
    variables: { id, version, input },
  });

  return unmask(data.updateTriageReplyTemplate);
}

export async function deleteTriageReplyTemplate(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteTriageReplyTemplateDocument,
    operationName: "DeleteTriageReplyTemplate",
    variables: { id },
  });

  return data.deleteTriageReplyTemplate;
}

export async function draftInboundLoad(messageId: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: DraftInboundLoadDocument,
    operationName: "DraftInboundLoad",
    variables: { messageId },
  });

  return unmask(data.draftInboundLoad);
}

export async function discardInboundLoad(messageId: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: DiscardInboundLoadDocument,
    operationName: "DiscardInboundLoad",
    variables: { messageId },
  });

  return unmask(data.discardInboundLoad);
}

export async function createInboundLoad(
  messageId: string,
  fields: TriageLoadFieldInput[],
  options?: IdempotentOptions,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: CreateInboundLoadDocument,
    operationName: "CreateInboundLoad",
    variables: { messageId, fields },
    idempotencyKey: options?.idempotencyKey,
  });

  return unmask(data.createInboundLoad);
}

export async function fileInboundAttachment(
  input: FileInboundAttachmentInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: FileInboundAttachmentDocument,
    operationName: "FileInboundAttachment",
    variables: { input },
  });

  return unmask(data.fileInboundAttachment);
}

export async function fileInboundMany(messageIds: readonly string[]): Promise<TriageBulkResult> {
  const data = await requestGraphQL({
    document: FileInboundManyDocument,
    operationName: "FileInboundMany",
    variables: { messageIds: [...messageIds] },
  });

  return unmask(data.fileInboundMany);
}

export async function unfileInboundAttachment(attachmentId: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: UnfileInboundAttachmentDocument,
    operationName: "UnfileInboundAttachment",
    variables: { attachmentId },
  });

  return unmask(data.unfileInboundAttachment);
}

export async function logInboundException(
  messageId: string,
  input: LogInboundExceptionInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: LogInboundExceptionDocument,
    operationName: "LogInboundException",
    variables: { messageId, input },
  });

  return unmask(data.logInboundException);
}

export async function blockInboundSender(messageId: string): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: BlockInboundSenderDocument,
    operationName: "BlockInboundSender",
    variables: { messageId },
  });

  return unmask(data.blockInboundSender);
}

export async function unblockInboundSender(
  mailboxId: string,
  sender: string,
): Promise<InboundMailbox> {
  const data = await requestGraphQL({
    document: UnblockInboundSenderDocument,
    operationName: "UnblockInboundSender",
    variables: { mailboxId, sender },
  });

  return mailbox(data.unblockInboundSender);
}

export async function decideInboundStops(
  messageId: string,
  decision: TriageStopsDecision,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: DecideInboundStopsDocument,
    operationName: "DecideInboundStops",
    variables: { messageId, decision },
  });

  return unmask(data.decideInboundStops);
}

function credentials(
  masked: FragmentType<typeof InboundMailboxCredentialsFieldsFragmentDoc>,
): InboundMailboxCredentials {
  return unmask(getFragmentData(InboundMailboxCredentialsFieldsFragmentDoc, masked));
}

function mailbox(masked: FragmentType<typeof InboundMailboxFieldsFragmentDoc>): InboundMailbox {
  return unmask(getFragmentData(InboundMailboxFieldsFragmentDoc, masked));
}

export async function createInboundMailbox(
  input: InboundMailboxInput,
  signingSecret: string | null,
  apiKey: string | null,
): Promise<InboundMailboxCredentials> {
  const data = await requestGraphQL({
    document: CreateInboundMailboxDocument,
    operationName: "CreateInboundMailbox",
    variables: { input, signingSecret, apiKey },
  });

  return credentials(data.createInboundMailbox);
}

export async function updateInboundMailbox(
  id: string,
  version: number,
  input: InboundMailboxInput,
): Promise<InboundMailbox> {
  const data = await requestGraphQL({
    document: UpdateInboundMailboxDocument,
    operationName: "UpdateInboundMailbox",
    variables: { id, version, input },
  });

  return mailbox(data.updateInboundMailbox);
}

export async function rotateInboundMailboxToken(id: string): Promise<InboundMailboxCredentials> {
  const data = await requestGraphQL({
    document: RotateInboundMailboxTokenDocument,
    operationName: "RotateInboundMailboxToken",
    variables: { id },
  });

  return credentials(data.rotateInboundMailboxToken);
}

export async function setInboundMailboxSigningSecret(
  id: string,
  secret: string,
): Promise<InboundMailbox> {
  const data = await requestGraphQL({
    document: SetInboundMailboxSigningSecretDocument,
    operationName: "SetInboundMailboxSigningSecret",
    variables: { id, secret },
  });

  return mailbox(data.setInboundMailboxSigningSecret);
}

export async function setInboundMailboxApiKey(id: string, apiKey: string): Promise<InboundMailbox> {
  const data = await requestGraphQL({
    document: SetInboundMailboxApiKeyDocument,
    operationName: "SetInboundMailboxApiKey",
    variables: { id, apiKey },
  });

  return mailbox(data.setInboundMailboxApiKey);
}
