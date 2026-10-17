import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaValueSource } from "@trenova/shared/types/formula-template";

/** Where a value came from, as a tag: a tone for the person's own, accents for the rest. */
const SOURCE_STYLES: Record<FormulaValueSource, { label: string; className: string }> = {
  field: { label: "shipment", className: "bg-info-subtle text-info-subtle-foreground" },
  computed: {
    label: "computed",
    className: "bg-accent-violet-subtle text-accent-violet-on-subtle",
  },
  input: { label: "input", className: "bg-success-subtle text-success-subtle-foreground" },
  override: {
    label: "override",
    className: "bg-warning-subtle text-warning-subtle-foreground",
  },
  default: { label: "default", className: "bg-surface-hover text-foreground-muted" },
  sample: { label: "sample", className: "bg-surface-hover text-foreground-muted" },
  provided: {
    label: "market data",
    className: "bg-accent-amber-subtle text-accent-amber-on-subtle",
  },
};

export function ValueSourceTag({ source }: { source: FormulaValueSource }) {
  const t = useT();
  const style = SOURCE_STYLES[source];
  return (
    <span className={cn("shrink-0 rounded-badge px-1.5 text-2xs", style.className)}>
      {t(style.label)}
    </span>
  );
}
