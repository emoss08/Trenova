import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  TariffOutline,
  TariffOutlineKind,
  TariffRange,
  TariffTestStatus,
} from "@trenova/shared/types/tariff";

type Group = "inputs" | "declarations" | "functions" | "charges" | "totals" | "tests";

const GROUPS: readonly { key: Group; kinds: readonly TariffOutlineKind[] }[] = [
  { key: "inputs", kinds: ["input"] },
  { key: "declarations", kinds: ["const", "type", "enum"] },
  { key: "functions", kinds: ["function"] },
  { key: "charges", kinds: ["charge"] },
  { key: "totals", kinds: ["total"] },
  { key: "tests", kinds: ["test"] },
];

const GROUP_LABELS: Record<Group, string> = defineLabels({
  inputs: "Inputs",
  declarations: "Declarations",
  functions: "Functions",
  charges: "Charges",
  totals: "Totals",
  tests: "Tests",
});

const TEST_LABELS: Record<TariffTestStatus, string> = defineLabels({
  passed: "Passed",
  failed: "Failed",
  error: "Could not run",
});

const TEST_DOT: Record<TariffTestStatus, string> = {
  passed: "bg-success",
  failed: "bg-danger",
  error: "bg-warning",
};

export type OutlinePanelProps = {
  outline: TariffOutline | null;
  onReveal: (range: TariffRange) => void;
  /** Each test's verdict from the last run, by name, drawn as a dot beside it. */
  testStatus?: ReadonlyMap<string, TariffTestStatus>;
  className?: string;
};

/**
 * The file at a glance, from the checker's outline: its inputs, declarations, functions, every
 * charge (with its code, indented under the `if` or `for` it is in), the total clauses and the
 * tests. A row selects what it names in the editor.
 */
export function OutlinePanel({ outline, onReveal, testStatus, className }: OutlinePanelProps) {
  const t = useT();
  const items = outline?.items ?? [];
  const groups = GROUPS.map((group) => ({
    label: GROUP_LABELS[group.key],
    items: items.filter((item) => group.kinds.includes(item.kind)),
  })).filter((group) => group.items.length > 0);

  return (
    <nav aria-label={t("Outline")} className={cn("flex flex-col gap-3 p-3", className)}>
      {groups.length === 0 ? (
        <p className="text-foreground-muted px-1 text-xs">
          {t("Inputs, charges and tests appear here as you write them.")}
        </p>
      ) : (
        groups.map((group) => (
          <section key={group.label} className="flex flex-col gap-0.5">
            <h3 className="text-foreground-subtle px-1 text-xs font-medium">{group.label}</h3>
            <ul className="flex flex-col">
              {group.items.map((item) => {
                const status = item.kind === "test" ? testStatus?.get(item.name) : undefined;
                return (
                  <li key={`${item.kind}:${item.range.from}`}>
                    <button
                      type="button"
                      onClick={() => onReveal(item.selection)}
                      style={{ paddingLeft: `${4 + item.depth * 12}px` }}
                      className="ui-focus-ring hover:bg-surface-hover flex w-full min-w-0 items-center gap-1.5 rounded-sm py-1 pr-1 text-left text-xs"
                    >
                      {status && (
                        <span
                          aria-label={TEST_LABELS[status]}
                          className={cn("size-1.5 shrink-0 rounded-full", TEST_DOT[status])}
                        />
                      )}
                      <span
                        className={cn(
                          "min-w-0 truncate",
                          item.kind !== "charge" && item.kind !== "test" && "font-mono",
                        )}
                      >
                        {item.name}
                      </span>
                      {item.code && (
                        <span className="text-syntax-meta ml-auto shrink-0 font-mono text-2xs">
                          {item.code}
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))
      )}
    </nav>
  );
}
