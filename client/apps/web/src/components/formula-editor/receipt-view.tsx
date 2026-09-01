import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  FormulaReceipt,
  FormulaValueSource,
  LookupMatch,
} from "@trenova/shared/types/formula-template";
import { Braces, TableIcon } from "lucide-react";
import { useMemo, useState } from "react";

const SOURCE_LABELS: Record<FormulaValueSource, { label: string; className: string }> = {
  field: { label: "shipment", className: "bg-sky-500/15 text-sky-700 dark:text-sky-300" },
  computed: {
    label: "computed",
    className: "bg-violet-500/15 text-violet-700 dark:text-violet-300",
  },
  input: { label: "input", className: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300" },
  override: {
    label: "override",
    className: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  },
  default: { label: "default", className: "bg-muted text-muted-foreground" },
  sample: { label: "sample", className: "bg-muted text-muted-foreground" },
};

/** Words for which entry answered a lookup, for the receipt and dispute letters alike. */
export function describeLookupMatch(match: LookupMatch | null | undefined): string {
  if (!match) return "no match";
  const moved = match.adjusted ? " (key moved into band)" : "";
  if (match.matchedKey) return `key ${match.matchedKey}${moved}`;
  if (match.bandMin != null) {
    const band =
      match.bandMax != null ? `band ${match.bandMin}–${match.bandMax}` : `band ${match.bandMin}+`;
    return `${band}${moved}`;
  }
  return "no match";
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "empty";
  if (typeof value === "number") return Number.isInteger(value) ? String(value) : value.toFixed(4);
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "string") return value === "" ? '""' : value;
  return JSON.stringify(value);
}

function isScalar(value: unknown): value is string | number | boolean {
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean";
}

type ReceiptViewProps = {
  receipt: FormulaReceipt;
  /** When given, offers to copy the receipt's scalar values back into sample data. */
  onUseValues?: (values: Record<string, unknown>) => void;
  className?: string;
};

/**
 * The calculation, read the way a person reads it: what every input was and
 * where it came from, which rate-table rows were consulted, and the amount
 * before guardrails and rounding touched it.
 */
export function ReceiptView({ receipt, onUseValues, className }: ReceiptViewProps) {
  const [showVariables, setShowVariables] = useState(false);
  const scalarValues = useMemo(
    () =>
      Object.fromEntries(
        receipt.variables
          .filter((variable) => isScalar(variable.value))
          .map((variable) => [variable.name, variable.value]),
      ),
    [receipt.variables],
  );

  const lookups = receipt.lookups ?? [];

  return (
    <div className={cn("space-y-3", className)}>
      <div className="text-muted-foreground text-2xs flex flex-wrap items-center gap-x-3 gap-y-1">
        <span>Raw {formatCurrency(receipt.rawAmount)} before guardrails and rounding</span>
        {receipt.versionNumber ? <span className="font-mono">v{receipt.versionNumber}</span> : null}
        {receipt.effectiveFrom ? <span>scheduled version</span> : null}
        {receipt.durationMicros ? (
          <span className="tabular-nums">{(receipt.durationMicros / 1000).toFixed(1)} ms</span>
        ) : null}
      </div>

      {lookups.length > 0 && (
        <div className="space-y-1">
          <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
            <TableIcon className="size-3" />
            Rate tables consulted
          </div>
          <ul className="bg-background/50 divide-y rounded-md border text-xs">
            {lookups.map((lookup, index) => (
              <li
                key={`${lookup.scope}:${lookup.table}:${index}`}
                className="flex items-center justify-between gap-3 px-3 py-1.5"
              >
                <div className="min-w-0">
                  <span className="font-mono">{lookup.table}</span>
                  <span className="text-muted-foreground">
                    {" "}
                    [{lookup.keys.map(formatValue).join(", ")}] →{" "}
                    {describeLookupMatch(lookup.match)}
                  </span>
                  {lookup.scope !== "expression" && (
                    <span className="text-muted-foreground"> · in {lookup.scope}</span>
                  )}
                </div>
                {lookup.error ? (
                  <span className="text-destructive shrink-0">{lookup.error}</span>
                ) : (
                  <span className="shrink-0 font-mono tabular-nums">
                    {formatValue(lookup.value)}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="space-y-1">
        <div className="flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={() => setShowVariables((prev) => !prev)}
            aria-expanded={showVariables}
            className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase"
          >
            <Braces className="size-3" />
            Variables ({receipt.variables.length})
          </button>
          {onUseValues && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              onClick={() => onUseValues(scalarValues)}
            >
              Use these values
            </Button>
          )}
        </div>
        {showVariables && (
          <ul className="bg-background/50 max-h-64 divide-y overflow-auto rounded-md border text-xs">
            {receipt.variables.map((variable) => {
              const source = SOURCE_LABELS[variable.source] ?? SOURCE_LABELS.sample;
              return (
                <li
                  key={variable.name}
                  className="flex items-center justify-between gap-3 px-3 py-1"
                >
                  <span className="min-w-0 truncate font-mono">{variable.name}</span>
                  <span className="flex shrink-0 items-center gap-2">
                    <span className="font-mono tabular-nums">{formatValue(variable.value)}</span>
                    <Badge
                      variant="outline"
                      className={cn("text-2xs border-transparent px-1 py-0", source.className)}
                    >
                      {source.label === "shipment" ? variable.source : source.label}
                    </Badge>
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
