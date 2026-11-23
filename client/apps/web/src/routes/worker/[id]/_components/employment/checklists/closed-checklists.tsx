import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { WorkerChecklistRow } from "@/lib/graphql/worker-checklist";
import { CheckIcon, ChevronRightIcon, MinusIcon } from "@trenova/shared/components/icons";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { itemLine } from "./checklist-item-row";

/** How a closed checklist ended, in one line. */
export function closedLine(checklist: WorkerChecklistRow): string {
  const { requiredDone, requiredTotal } = checklist.progress;
  if (checklist.status === "Cancelled") {
    const on = checklist.cancelledAt ? formatUnixDateMedium(checklist.cancelledAt) : null;
    if (on && checklist.cancelReason) {
      return translate("Cancelled {0}: {1}", on, checklist.cancelReason);
    }
    return on ? translate("Cancelled {0}", on) : translate("Cancelled");
  }
  return checklist.completedAt
    ? translate(
        "Completed {0} · {1} of {2}",
        formatUnixDateMedium(checklist.completedAt),
        requiredDone,
        requiredTotal,
      )
    : translate("Completed · {0} of {1}", requiredDone, requiredTotal);
}

/**
 * Closed checklists, newest first, each opening in place to show how every item settled. They
 * stay because an auditor asks, not because anybody reads them day to day.
 */
export function ClosedChecklists({
  checklists,
  expandedId,
}: {
  checklists: readonly WorkerChecklistRow[];
  /** A closed checklist a link sent the reader to: opened when first drawn. */
  expandedId: string | null;
}) {
  const t = useT();
  const sorted = [...checklists].sort(
    (a, b) =>
      (b.completedAt ?? b.cancelledAt ?? b.startedAt) -
      (a.completedAt ?? a.cancelledAt ?? a.startedAt),
  );
  const [open, setOpen] = useState<ReadonlySet<string>>(
    () => new Set(expandedId ? [expandedId] : []),
  );
  const [seenExpanded, setSeenExpanded] = useState(expandedId);
  if (expandedId !== seenExpanded) {
    setSeenExpanded(expandedId);
    if (expandedId && !open.has(expandedId)) setOpen(new Set(open).add(expandedId));
  }

  const toggle = (id: string) => {
    const next = new Set(open);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setOpen(next);
  };

  return (
    <SectionPanel title={t("Closed")} count={sorted.length}>
      {sorted.length === 0 ? (
        <SectionPanelQuiet>{t("No closed checklists yet.")}</SectionPanelQuiet>
      ) : (
        <ScrollArea className="max-h-[26rem]" maskVariant="card">
          <ul className="flex flex-col gap-px px-1.5 py-1">
            {sorted.map((checklist) => {
              const expanded = open.has(checklist.id);
              const regionId = `closed-checklist-${checklist.id}`;
              return (
                <li key={checklist.id} data-testid={`closed-checklist-${checklist.id}`}>
                  <button
                    type="button"
                    aria-expanded={expanded}
                    aria-controls={regionId}
                    onClick={() => toggle(checklist.id)}
                    className="grid w-full grid-cols-[0.875rem_minmax(0,1fr)] items-start gap-2.5 rounded-md ui-focus-ring px-2 py-2 text-left hover:bg-surface-hover"
                  >
                    <ChevronRightIcon
                      aria-hidden
                      className={cn(
                        "mt-0.5 size-3.5 text-foreground-subtle transition-transform duration-200 ease-settle",
                        expanded && "rotate-90",
                      )}
                    />
                    <span className="min-w-0">
                      <span className="block text-sm font-medium">{checklist.name}</span>
                      <span className="block text-xs text-muted-foreground tabular-nums">
                        {closedLine(checklist)}
                      </span>
                    </span>
                  </button>
                  {expanded ? (
                    <ul
                      id={regionId}
                      className="mr-2 mb-2 ml-8 animate-expand-in border-l border-border-subtle py-0.5 motion-reduce:animate-none"
                    >
                      {[...checklist.items]
                        .sort((a, b) => a.sortOrder - b.sortOrder)
                        .map((item) => (
                          <li
                            key={item.id}
                            className="grid grid-cols-[minmax(0,1fr)_auto] gap-2 py-1 pl-2.5 text-xs"
                          >
                            <span className="flex min-w-0 items-start gap-1">
                              {item.status === "Done" ? (
                                <CheckIcon
                                  aria-hidden
                                  className="mt-0.5 size-3 shrink-0 text-success-foreground"
                                />
                              ) : (
                                <MinusIcon
                                  aria-hidden
                                  className="mt-0.5 size-3 shrink-0 text-foreground-subtle"
                                />
                              )}
                              <span className="min-w-0">{item.label}</span>
                            </span>
                            <span className="text-right text-muted-foreground tabular-nums">
                              {itemLine(item)}
                            </span>
                          </li>
                        ))}
                    </ul>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
