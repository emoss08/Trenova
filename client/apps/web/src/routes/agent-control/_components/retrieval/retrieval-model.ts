import { centsToDecimal, decimalToCents } from "@/lib/decimal-cents";
import type {
  AIRetrievalAvailability,
  AIRetrievalFailedEntryRow,
  AIRetrievalModelChange,
  AIRetrievalReindexEstimate,
  AIRetrievalSettings,
  AIRetrievalSettingsPatch,
  AIRetrievalSource,
  AIRetrievalSourceType,
  AIRetrievalStatus,
  AIRetrievalUnavailableReason,
} from "@/lib/graphql/ai-retrieval";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { z } from "zod";

/** The index moves with the indexer, a batch at a time; half a minute old is still true. */
export const RETRIEVAL_STALE_MS = 30_000;

/** How often the section reads the status again while something is being indexed. */
export const RETRIEVAL_REFRESH_MS = 30_000;

/** The command that creates the vector tables once pgvector is installed. */
export const ENABLE_VECTOR_COMMAND = "trenova db enable-vector";

/** The sources in the order the server lists them and the section shows them. */
export const RETRIEVAL_SOURCE_TYPES: readonly AIRetrievalSourceType[] = [
  "Memory",
  "Document",
  "InboundMessage",
];

/** What each source is, in the words of the settings and the status list. */
export const SOURCE_LABEL: Record<AIRetrievalSourceType, { label: string; description: string }> = {
  Memory: {
    label: "Memories",
    description: "What the organization told its agents and what they recorded",
  },
  Document: {
    label: "Documents",
    description: "Document text, when the record it belongs to may be read by a model",
  },
  InboundMessage: {
    label: "Inbound email",
    description: "The sender's own words, never the quoted thread",
  },
};

export function sourceTypeChoices(t: TranslateFn): { value: string; label: string }[] {
  return RETRIEVAL_SOURCE_TYPES.map((sourceType) => ({
    value: sourceType,
    label: t(SOURCE_LABEL[sourceType].label),
  }));
}

/** Where a notice points the operator for its fix. */
export type RetrievalFix = "providers" | "settings" | "command" | "none";

export type RetrievalNotice = {
  variant: "info" | "warning" | "destructive";
  title: string;
  message: string;
  fix: RetrievalFix;
};

/**
 * Why search by meaning is not answering, and the one thing that fixes it.
 * Every reason the server can give has an entry, so a new reason fails the
 * type check rather than showing nothing.
 */
export const UNAVAILABLE_NOTICE: Record<AIRetrievalUnavailableReason, RetrievalNotice> = {
  ExtensionMissing: {
    variant: "warning",
    title: "The database has no pgvector",
    message:
      "Search is by keyword only. Install pgvector 0.8 or newer in the database, then run this on the server:",
    fix: "command",
  },
  TooOld: {
    variant: "warning",
    title: "pgvector is older than 0.8",
    message:
      "Search is by keyword only. Upgrade pgvector to 0.8 or newer, then run this on the server:",
    fix: "command",
  },
  SchemaMissing: {
    variant: "warning",
    title: "The retrieval tables are missing",
    message:
      "pgvector is installed, but the tables that hold the index were never created. Run this on the server; running services notice within five minutes:",
    fix: "command",
  },
  NoProvider: {
    variant: "info",
    title: "No embedding model is routed",
    message:
      "Search is by keyword only until a provider serves the Embedding task. Route it on the Providers tab.",
    fix: "providers",
  },
  Disabled: {
    variant: "info",
    title: "Indexing is paused",
    message:
      "Search is by keyword only and nothing new is indexed. Resume indexing in the settings below.",
    fix: "settings",
  },
  BudgetPaused: {
    variant: "warning",
    title: "This month's indexing budget is spent",
    message:
      "Search is by keyword only and nothing new is indexed until the first of next month (UTC). Raise the monthly budget in the settings below to resume now.",
    fix: "settings",
  },
  NotIndexed: {
    variant: "info",
    title: "Indexing will start",
    message:
      "An embedding model is routed, but nothing is indexed under it yet. Indexing starts on its own within the hour, and search is by keyword until then.",
    fix: "none",
  },
  QueryTimeout: {
    variant: "warning",
    title: "The embedding provider is too slow",
    message:
      "A search waited too long for the provider and fell back to keywords. Check the provider's health on the Providers tab.",
    fix: "providers",
  },
  ProviderFailed: {
    variant: "destructive",
    title: "The embedding provider is failing",
    message:
      "A search could not be embedded and fell back to keywords. Test the provider on the Providers tab.",
    fix: "providers",
  },
};

/** The notice to show above the section, or none when search by meaning works. */
export function availabilityNotice(availability: AIRetrievalAvailability): RetrievalNotice | null {
  if (availability.available || !availability.reason) {
    return null;
  }

  return UNAVAILABLE_NOTICE[availability.reason];
}

/** Reasons that stop the indexer outright, whatever the settings say. */
const BLOCKING_REASONS: ReadonlySet<AIRetrievalUnavailableReason> = new Set([
  "ExtensionMissing",
  "TooOld",
  "SchemaMissing",
  "NoProvider",
]);

export type SourceState = "off" | "paused" | "pending" | "indexing" | "indexed" | "failed";

/** Where a source's indexing stands; the tone follows from the phase. */
export const SOURCE_STATE: Record<SourceState, BadgeAttrProps> = {
  off: { phase: "closed", text: "Off", description: "Not indexed; found by its words only" },
  paused: {
    phase: "attention",
    text: "Paused",
    description: "Nothing new is indexed until the notice above is dealt with",
  },
  pending: {
    phase: "queued",
    text: "Pending",
    description: "Waiting for the indexer to reach it",
  },
  indexing: { phase: "active", text: "Indexing", description: "Being embedded now" },
  indexed: { phase: "complete", text: "Indexed", description: "Everything is indexed" },
  failed: {
    phase: "failed",
    text: "Failed",
    description: "Some items could not be indexed; they are listed below",
  },
};

/** Items of a source not yet indexed: queued entries and rows the sweep has not reached. */
export function sourceWaiting(source: AIRetrievalSource): number {
  if (!source.enabled) {
    return 0;
  }

  return Math.max(
    0,
    source.pending,
    source.total - source.indexed - source.skipped - source.failed,
  );
}

export function sourceState(
  source: AIRetrievalSource,
  status: Pick<AIRetrievalStatus, "availability" | "settings">,
): SourceState {
  if (!source.enabled) {
    return "off";
  }

  const reason = status.availability.reason;
  if (status.settings.paused || (reason !== null && BLOCKING_REASONS.has(reason))) {
    return "paused";
  }

  if (source.pending > 0) {
    const started =
      source.indexed + source.failed + source.skipped > 0 || source.lastAttemptAt !== null;
    return started ? "indexing" : "pending";
  }

  if (source.failed > 0) {
    return "failed";
  }

  return sourceWaiting(source) > 0 ? "pending" : "indexed";
}

export type RetrievalTotals = {
  indexed: number;
  waiting: number;
  failed: number;
};

/** The figures across every source the organization indexes. */
export function retrievalTotals(sources: readonly AIRetrievalSource[]): RetrievalTotals {
  return sources.reduce<RetrievalTotals>(
    (totals, source) => ({
      indexed: totals.indexed + source.indexed,
      waiting: totals.waiting + sourceWaiting(source),
      failed: totals.failed + source.failed,
    }),
    { indexed: 0, waiting: 0, failed: 0 },
  );
}

/** Indexing and search together, the month's AI spend on retrieval. */
export function monthCost(status: AIRetrievalStatus): number {
  const indexing = Number(status.indexingCostMonthUsd);
  const retrieval = Number(status.retrievalCostMonthUsd);

  return (Number.isFinite(indexing) ? indexing : 0) + (Number.isFinite(retrieval) ? retrieval : 0);
}

/** Whether this month's indexing has reached its budget. */
export function budgetReached(status: AIRetrievalStatus): boolean {
  const spent = Number(status.indexingCostMonthUsd);
  const budget = Number(status.settings.monthlyIndexingBudgetUsd);

  return Number.isFinite(spent) && Number.isFinite(budget) && spent >= budget;
}

/** Keep reading the status while the index is moving, and stop once it is still. */
export function retrievalRefetchInterval(status: AIRetrievalStatus | undefined): number | false {
  if (!status) {
    return false;
  }

  const moving =
    status.modelChange !== null ||
    status.sources.some((source) => source.enabled && source.pending > 0);

  return moving ? RETRIEVAL_REFRESH_MS : false;
}

/** How far a model change has come, from 0 to 1. */
export function modelChangeShare(change: AIRetrievalModelChange): number {
  if (change.total <= 0) {
    return 0;
  }

  return Math.min(1, Math.max(0, change.indexed / change.total));
}

/** What the rail says about retrieval before the section is opened. */
export type RetrievalRailState = {
  available: boolean;
  reason: AIRetrievalUnavailableReason | null;
  failed: number;
  waiting: number;
};

export function retrievalRailState(status: AIRetrievalStatus): RetrievalRailState {
  const totals = retrievalTotals(status.sources);

  return {
    available: status.availability.available,
    reason: status.availability.available ? null : status.availability.reason,
    failed: totals.failed,
    waiting: totals.waiting,
  };
}

/** Reasons that mean something already set up has stopped working. */
const ATTENTION_REASONS: ReadonlySet<AIRetrievalUnavailableReason> = new Set([
  "BudgetPaused",
  "QueryTimeout",
  "ProviderFailed",
  "SchemaMissing",
  "TooOld",
]);

export function retrievalRailStatus(
  state: RetrievalRailState,
  t: TranslateFn,
): { status: string; attention: boolean } {
  if (!state.available) {
    const attention = state.reason !== null && ATTENTION_REASONS.has(state.reason);
    switch (state.reason) {
      case "BudgetPaused":
        return { status: t("Budget spent"), attention };
      case "QueryTimeout":
      case "ProviderFailed":
        return { status: t("Provider failing"), attention };
      case "Disabled":
        return { status: t("Paused"), attention };
      case "NotIndexed":
        return { status: t("Starting"), attention };
      default:
        return { status: t("Words only"), attention };
    }
  }

  if (state.failed > 0) {
    return {
      status: t("{0, plural, one {# failed} other {# failed}}", state.failed),
      attention: true,
    };
  }
  if (state.waiting > 0) {
    return {
      status: t("{0, plural, one {# to index} other {# to index}}", state.waiting),
      attention: false,
    };
  }

  return { status: t("On"), attention: false };
}

/** The failed-entry statuses a table filters by: a retry scheduled, or none left. */
export const FAILED_ENTRY_STATUS: Record<"Pending" | "Failed", BadgeAttrProps> = {
  Pending: {
    phase: "queued",
    text: "Retrying",
    description: "Another attempt is scheduled",
  },
  Failed: {
    phase: "failed",
    text: "Failed",
    description: "No retry is left; re-index the source after fixing the cause",
  },
};

export function failedEntryStatus(row: Pick<AIRetrievalFailedEntryRow, "status">): BadgeAttrProps {
  return row.status === "Failed" ? FAILED_ENTRY_STATUS.Failed : FAILED_ENTRY_STATUS.Pending;
}

export function failedEntryStatusChoices(t: TranslateFn): { value: string; label: string }[] {
  return (Object.keys(FAILED_ENTRY_STATUS) as (keyof typeof FAILED_ENTRY_STATUS)[]).map(
    (status) => ({ value: status, label: t(FAILED_ENTRY_STATUS[status].text) }),
  );
}

/** Whether a re-index could cost more than is left of this month's budget. */
export function estimateExceedsBudget(estimate: AIRetrievalReindexEstimate): boolean {
  if (estimate.estimatedCostUsd === null) {
    return false;
  }

  const cost = Number(estimate.estimatedCostUsd);
  const remaining = Number(estimate.remainingBudgetUsd);

  return Number.isFinite(cost) && Number.isFinite(remaining) && cost > remaining;
}

export const retrievalSettingsSchema = z.object({
  memoryEnabled: z.boolean(),
  documentsEnabled: z.boolean(),
  inboundMessagesEnabled: z.boolean(),
  paused: z.boolean(),
  monthlyIndexingBudgetCents: z
    .number()
    .int()
    .min(0, "A budget cannot be negative")
    .max(10_000_000, "A budget is at most 100,000"),
});

export type RetrievalSettingsFormValues = z.infer<typeof retrievalSettingsSchema>;

/** What the settings form starts from: the saved settings, in the units a person edits. */
export function toSettingsFormValues(settings: AIRetrievalSettings): RetrievalSettingsFormValues {
  return {
    memoryEnabled: settings.memoryEnabled,
    documentsEnabled: settings.documentsEnabled,
    inboundMessagesEnabled: settings.inboundMessagesEnabled,
    paused: settings.paused && settings.pausedReason === "Manual",
    monthlyIndexingBudgetCents: decimalToCents(settings.monthlyIndexingBudgetUsd),
  };
}

/**
 * Only what the person changed. The server leaves an absent field alone, so
 * saving one switch never overwrites a budget someone else raised meanwhile,
 * and a budget pause is not lifted by a form that never touched pausing.
 */
export function toSettingsPatch(
  values: RetrievalSettingsFormValues,
  initial: RetrievalSettingsFormValues,
): AIRetrievalSettingsPatch {
  const patch: AIRetrievalSettingsPatch = {};

  if (values.memoryEnabled !== initial.memoryEnabled) {
    patch.memoryEnabled = values.memoryEnabled;
  }
  if (values.documentsEnabled !== initial.documentsEnabled) {
    patch.documentsEnabled = values.documentsEnabled;
  }
  if (values.inboundMessagesEnabled !== initial.inboundMessagesEnabled) {
    patch.inboundMessagesEnabled = values.inboundMessagesEnabled;
  }
  if (values.paused !== initial.paused) {
    patch.paused = values.paused;
  }
  if (values.monthlyIndexingBudgetCents !== initial.monthlyIndexingBudgetCents) {
    patch.monthlyIndexingBudgetUsd = centsToDecimal(values.monthlyIndexingBudgetCents);
  }

  return patch;
}

/** The settings field that turns a source on or off. */
export const SOURCE_SETTING: Record<
  AIRetrievalSourceType,
  "memoryEnabled" | "documentsEnabled" | "inboundMessagesEnabled"
> = {
  Memory: "memoryEnabled",
  Document: "documentsEnabled",
  InboundMessage: "inboundMessagesEnabled",
};
