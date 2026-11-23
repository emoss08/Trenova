import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import type { WorkerChecklistRow } from "@/lib/graphql/worker-checklist";
import { CheckIcon, ChevronRightIcon, MinusIcon } from "@trenova/shared/components/icons";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";
import { ListCapLine, SERVER_LIST_CAP } from "../../list-more";
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
 * stay because an auditor asks, not because anybody reads them day to day. Only the rows near
 * the view are mounted; an opened row is measured again as it grows.
 */
export function ClosedChecklists({
  checklists,
  readCount,
  expandedId,
}: {
  checklists: readonly WorkerChecklistRow[];
  /** How many checklists the read returned, open ones included: at the server's cap, older ones exist. */
  readCount: number;
  /** A closed checklist a link sent the reader to: opened when first drawn. */
  expandedId: string | null;
}) {
  const t = useT();
  const capped = readCount >= SERVER_LIST_CAP;
  const listRef = useRef<VirtualRowsHandle>(null);
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
  const focusShown = expandedId !== null && sorted.some((row) => row.id === expandedId);

  useEffect(() => {
    if (focusShown && expandedId) listRef.current?.scrollToKey(expandedId);
  }, [focusShown, expandedId]);

  const toggle = (id: string) => {
    const next = new Set(open);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setOpen(next);
  };

  const rows: VirtualRow[] = sorted.map((checklist) => ({
    key: checklist.id,
    render: () => (
      <ClosedChecklistRow
        checklist={checklist}
        expanded={open.has(checklist.id)}
        onToggle={() => toggle(checklist.id)}
      />
    ),
  }));

  return (
    <SectionPanel title={t("Closed")} count={capped ? undefined : sorted.length}>
      {sorted.length === 0 ? (
        <SectionPanelQuiet>{t("No closed checklists yet.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={rows}
          aria-label={t("Closed checklists")}
          mask="card"
          className="max-h-104"
          contentClassName="px-1.5 py-1"
          estimateSize={52}
          handleRef={listRef}
          footer={<ListCapLine count={readCount} />}
        />
      )}
    </SectionPanel>
  );
}

function ClosedChecklistRow({
  checklist,
  expanded,
  onToggle,
}: {
  checklist: WorkerChecklistRow;
  expanded: boolean;
  onToggle: () => void;
}) {
  const regionId = `closed-checklist-${checklist.id}`;
  return (
    <div data-testid={`closed-checklist-${checklist.id}`} className="pb-px">
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={regionId}
        onClick={onToggle}
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
    </div>
  );
}
