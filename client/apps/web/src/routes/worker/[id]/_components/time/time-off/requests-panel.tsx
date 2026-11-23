import { HoverGlideGroup } from "@/components/hover-glide";
import { formatRange } from "@trenova/shared/lib/date";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { approveWorkerPTO } from "@/lib/graphql/worker-mutations";
import { canApplyPTOAction } from "@/routes/worker/_components/pto/pto-actions";
import { ptoDaysOf } from "@/routes/worker/_components/pto/pto-columns";
import { ptoTypeLabel } from "@/routes/worker/_components/pto/worker-pto-balances";
import { DotsHorizontalIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionEmpty } from "@trenova/shared/components/ui/description-list";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import {
  REQUEST_STATUS_LABELS,
  REQUEST_STATUS_VARIANTS,
  requestDecision,
} from "./request-decision";

export type RequestAction = { kind: "reject" | "cancel"; pto: WorkerPTO };

export type RequestsPanelProps = {
  entries: readonly WorkerPTO[];
  /** Opens one request in the work pane: edit while requested, read once decided. */
  onOpenRequest: (id: string) => void;
  onReason: (action: RequestAction) => void;
  /** Refreshes the balances, the ledger and the roster once a decision lands. */
  onDecided: () => Promise<unknown>;
  /** The row a decision just changed, which takes a brief wash. */
  changed: { id: string; tick: "a" | "b" } | null;
  onChanged: (id: string) => void;
};

const COLUMNS = [
  { label: "Dates" },
  { label: "Type" },
  { label: "Days", numeric: true },
  { label: "Status" },
  { label: "Decision" },
] as const;

/**
 * Every time off request the worker has made or had made for them, newest first, with the
 * decision in the row while one is waiting.
 */
export function RequestsPanel({
  entries,
  onOpenRequest,
  onReason,
  onDecided,
  changed,
  onChanged,
}: RequestsPanelProps) {
  const t = useT();
  const waiting = entries.filter((entry) => entry.status === "Requested").length;

  return (
    <SectionPanel
      title={t("Requests")}
      count={entries.length}
      hint={
        entries.length === 0
          ? undefined
          : waiting > 0
            ? t(
                "{0, plural, one {# waiting on a decision} other {# waiting on a decision}}",
                waiting,
              )
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
        <ScrollArea className="max-h-88" maskVariant="card">
          <HoverGlideGroup>
            <table className="w-full min-w-[35rem] text-sm">
              <thead className="sticky top-0 z-10 bg-card">
                <tr className="border-b text-left text-xs text-foreground-subtle">
                  <th className="px-3 py-2 font-medium">{t("Dates")}</th>
                  <th className="px-3 py-2 font-medium">{t("Type")}</th>
                  <th className="px-3 py-2 text-right font-medium">{t("Days")}</th>
                  <th className="px-3 py-2 font-medium">{t("Status")}</th>
                  <th className="px-3 py-2 font-medium">{t("Decision")}</th>
                  <th className="w-10 px-2 py-2">
                    <span className="sr-only">{t("Actions")}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <RequestRow
                    key={entry.id}
                    entry={entry}
                    onOpenRequest={onOpenRequest}
                    onReason={onReason}
                    onDecided={onDecided}
                    changed={changed && changed.id === entry.id ? changed.tick : null}
                    onChanged={onChanged}
                  />
                ))}
              </tbody>
            </table>
          </HoverGlideGroup>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}

function RequestRow({
  entry,
  onOpenRequest,
  onReason,
  onDecided,
  changed,
  onChanged,
}: {
  entry: WorkerPTO;
  onOpenRequest: (id: string) => void;
  onReason: (action: RequestAction) => void;
  onDecided: () => Promise<unknown>;
  changed: "a" | "b" | null;
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

  return (
    <tr
      data-glide-target
      data-changed={changed ?? undefined}
      className="border-b border-border-subtle last:border-0 data-[changed=a]:[&>td]:animate-row-changed-a data-[changed=b]:[&>td]:animate-row-changed-b"
    >
      <td className="px-3 py-2">
        <button
          type="button"
          onClick={() => onOpenRequest(id)}
          className="rounded-sm ui-focus-ring text-left font-mono text-xs tabular-nums hover:underline"
          aria-label={t("Open the request for {0}", range)}
        >
          {range}
        </button>
      </td>
      <td className="px-3 py-2">{t(ptoTypeLabel(entry.type))}</td>
      <td className="px-3 py-2 text-right font-mono text-xs tabular-nums">
        {String(ptoDaysOf(entry))}
      </td>
      <td className="px-3 py-2">
        <Badge variant={REQUEST_STATUS_VARIANTS[entry.status]} dot>
          {REQUEST_STATUS_LABELS[entry.status]}
        </Badge>
      </td>
      <td className="max-w-56 px-3 py-2">
        {entry.status === "Requested" && (showApprove || showReject) ? (
          <div className="flex gap-1.5">
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
      <td className="px-2 py-2 text-right">
        {showEdit || showCancel ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button size="icon-xs" variant="ghost" aria-label={t("Request options")}>
                  <DotsHorizontalIcon />
                </Button>
              }
            />
            <DropdownMenuContent side="bottom" align="end">
              <DropdownMenuGroup>
                {showEdit ? (
                  <DropdownMenuItem title={t("Edit")} onClick={() => onOpenRequest(id)} />
                ) : null}
                {showCancel ? (
                  <DropdownMenuItem
                    title={t("Cancel request…")}
                    color="danger"
                    onClick={() => onReason({ kind: "cancel", pto: entry })}
                  />
                ) : null}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </td>
    </tr>
  );
}
