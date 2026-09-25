import type { Tone } from "@/components/kpi/tone";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import { agentSubjectPath } from "@/lib/agent-subjects";
import type {
  AIAuditChainStatus,
  AiAuditEventKind,
  AiAuditEventOutcome,
  AiAuditExportFormat,
  AiAuditExportStatus,
  AiAuditVerificationStatus,
  RequestAiAuditExportInput,
} from "@/lib/graphql/ai-audit";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { endOfDay, startOfDay } from "date-fns";
import { parseAsJson } from "nuqs";
import type { FieldFilter, FilterGroup, SortField } from "@trenova/shared/types/data-table";
import { z } from "zod";

const DAY_SECONDS = 86_400;

/** The widest range one export may cover; the server refuses anything wider. */
export const AUDIT_MAX_EXPORT_RANGE_SECONDS = 2556 * DAY_SECONDS;

/** The most filters and filter groups one export may carry. */
export const AUDIT_MAX_EXPORT_FILTERS = 20;

/** The relative ranges the trail offers, in days back from now. */
export const AUDIT_RANGE_DAYS = [1, 7, 30, 90] as const;
export type AuditRangeDays = (typeof AUDIT_RANGE_DAYS)[number];

const rangeSchema = z.discriminatedUnion("kind", [
  z.object({
    kind: z.literal("last"),
    days: z.union([z.literal(1), z.literal(7), z.literal(30), z.literal(90)]),
  }),
  z
    .object({
      kind: z.literal("between"),
      from: z.number().int().positive(),
      to: z.number().int().positive(),
    })
    .refine((range) => range.to >= range.from),
]);

const scopeSchema = z.object({
  range: rangeSchema,
  agent: z
    .object({ id: z.string().min(1), name: z.string() })
    .nullable()
    .default(null),
  personId: z.string().min(1).nullable().default(null),
  includeEvaluations: z.boolean().default(false),
});

/**
 * What the trail is narrowed to from beside the table: when, which agent,
 * whose work, and whether evaluations are shown. It lives in the address, so
 * a narrowed trail can be shared; the table's own filters narrow it further.
 */
export type AuditTrailScope = z.infer<typeof scopeSchema>;
export type AuditRange = AuditTrailScope["range"];

export const DEFAULT_AUDIT_SCOPE: AuditTrailScope = {
  range: { kind: "last", days: 7 },
  agent: null,
  personId: null,
  includeEvaluations: false,
};

/** A scope read from the address, or null when the address cannot mean one. */
export function parseAuditScope(value: unknown): AuditTrailScope | null {
  const parsed = scopeSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

/** The address key the trail's scope is kept under; moving to another table clears it. */
export const AUDIT_SCOPE_PARAM = "trailScope";

export const auditScopeParser = parseAsJson(parseAuditScope)
  .withOptions({ history: "replace", shallow: true })
  .withDefault(DEFAULT_AUDIT_SCOPE);

function rangeFilters(range: AuditRange): FieldFilter[] {
  if (range.kind === "last") {
    return [{ field: "occurredAt", operator: "lastndays", value: range.days }];
  }

  return [
    { field: "occurredAt", operator: "gte", value: range.from },
    { field: "occurredAt", operator: "lte", value: range.to },
  ];
}

function narrowingFilters(scope: AuditTrailScope): FieldFilter[] {
  const filters: FieldFilter[] = [];
  if (scope.agent) {
    filters.push({ field: "agentDefinitionId", operator: "eq", value: scope.agent.id });
  }
  if (scope.personId) {
    filters.push({ field: "personId", operator: "eq", value: scope.personId });
  }
  if (!scope.includeEvaluations) {
    filters.push({ field: "purpose", operator: "eq", value: "Live" });
  }

  return filters;
}

/**
 * The scope as the filters the trail's query is sent with. `personId` is
 * the server's own filter for "done for or decided by this person".
 */
export function auditScopeFilters(scope: AuditTrailScope): FieldFilter[] {
  return [...rangeFilters(scope.range), ...narrowingFilters(scope)];
}

/** The scope's range as instants: a relative one ends now. */
export function auditScopeRange(
  scope: AuditTrailScope,
  nowSeconds: number,
): { from: number; to: number } {
  const { range } = scope;
  if (range.kind === "last") {
    return { from: nowSeconds - range.days * DAY_SECONDS, to: nowSeconds };
  }

  return { from: range.from, to: range.to };
}

export type AuditTableState = {
  query: string;
  fieldFilters: FieldFilter[];
  filterGroups: FilterGroup[];
  sort: SortField[];
};

export type AuditExportRequestParams = {
  format: AiAuditExportFormat;
  from: number;
  to: number;
  /** Carry what the trail is narrowed to into the file. */
  useCurrentFilters: boolean;
  scope: AuditTrailScope;
  table: AuditTableState;
};

/**
 * The export as the server takes it. Without current filters the file is
 * every row in the range, which is the only file whose chain can be checked
 * end to end. With them, the scope's narrowing and the table's own filters,
 * search and sort go along; the dialog's range replaces the scope's.
 */
export function buildAuditExportRequest({
  format,
  from,
  to,
  useCurrentFilters,
  scope,
  table,
}: AuditExportRequestParams): RequestAiAuditExportInput {
  if (!useCurrentFilters) {
    return { format, from, to };
  }

  const query = table.query.trim();
  return {
    format,
    from,
    to,
    ...(query === "" ? {} : { query }),
    fieldFilters: [...narrowingFilters(scope), ...table.fieldFilters],
    filterGroups: table.filterGroups,
    sort: table.sort,
  };
}

/** The first second of the reader's day an instant falls on. */
export function dayStartOf(unixSeconds: number): number {
  return fromUserWallClock(startOfDay(toUserWallClock(unixSeconds) ?? new Date(0))) ?? unixSeconds;
}

/** The last second of the reader's day an instant falls on. */
export function dayEndOf(unixSeconds: number): number {
  return fromUserWallClock(endOfDay(toUserWallClock(unixSeconds) ?? new Date(0))) ?? unixSeconds;
}

/**
 * The export dialog's form. Its dates are whole days in the reader's zone:
 * the start is the first second of its day, the end the last second of its
 * day, and together they may not span more than the server allows.
 */
export const auditExportFormSchema = z
  .object({
    format: z.enum(["CSV", "JSON"]),
    from: z.number({ error: "The start of the range is required" }).int().positive(),
    to: z.number({ error: "The end of the range is required" }).int().positive(),
    useCurrentFilters: z.boolean(),
  })
  .superRefine((value, ctx) => {
    const from = dayStartOf(value.from);
    const to = dayEndOf(value.to);
    if (to < from) {
      ctx.addIssue({
        code: "custom",
        path: ["to"],
        message: "The end of the range must not be before its start",
      });
    } else if (to - from > AUDIT_MAX_EXPORT_RANGE_SECONDS) {
      ctx.addIssue({
        code: "custom",
        path: ["to"],
        message: "An export can cover at most seven years",
      });
    }
  });

export type AuditExportFormValues = z.infer<typeof auditExportFormSchema>;

/** The dialog's opening values: the trail's own range, as whole days, and its filters carried. */
export function auditExportDefaults(
  scope: AuditTrailScope,
  nowSeconds: number,
): AuditExportFormValues {
  const { from, to } = auditScopeRange(scope, nowSeconds);
  return { format: "CSV", from: dayStartOf(from), to: dayStartOf(to), useCurrentFilters: true };
}

/** How many filters an export request carries, against the server's cap. */
export function auditExportFilterCount(request: RequestAiAuditExportInput): number {
  return (request.fieldFilters?.length ?? 0) + (request.filterGroups?.length ?? 0);
}

/**
 * How each outcome reads and where it sits: waiting on a person, done,
 * finished without a write, or gone wrong. The tone follows the phase.
 */
export function auditOutcomeAttrs(t: TranslateFn): Record<AiAuditEventOutcome, BadgeAttrProps> {
  return {
    Started: { phase: "active", text: t("Started") },
    Completed: { phase: "complete", text: t("Completed") },
    Succeeded: { phase: "complete", text: t("Succeeded") },
    Ran: { phase: "complete", text: t("Ran") },
    Accepted: { phase: "complete", text: t("Accepted") },
    Modified: { phase: "complete", text: t("Accepted with changes") },
    Proposed: { phase: "awaiting", text: t("Proposed") },
    Filed: { phase: "awaiting", text: t("Filed") },
    Refused: { phase: "attention", text: t("Refused") },
    Unknown: { phase: "attention", text: t("Outcome unknown") },
    Exhausted: { phase: "attention", text: t("Ran out of steps") },
    Simulated: { phase: "closed", text: t("Simulated") },
    Stopped: { phase: "closed", text: t("Stopped") },
    Declined: { phase: "closed", text: t("Declined") },
    Failed: { phase: "failed", text: t("Failed") },
    Denied: { phase: "failed", text: t("Denied") },
    Rejected: { phase: "failed", text: t("Rejected") },
    Expired: { phase: "failed", text: t("Expired") },
  };
}

export function auditKindLabel(t: TranslateFn, kind: AiAuditEventKind): string {
  switch (kind) {
    case "RunStarted":
      return t("Run started");
    case "RunEnded":
      return t("Run ended");
    case "ModelCall":
      return t("Model call");
    case "ToolCall":
      return t("Tool call");
    case "ToolRefused":
      return t("Tool refused");
    case "ProposalFiled":
      return t("Proposal filed");
    case "ProposalDecided":
      return t("Proposal decided");
    case "ProposalExecuted":
      return t("Proposal carried out");
    case "ProposalExecutionFailed":
      return t("Proposal failed to run");
    case "ProposalSimulated":
      return t("Proposal simulated");
    case "ProposalExpired":
      return t("Proposal expired");
    case "DelegationStarted":
      return t("Task handed off");
    case "DelegationEnded":
      return t("Handed-off task ended");
  }
}

export const AUDIT_EVENT_KINDS: readonly AiAuditEventKind[] = [
  "RunStarted",
  "RunEnded",
  "ModelCall",
  "ToolCall",
  "ToolRefused",
  "ProposalFiled",
  "ProposalDecided",
  "ProposalExecuted",
  "ProposalExecutionFailed",
  "ProposalSimulated",
  "ProposalExpired",
  "DelegationStarted",
  "DelegationEnded",
];

export const AUDIT_EVENT_OUTCOMES: readonly AiAuditEventOutcome[] = [
  "Started",
  "Completed",
  "Succeeded",
  "Ran",
  "Proposed",
  "Filed",
  "Accepted",
  "Modified",
  "Rejected",
  "Denied",
  "Refused",
  "Failed",
  "Unknown",
  "Simulated",
  "Stopped",
  "Expired",
  "Exhausted",
  "Declined",
];

export function auditExportStatusAttrs(
  t: TranslateFn,
): Record<AiAuditExportStatus, BadgeAttrProps> {
  return {
    Pending: { phase: "queued", text: t("Queued") },
    Running: { phase: "active", text: t("Writing") },
    Succeeded: { phase: "complete", text: t("Ready") },
    Failed: { phase: "failed", text: t("Failed") },
    Expired: { phase: "closed", text: t("Expired") },
  };
}

export function verificationLabel(
  t: TranslateFn,
  status: AiAuditVerificationStatus | null,
): string {
  switch (status) {
    case "Verified":
      return t("Verified");
    case "Mismatch":
      return t("Mismatch");
    case "KeyMissing":
      return t("Key missing");
    case null:
      return t("Not yet verified");
  }
}

/** How often the status is read again while a check someone started is running. */
export const VERIFY_POLL_MS = 5_000;

/** How long a started check is waited on before the button comes back regardless. */
export const VERIFY_WAIT_MS = 10 * 60_000;

/** A check someone started, and what the chain's last check said when they did. */
export type VerificationRequest = { baseline: number | null };

/**
 * Whether a check someone started is still running: its result has not
 * reached the chain's status yet. The server stores each result with the time
 * it finished, so a status whose last check is newer than the one seen when
 * the check was started carries its result.
 */
export function isVerificationPending(
  request: VerificationRequest | null,
  status: Pick<AIAuditChainStatus, "lastVerifiedAt"> | null,
): boolean {
  if (request === null || status === null) {
    return false;
  }

  return (status.lastVerifiedAt ?? null) === request.baseline;
}

/** A mismatch is the trail no longer matching its chain; a missing key only stops the check. */
export function verificationTone(status: AiAuditVerificationStatus | null): Tone {
  switch (status) {
    case "Verified":
      return "success";
    case "Mismatch":
      return "danger";
    case "KeyMissing":
      return "warning";
    case null:
      return "muted";
  }
}

export function tierSourceLabel(t: TranslateFn, source: string): string {
  switch (source) {
    case "PolicyDefault":
      return t("The tool's rule");
    case "PersonSetting":
      return t("The person's setting");
    case "TrustEarned":
      return t("Trust the agent earned");
    case "PersonalExemption":
      return t("Own records, person present");
    default:
      return source;
  }
}

export function auditTierLabel(t: TranslateFn, tier: string): string {
  switch (tier) {
    case "AutoExecute":
      return t("Automatic");
    case "ActWithApproval":
      return t("Ask first");
    case "Propose":
      return t("Propose");
    default:
      return tier;
  }
}

/**
 * Where the record an event touched opens. The trail names an agent's
 * subject by its subject type and a tool's target by its permission
 * resource, so both are tried against the record-link registry.
 */
export function auditEventRecordPath(
  entityType: string | null | undefined,
  entityId: string | null | undefined,
): string | null {
  if (!entityType || !entityId) {
    return null;
  }

  const subject = agentSubjectPath(entityType, entityId);
  if (subject !== null) {
    return subject;
  }

  return isRecordEntityType(entityType) ? recordPath(entityType, entityId) : null;
}
