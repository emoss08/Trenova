import { approvalFacts, recordCount } from "@/components/desk-chat/conversation/approval-facts";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { AgentIdentityInput } from "@/components/agent-identity/agent-identity";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { deskListRow } from "../list/desk-list";
import { QueueUnseenDot } from "../queue/queue-list";
import type { QueueLeave } from "../queue/use-queue-cursor";
import { batchReview, type BatchPreviewEntry } from "./batch-review-model";
import type { DecisionFlag } from "./decision-gates";

/**
 * What a row is called. Its group already names the kind of change, so the
 * row leads with what is its own: who a message goes to, the record a write
 * touches, or how many when it touches several. Until its preview is read,
 * the change's own title stands in.
 */
export function rowSubject(
  entry: BatchPreviewEntry | undefined,
  fallback: { title: string; summary: string },
  t: TranslateFn,
): { title: string; detail: string } {
  const preview = entry?.preview ?? null;
  if (entry === undefined || preview === null) {
    return { title: fallback.title, detail: fallback.summary };
  }
  const row = batchReview([entry]).rows[0];
  if (row && row.to.length > 0) {
    return { title: row.to.join(", "), detail: row.subject || fallback.summary };
  }
  // The preview's own name for what it touches leads ("S-1001 and 2 more");
  // a count stands in only when it has none.
  if (row && row.title !== "") {
    return { title: row.title, detail: fallback.summary };
  }
  const facts = approvalFacts(preview, t);

  return {
    title: facts.count > 1 ? recordCount(facts.resource, facts.count, t) : fallback.title,
    detail: fallback.summary,
  };
}

/**
 * One waiting change in the list. Its agent's mark gives way to a checkbox
 * when the row is pointed at or changes are being ticked; the two share one
 * box and only cross-fade, so nothing in the row moves. The row's hover is
 * the list's own glide.
 */
export function DecisionRow({
  selected,
  ticked,
  tickable,
  selecting,
  agent,
  title,
  detail,
  flag,
  unseen,
  leaving,
  onOpen,
  onTick,
}: {
  selected: boolean;
  ticked: boolean;
  /** A plan and a held change are decided on their own and take no tick. */
  tickable: boolean;
  /** Some change is ticked, so every row that can be shows its checkbox. */
  selecting: boolean;
  agent: AgentIdentityInput | null;
  title: string;
  detail: ReactNode;
  flag: DecisionFlag;
  unseen: boolean;
  leaving: QueueLeave | null;
  onOpen: () => void;
  onTick: (ticked: boolean) => void;
}) {
  const t = useT();
  const row = deskListRow(selected);
  const boxed = tickable && (ticked || selecting);

  return (
    <div
      data-glide-target={row["data-glide-target"]}
      aria-current={selected ? "true" : undefined}
      className={cn(
        "group/row flex h-11.5 w-full items-center gap-2.5 pr-2.5 pl-2 transition-[opacity,translate] duration-300 ease-out",
        row.className,
        // dk-w-ar is the row's entrance (desk-room.css), which the list turns off once
        // settled. A row on its way out gives it up, so nothing holds it in place.
        leaving === null && "dk-w-ar",
        leaving === "ok" && "translate-x-3 opacity-0",
        leaving === "no" && "-translate-x-3 opacity-0",
      )}
    >
      <span className="relative grid size-5 shrink-0 place-items-center">
        <span
          className={cn(
            "col-start-1 row-start-1 transition-opacity duration-150",
            tickable && "group-focus-within/row:opacity-0 group-hover/row:opacity-0",
            boxed && "opacity-0",
          )}
        >
          <DeskAgentTile agent={agent} size="xs" />
        </span>
        {tickable && (
          <span
            className={cn(
              "col-start-1 row-start-1 grid place-items-center opacity-0 transition-opacity duration-150 group-focus-within/row:opacity-100 group-hover/row:opacity-100",
              boxed && "opacity-100",
            )}
          >
            <Checkbox
              checked={ticked}
              onCheckedChange={(checked) => onTick(checked === true)}
              aria-label={t("Select {0}", title)}
            />
          </span>
        )}
      </span>
      <Button
        variant="bare"
        size="bare"
        className="flex h-full min-w-0 flex-1 flex-col items-stretch justify-center gap-0 text-left"
        onClick={onOpen}
      >
        <b className="truncate text-sm font-medium text-dsk-fg">{title}</b>
        <em className="truncate text-xs font-normal text-dsk-subtle not-italic">{detail}</em>
      </Button>
      {flag !== null && (
        <span
          className={cn(
            "size-1.5 shrink-0 rounded-full",
            flag === "danger" ? "bg-danger" : "bg-warning",
          )}
          title={flag === "danger" ? t("This can't run as it stands") : t("Something to look at")}
        >
          <span className="sr-only">
            {flag === "danger" ? t("This can't run as it stands") : t("Something to look at")}
          </span>
        </span>
      )}
      {unseen && <QueueUnseenDot label={t("Not opened yet")} />}
    </div>
  );
}

const PICK_CLASS =
  "rounded-[5px] px-1.5 py-0.5 text-xs font-normal text-dsk-subtle transition-[opacity,color,background-color] duration-150 hover:bg-dsk-hover hover:text-dsk-fg";

/**
 * A kind's heading in the list: its name, how many wait, and the ways to
 * tick the kind at once. They show when the heading is pointed at and stay
 * while changes are being ticked.
 */
export function DecisionGroupHeading({
  label,
  count,
  selecting,
  clear,
  all,
  onSelectClear,
  onToggleAll,
}: {
  label: string;
  count: number;
  selecting: boolean;
  /** How many of the kind have nothing flagged, when that is some but not all of them. */
  clear: number;
  /** Every change of the kind that can be ticked is. Null when none can be. */
  all: boolean | null;
  onSelectClear: () => void;
  onToggleAll: () => void;
}) {
  const t = useT();
  const shown = selecting
    ? "opacity-100"
    : "opacity-0 group-hover/gh:opacity-100 focus-visible:opacity-100";

  return (
    <span className="group/gh flex min-h-5 w-full items-center gap-1.5">
      {label}
      <i className="text-dsk-subtle not-italic tabular-nums">{count}</i>
      {all !== null && (
        <span className="ml-auto flex items-center gap-0.5">
          {clear > 0 && (
            <Button
              variant="bare"
              size="bare"
              className={cn(PICK_CLASS, shown)}
              onClick={onSelectClear}
            >
              {t("Select {0} with nothing flagged", clear)}
            </Button>
          )}
          <Button
            variant="bare"
            size="bare"
            className={cn(PICK_CLASS, shown)}
            onClick={onToggleAll}
          >
            {all ? t("Clear") : t("Select all")}
          </Button>
        </span>
      )}
    </span>
  );
}
