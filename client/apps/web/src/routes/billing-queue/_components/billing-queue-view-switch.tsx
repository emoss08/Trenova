import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { LayersIcon, ListChecksIcon } from "lucide-react";
import type { BillingQueueView } from "../use-billing-queue-state";

const VIEWS: readonly {
  key: BillingQueueView;
  label: string;
  hint: string;
  Icon: typeof ListChecksIcon;
}[] = [
  {
    key: "shipments",
    label: "Shipments",
    hint: "Review and approve shipments one at a time",
    Icon: ListChecksIcon,
  },
  {
    key: "statements",
    label: "Statements",
    hint: "Watch each customer's period build toward one invoice",
    Icon: LayersIcon,
  },
];

/**
 * The two halves of the queue.
 *
 * A segmented control rather than tabs: both views are the same work at
 * different grain, and tabs would imply the statements view is a detail of the
 * shipments one. The count rides on the Statements side because the reason to
 * look is almost always "how many are about to bill".
 */
export function BillingQueueViewSwitch({
  view,
  statementCount,
  onChange,
}: {
  view: BillingQueueView;
  statementCount: number | null;
  onChange: (next: BillingQueueView) => void;
}) {
  const t = useT();

  return (
    <div
      role="tablist"
      aria-label={t("Billing queue view")}
      className="bg-muted/60 inline-flex items-center gap-0.5 rounded-lg p-0.5"
    >
      {VIEWS.map(({ key, label, hint, Icon }) => {
        const active = view === key;
        return (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={active}
            title={hint}
            onClick={() => onChange(key)}
            className={cn(
              "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
              active
                ? "bg-card text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Icon className="size-3.5" />
            {label}
            {key === "statements" && statementCount != null && statementCount > 0 && (
              <span
                className={cn(
                  "rounded-full px-1.5 text-[10px] tabular-nums",
                  active ? "bg-muted text-foreground" : "bg-muted-foreground/15",
                )}
              >
                {statementCount}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
