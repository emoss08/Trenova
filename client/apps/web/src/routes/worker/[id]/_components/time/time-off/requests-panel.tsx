import { SectionPanel } from "@/components/section-panel";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { preloadOn } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { approveWorkerPTO } from "@/lib/graphql/worker-mutations";
import { canApplyPTOAction } from "@/routes/worker/_components/pto/pto-actions";
import { ptoDaysOf } from "@/routes/worker/_components/pto/pto-columns";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionEmpty } from "@trenova/shared/components/ui/description-list";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRange } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import type { KeyboardEvent } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedMore,
  type BoundedRowProps,
} from "../../bounded-table";
import { ROW_CHANGED_CLASS, type FlashPhase } from "../../use-row-flash";
import {
  REQUEST_STATUS_LABELS,
  REQUEST_STATUS_VARIANTS,
  requestDecision,
} from "./request-decision";

export type RequestAction = { kind: "reject" | "cancel"; pto: WorkerPTO };

export type RequestsPanelProps = {
  /** The requests read so far, latest start first. */
  entries: readonly WorkerPTO[];
  /** Every request on file, from the first page's count; null until it is known. */
  total: number | null;
  /** The pages after the first, read as the table nears its end. */
  more: BoundedMore;
  /** Opens one request in the work pane: edit while requested, read once decided. */
  onOpenRequest: (id: string) => void;
  /** The pane a row opens, preloaded when the row is pointed at or focused. */
  requestPane: { preload: () => Promise<unknown> };
  onReason: (action: RequestAction) => void;
  /** Refreshes the balances, the ledger and the roster once a decision lands. */
  onDecided: () => Promise<unknown>;
  /** The wash a row takes after a decision changed it. */
  flashOf: (id: string) => FlashPhase | undefined;
  onChanged: (id: string) => void;
};

const COLUMNS = [
  { label: "Dates" },
  { label: "Type" },
  { label: "Days", numeric: true },
  { label: "Status" },
  { label: "Decision" },
] as const;

function requestColumns(t: TranslateFn): BoundedColumn[] {
  return [
    { key: "dates", label: t("Dates") },
    { key: "type", label: t("Type") },
    { key: "days", label: t("Days"), align: "end" },
    { key: "status", label: t("Status") },
    { key: "decision", label: t("Decision") },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];
}

/**
 * Every time off request the worker has made or had made for them, latest first, with the
 * decision in the row while one is waiting. It reads a page at a time as the reader scrolls to
 * the end, and mounts only the rows in view.
 */
export function RequestsPanel({
  entries,
  total,
  more,
  onOpenRequest,
  requestPane,
  onReason,
  onDecided,
  flashOf,
  onChanged,
}: RequestsPanelProps) {
  const t = useT();
  // Only a whole list can say how many wait; a partial one would undercount.
  const waiting = more.hasMore
    ? null
    : entries.filter((entry) => entry.status === "Requested").length;

  return (
    <SectionPanel
      title={t("Requests")}
      count={total ?? undefined}
      hint={
        entries.length === 0 || waiting === null
          ? undefined
          : waiting > 0
            ? t("{0} waiting on a decision", waiting)
            : t("Nothing waiting")
      }
    >
      {entries.length === 0 ? (
        <EmptyTable
          title={t("No time off on record")}
          description={t(
            "Requests they make in Dash, and any recorded here, show up in this list.",
          )}
          columns={COLUMNS}
        />
      ) : (
        <BoundedTable
          label={t("Requests")}
          columns={requestColumns(t)}
          maxHeightClassName="max-h-88"
          estimateRowHeight={45}
          rows={entries.map((entry) => ({
            key: entry.id ?? "",
            render: (props) => (
              <RequestRow
                props={props}
                entry={entry}
                onOpenRequest={onOpenRequest}
                requestPane={requestPane}
                onReason={onReason}
                onDecided={onDecided}
                changed={flashOf(entry.id ?? "")}
                onChanged={onChanged}
              />
            ),
          }))}
          more={more}
        />
      )}
    </SectionPanel>
  );
}

function RequestRow({
  props,
  entry,
  onOpenRequest,
  requestPane,
  onReason,
  onDecided,
  changed,
  onChanged,
}: {
  props: BoundedRowProps;
  entry: WorkerPTO;
  onOpenRequest: (id: string) => void;
  requestPane: { preload: () => Promise<unknown> };
  onReason: (action: RequestAction) => void;
  onDecided: () => Promise<unknown>;
  changed: FlashPhase | undefined;
  onChanged: (id: string) => void;
}) {
  const t = useT();
  const { allowed: canApprove } = usePermission(Resource.WorkerPTO, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.WorkerPTO, Operation.Reject);
  const { allowed: canCancel } = usePermission(Resource.WorkerPTO, Operation.Cancel);
  const { allowed: canUpdate } = usePermission(Resource.WorkerPTO, Operation.Update);
  const id = entry.id ?? "";
  const approve = useApiMutation({
    resourceName: "PTO",
    scope: { id: `worker-pto-approve:${id}` },
    mutationFn: () => approveWorkerPTO(id),
    onSuccess: async () => {
      onChanged(id);
      await onDecided();
    },
  });
  const showApprove = canApprove && canApplyPTOAction(entry.status, "Approve");
  const showReject = canReject && canApplyPTOAction(entry.status, "Reject");
  const showCancel = canCancel && canApplyPTOAction(entry.status, "Cancel");
  const showEdit = canUpdate && entry.status === "Requested";
  const decision = requestDecision(t, entry);
  const range = formatRange(entry.startDate, entry.endDate, { year: "always", join: "words" });
  const actions: RowAction[] = [
    ...(showEdit
      ? [
          {
            id: "edit",
            label: t("Edit"),
            preload: requestPane.preload,
            onSelect: () => onOpenRequest(id),
          },
        ]
      : []),
    ...(showCancel
      ? [
          {
            id: "cancel",
            label: t("Cancel request…"),
            destructive: true,
            onSelect: () => onReason({ kind: "cancel", pto: entry }),
          },
        ]
      : []),
  ];
  const openOnKey = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.target !== event.currentTarget) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onOpenRequest(id);
    }
  };

  return (
    <tr
      {...props}
      {...preloadOn(requestPane)}
      data-glide-target
      data-changed={changed}
      tabIndex={0}
      aria-label={t("Open the request for {0}", range)}
      onClick={() => onOpenRequest(id)}
      onKeyDown={openOnKey}
      className={cn("cursor-pointer ui-focus-ring", ROW_CHANGED_CLASS)}
    >
      <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs tabular-nums")}>{range}</td>
      <td className={BOUNDED_CELL_CLASS}>{t(ptoTypeLabel(entry.type))}</td>
      <td className={cn(BOUNDED_CELL_CLASS, "text-right font-mono text-xs tabular-nums")}>
        {String(ptoDaysOf(entry))}
      </td>
      <td className={BOUNDED_CELL_CLASS}>
        <Badge variant={REQUEST_STATUS_VARIANTS[entry.status]} dot>
          {REQUEST_STATUS_LABELS[entry.status]}
        </Badge>
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "max-w-56")}>
        {entry.status === "Requested" && (showApprove || showReject) ? (
          <div
            className="flex gap-1.5"
            onClick={(event) => event.stopPropagation()}
            onKeyDown={(event) => event.stopPropagation()}
          >
            {showApprove ? (
              <Button
                size="xs"
                isLoading={approve.isPending}
                onClick={() => void approve.mutateAsync().catch(() => undefined)}
              >
                {t("Approve")}
              </Button>
            ) : null}
            {showReject ? (
              <Button
                size="xs"
                variant="outline"
                disabled={approve.isPending}
                onClick={() => onReason({ kind: "reject", pto: entry })}
              >
                {t("Reject")}
              </Button>
            ) : null}
          </div>
        ) : decision.kind === "waiting" ? (
          <span className="text-muted-foreground">{t("Waiting")}</span>
        ) : (
          <span data-testid="pto-decision" className="block min-w-0">
            <span className="block truncate">{decision.actor ?? <DescriptionEmpty />}</span>
            {decision.note ? (
              <span className="block truncate text-xs text-muted-foreground" title={decision.note}>
                {decision.note}
              </span>
            ) : null}
          </span>
        )}
      </td>
      <td
        className={cn(BOUNDED_CELL_CLASS, "text-right")}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        <RowActionsMenu label={t("Request options")} actions={actions} />
      </td>
    </tr>
  );
}
