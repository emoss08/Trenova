import { AmountDisplay } from "@/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { usePayCodeOptions } from "@/components/fields/pay-code-select-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  fetchSettlementDisputeDetail,
  resolveSettlementDispute,
  startSettlementDisputeReview,
  type SettlementDisputeRow,
} from "@/lib/graphql/driver-portal";
import type { DataTablePanelProps } from "@/types/data-table";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { disputeCategoryLabels, SettlementDisputeStatusBadge } from "./dispute-columns";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function DisputePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<SettlementDisputeRow>) {
  if (mode !== "edit" || !row) {
    return null;
  }

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title="Settlement Dispute"
      description={row.worker ? `${row.worker.firstName} ${row.worker.lastName}`.trim() : undefined}
      size="lg"
    >
      <DisputeDetail disputeId={row.id} onClose={() => onOpenChange(false)} />
    </DataTablePanelContainer>
  );
}

function DisputeDetail({ disputeId, onClose }: { disputeId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const detail = useQuery({
    queryKey: ["settlement-dispute-detail", disputeId],
    queryFn: () => fetchSettlementDisputeDetail(disputeId),
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ["settlement-dispute-detail", disputeId] });
    await queryClient.invalidateQueries({ queryKey: ["settlement-dispute-list"] });
  };

  const startReview = useMutation({
    mutationFn: () => startSettlementDisputeReview(disputeId),
    onSuccess: async () => {
      toast.success("Dispute moved to review");
      await invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to start review"),
  });

  if (detail.isPending) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  const dispute = detail.data;
  if (!dispute) {
    return (
      <p className="p-6 text-center text-sm text-muted-foreground">
        This dispute could not be loaded.
      </p>
    );
  }

  const isTerminal =
    dispute.status === "Resolved" || dispute.status === "Denied" || dispute.status === "Withdrawn";

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="rounded-lg border border-border bg-muted/40 p-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold">
            {disputeCategoryLabels[dispute.category] ?? dispute.category}
          </p>
          <SettlementDisputeStatusBadge status={dispute.status} />
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          Submitted {formatDate(dispute.createdAt)}
          {dispute.worker
            ? ` by ${`${dispute.worker.firstName} ${dispute.worker.lastName}`.trim()}`
            : ""}
        </p>
        <p className="mt-3 text-sm whitespace-pre-wrap">{dispute.description}</p>
      </div>

      {dispute.settlement ? (
        <div className="rounded-lg border border-border p-3">
          <p className="text-xs font-medium text-muted-foreground uppercase">Settlement</p>
          <div className="mt-1 flex items-center justify-between text-sm">
            <span className="font-mono font-medium">{dispute.settlement.settlementNumber}</span>
            <span className="tabular-nums">
              Net{" "}
              <AmountDisplay
                value={dispute.settlement.netPayMinor}
                currency={dispute.settlement.currencyCode}
              />
            </span>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatDate(dispute.settlement.periodStart)} –{" "}
            {formatDate(dispute.settlement.periodEnd)}
          </p>
          {dispute.settlementLine ? (
            <>
              <Separator className="my-2" />
              <p className="text-xs font-medium text-muted-foreground uppercase">Disputed line</p>
              <div className="mt-1 flex items-center justify-between text-sm">
                <span>{dispute.settlementLine.description}</span>
                <AmountDisplay value={dispute.settlementLine.amountMinor} />
              </div>
            </>
          ) : null}
        </div>
      ) : null}

      {isTerminal ? (
        <div className="rounded-lg border border-border p-3">
          <p className="text-xs font-medium text-muted-foreground uppercase">Resolution</p>
          <p className="mt-1 text-sm whitespace-pre-wrap">{dispute.resolutionNote || "—"}</p>
          <p className="mt-2 text-xs text-muted-foreground">
            {formatDate(dispute.resolvedAt)}
            {dispute.resolvedBy ? ` by ${dispute.resolvedBy.name}` : ""}
            {dispute.resolutionLineId ? " · correcting adjustment applied" : ""}
          </p>
        </div>
      ) : (
        <>
          {dispute.status === "Open" ? (
            <Button
              variant="outline"
              onClick={() => startReview.mutate()}
              disabled={startReview.isPending}
            >
              {startReview.isPending ? "Updating..." : "Start review"}
            </Button>
          ) : null}
          <ResolveForm disputeId={disputeId} onDone={invalidate} onClose={onClose} />
        </>
      )}
    </div>
  );
}

function ResolveForm({
  disputeId,
  onDone,
  onClose,
}: {
  disputeId: string;
  onDone: () => Promise<void>;
  onClose: () => void;
}) {
  const [approve, setApprove] = useState(true);
  const [resolutionNote, setResolutionNote] = useState("");
  const [withAdjustment, setWithAdjustment] = useState(false);
  const [adjustmentDescription, setAdjustmentDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [payCodeId, setPayCodeId] = useState("none");
  const { data: payCodes } = usePayCodeOptions();

  const parsedAmount = Number(amount);
  const adjustmentValid =
    !withAdjustment ||
    (adjustmentDescription.trim() !== "" && !Number.isNaN(parsedAmount) && parsedAmount !== 0);
  const valid = resolutionNote.trim() !== "" && adjustmentValid;

  const mutation = useMutation({
    mutationFn: () =>
      resolveSettlementDispute({
        disputeId,
        approve,
        resolutionNote: resolutionNote.trim(),
        adjustment:
          approve && withAdjustment
            ? {
                description: adjustmentDescription.trim(),
                amountMinor: Math.round(parsedAmount * 100),
                payCodeId: payCodeId === "none" ? undefined : payCodeId,
              }
            : undefined,
      }),
    onSuccess: async () => {
      toast.success(approve ? "Dispute resolved" : "Dispute denied");
      await onDone();
      onClose();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to resolve dispute"),
  });

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-border p-3">
      <p className="text-sm font-semibold">Resolve this dispute</p>

      <div className="grid grid-cols-2 gap-2">
        <Button
          type="button"
          variant={approve ? "default" : "outline"}
          onClick={() => setApprove(true)}
        >
          Resolve in driver&apos;s favor
        </Button>
        <Button
          type="button"
          variant={approve ? "outline" : "default"}
          onClick={() => {
            setApprove(false);
            setWithAdjustment(false);
          }}
        >
          Deny
        </Button>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="dispute-resolution-note">Resolution note</Label>
        <Textarea
          id="dispute-resolution-note"
          value={resolutionNote}
          onChange={(event) => setResolutionNote(event.target.value)}
          placeholder="Explain the outcome — the driver sees this in Dash."
          rows={3}
        />
        <p className="text-[11px] text-muted-foreground">
          Shown to the driver verbatim; say what was checked and why the outcome is right.
        </p>
      </div>

      {approve ? (
        <div className="flex items-center justify-between">
          <div>
            <Label htmlFor="dispute-with-adjustment">Apply a correcting adjustment</Label>
            <p className="text-[11px] text-muted-foreground">
              Adds a line to the driver&apos;s open settlement (an off-cycle draft is created if
              none exists).
            </p>
          </div>
          <Switch
            id="dispute-with-adjustment"
            checked={withAdjustment}
            onCheckedChange={setWithAdjustment}
          />
        </div>
      ) : null}

      {approve && withAdjustment ? (
        <div className="flex flex-col gap-3">
          <div>
            <Input
              value={adjustmentDescription}
              onChange={(event) => setAdjustmentDescription(event.target.value)}
              placeholder="Description (e.g. Detention correction - PRO 12345)"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Appears as the line item on the driver&apos;s statement.
            </p>
          </div>
          <div>
            <Input
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              placeholder="Amount (e.g. 150.00 or -75.00)"
              inputMode="decimal"
            />
            <p className="mt-1 text-[11px] text-muted-foreground">
              Dollars, not cents; positive adds pay, negative deducts.
            </p>
          </div>
          <div>
            <Select
              value={payCodeId}
              items={[
                { label: "No pay code", value: "none" },
                ...(payCodes ?? []).map((code) => ({
                  label: `${code.code} — ${code.name} (${code.direction})`,
                  value: code.id,
                })),
              ]}
              onValueChange={(value) => setPayCodeId(value ?? "none")}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Pay code (optional)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">No pay code</SelectItem>
                {(payCodes ?? []).map((code) => (
                  <SelectItem key={code.id} value={code.id}>
                    {code.code} — {code.name} ({code.direction})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Optional — routes the adjustment to that code&apos;s GL account when posting.
            </p>
          </div>
        </div>
      ) : null}

      <Button onClick={() => mutation.mutate()} disabled={!valid || mutation.isPending}>
        {mutation.isPending ? "Saving..." : approve ? "Resolve dispute" : "Deny dispute"}
      </Button>
    </div>
  );
}
