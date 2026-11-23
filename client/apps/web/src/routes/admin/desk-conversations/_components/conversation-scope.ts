import type {
  DeskConversationFilterInput,
  DeskConversationOrigin,
  DeskTranscriptTimeframePreset,
} from "@/lib/graphql/desk-transcripts";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { searchCodec, searchJson } from "@trenova/shared/lib/router/search-codec";
import { z } from "zod";

/**
 * The timeframes the list filters by and an export takes, over a
 * conversation's last message. The server resolves each in the organization's
 * time zone (`desktranscript.Timeframe.Resolve`): "Last 7 days" is today and
 * the six days before it, "Last 12 months" starts on today's date a year ago,
 * and a custom range runs from the start of its first day to the end of its
 * last. The list and the export send the same preset, so they never disagree.
 */
export const TIMEFRAME_PRESETS = [
  "Last7Days",
  "Last30Days",
  "Last90Days",
  "Last12Months",
  "AllTime",
] as const satisfies readonly DeskTranscriptTimeframePreset[];

export const CONVERSATION_ORIGINS = [
  "Desk",
  "Panel",
  "Watchtower",
  "Briefing",
  "Import",
  "Formula",
] as const satisfies readonly DeskConversationOrigin[];

const DATE = /^\d{4}-\d{2}-\d{2}$/;

const timeframeSchema = z.discriminatedUnion("preset", [
  z.object({ preset: z.enum(TIMEFRAME_PRESETS) }),
  z
    .object({
      preset: z.literal("Custom"),
      fromDate: z.string().regex(DATE),
      toDate: z.string().regex(DATE),
    })
    .refine((range) => range.toDate >= range.fromDate),
]);

const scopeSchema = z.object({
  timeframe: timeframeSchema,
  personId: z.string().min(1).nullable().default(null),
  agent: z
    .object({ id: z.string().min(1), name: z.string() })
    .nullable()
    .default(null),
  origin: z.enum(CONVERSATION_ORIGINS).nullable().default(null),
});

export type ConversationTimeframe = z.infer<typeof timeframeSchema>;

/**
 * What the list of everyone's conversations is narrowed to, kept in the
 * address so a narrowed list can be shared and an export opens from it.
 */
export type ConversationScope = z.infer<typeof scopeSchema>;

export const DEFAULT_CONVERSATION_SCOPE: ConversationScope = {
  timeframe: { preset: "Last30Days" },
  personId: null,
  agent: null,
  origin: null,
};

export function parseConversationScope(value: unknown): ConversationScope | null {
  const parsed = scopeSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

export const CONVERSATION_SCOPE_PARAM = "scope";

export const conversationScopeParser = searchJson(parseConversationScope)
  .withOptions({ history: "replace" })
  .withDefault(DEFAULT_CONVERSATION_SCOPE);

export const CONVERSATION_VIEW_PARAM = "view";

export type ConversationView = "conversations" | "exports";

export const conversationViewParser = searchCodec<ConversationView>({
  parse: (value) => (value === "exports" ? "exports" : null),
  serialize: (value) => value,
})
  .withOptions({ history: "replace" })
  .withDefault("conversations");

/** The scope as the server's filter. */
export function scopeFilter(scope: ConversationScope): DeskConversationFilterInput {
  return {
    timeframe:
      scope.timeframe.preset === "Custom"
        ? {
            preset: "Custom",
            fromDate: scope.timeframe.fromDate,
            toDate: scope.timeframe.toDate,
          }
        : { preset: scope.timeframe.preset },
    userId: scope.personId,
    agentId: scope.agent?.id ?? null,
    origins: scope.origin ? [scope.origin] : null,
  };
}

/** A stable key for the scope, so the table starts over when the filters change. */
export function scopeKey(scope: ConversationScope): string {
  return JSON.stringify(scopeFilter(scope));
}

function shortDate(value: string): string {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export function timeframeLabel(
  t: TranslateFn,
  timeframe: {
    preset: DeskTranscriptTimeframePreset;
    fromDate?: string | null;
    toDate?: string | null;
  },
): string {
  switch (timeframe.preset) {
    case "Last7Days":
      return t("Last 7 days");
    case "Last30Days":
      return t("Last 30 days");
    case "Last90Days":
      return t("Last 90 days");
    case "Last12Months":
      return t("Last 12 months");
    case "AllTime":
      return t("All time");
    case "Custom":
      return timeframe.fromDate && timeframe.toDate
        ? t("{0} to {1}", shortDate(timeframe.fromDate), shortDate(timeframe.toDate))
        : t("Custom range");
  }
}

export function originLabel(t: TranslateFn, origin: DeskConversationOrigin): string {
  switch (origin) {
    case "Desk":
      return t("Desk");
    case "Panel":
      return t("Assistant panel");
    case "Watchtower":
      return t("Watchtower");
    case "Briefing":
      return t("Briefing");
    case "Import":
      return t("Shipment import");
    case "Formula":
      return t("Formula studio");
  }
}
