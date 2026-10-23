import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatShownValue, formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffLookup, TariffRange, TariffTraceEvent } from "@trenova/shared/types/tariff";

/** What a trace event says, in the reader's language; labels and notes are shown as written. */
export function describeTraceEvent(
  event: TariffTraceEvent,
  lookups: readonly TariffLookup[],
  t: TranslateFn,
): string {
  switch (event.kind) {
    case "bind":
      return t("{0} = {1}", event.name, formatShownValue(event, t));
    case "assign":
      return t("{0} becomes {1}", event.name, formatShownValue(event, t));
    case "branch":
      return event.taken === null
        ? t("No branch of the if ran")
        : event.taken === 0
          ? t("Took the if branch")
          : t("Took branch {0} of the if", event.taken + 1);
    case "arm":
      return t("Matched arm {0}", event.arm + 1);
    case "loop":
      return t(
        "{0, plural, one {Looped once} other {Looped # times}} over {1}",
        event.iterations,
        event.name,
      );
    case "iteration":
      return t("{0} = {1} (pass {2})", event.name, formatShownValue(event, t), event.index + 1);
    case "charge":
      return event.code
        ? t("Billed {0} ({1}): {2}", event.label, event.code, formatShownValue(event, t))
        : t("Billed {0}: {1}", event.label, formatShownValue(event, t));
    case "dropped":
      return t("Dropped {0}: it came to zero", event.label);
    case "note":
      return t("Note: {0}", event.text);
    case "refusal":
      return t("Refused: {0}", event.message);
    case "lookup": {
      const lookup = lookups[event.lookup];
      if (!lookup) return t("Read a rate table");
      const keys = lookup.keys.map((key) => formatTariffValue(key, t)).join(", ");
      return lookup.value === undefined || lookup.value === null
        ? t("tables.{0}[{1}] found no cell", lookup.table, keys)
        : t("tables.{0}[{1}] = {2}", lookup.table, keys, formatShownValue(lookup, t));
    }
    case "choice":
      return event.chosen === null
        ? t("{0}: every option refused", event.label)
        : t("{0}: chose {1}", event.label, event.chosen);
    case "truncated":
      return t("The trace stops here; the rest of the rating is not recorded.");
  }
}

const TONE: Partial<Record<TariffTraceEvent["kind"], string>> = {
  charge: "text-foreground",
  choice: "text-foreground font-medium",
  refusal: "text-danger-foreground",
  dropped: "text-foreground-subtle",
  truncated: "text-foreground-subtle italic",
};

export type TraceViewProps = {
  trace: readonly TariffTraceEvent[];
  lookups?: readonly TariffLookup[];
  /** Selects a statement's source when its row is picked. */
  onSelectRange?: (range: TariffRange) => void;
  className?: string;
};

/**
 * The rating, statement by statement as it ran: each binding with its value, the branches and
 * arms taken, loops, the charges billed or dropped, notes, a refusal and every rate table read.
 */
export function TraceView({ trace, lookups = [], onSelectRange, className }: TraceViewProps) {
  const t = useT();
  if (trace.length === 0) {
    return <p className="text-foreground-muted text-xs">{t("Nothing ran.")}</p>;
  }
  return (
    <ol aria-label={t("Trace")} className={cn("flex flex-col font-mono text-xs", className)}>
      {trace.map((event, index) => {
        const range = "range" in event ? event.range : null;
        const text = describeTraceEvent(event, lookups, t);
        const indent = event.kind === "iteration" ? "pl-4" : "";
        return (
          // oxlint-disable-next-line react/no-array-index-key -- a trace has no identity but its order
          <li key={index} className={cn("text-foreground-muted", TONE[event.kind], indent)}>
            {onSelectRange && range ? (
              <button
                type="button"
                onClick={() => onSelectRange(range)}
                className="ui-focus-ring hover:bg-surface-hover w-full rounded-sm px-1 py-0.5 text-left break-words"
              >
                {text}
              </button>
            ) : (
              <span className="block px-1 py-0.5 break-words">{text}</span>
            )}
          </li>
        );
      })}
    </ol>
  );
}
