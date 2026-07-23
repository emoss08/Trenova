import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { DriverSettlementStatusBadge, SettlementBatchStatusBadge } from "@/components/status-badge";
import {
  exportSettlementBatchCsv,
  fetchCurrentSettlementPeriod,
  generateSettlementBatch,
  type SettlementBatchRow,
} from "@/lib/graphql/driver-settlement";
import { requestGraphQL } from "@/lib/graphql";
import { SettlementBatchDetailDocument } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@/types/data-table";
import type { DriverSettlementStatus, SettlementBatchStatus } from "@/types/driver-pay";
import { generateBatchFormSchema, type GenerateBatchFormValues } from "@/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, type Resolver } from "react-hook-form";
import { Download, TriangleAlert } from "lucide-react";
import { toast } from "sonner";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function BatchPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<SettlementBatchRow>) {
  if (mode === "edit" && row) {
    return (
      <DataTablePanelContainer open={open} onOpenChange={onOpenChange} title={row.name} size="xl">
        <BatchDetail batchId={row.id} />
      </DataTablePanelContainer>
    );
  }

  return <GenerateBatchPanel open={open} onOpenChange={onOpenChange} />;
}

function GenerateBatchPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const form = useForm<GenerateBatchFormValues>({
    resolver: zodResolver(generateBatchFormSchema) as Resolver<GenerateBatchFormValues>,
    defaultValues: { name: "", notes: "" },
  });
  const { control } = form;
  const { data: period } = useQuery({
    queryKey: ["current-settlement-period"],
    queryFn: fetchCurrentSettlementPeriod,
    enabled: open,
  });

  return (
    <FormCreatePanel<GenerateBatchFormValues, SettlementBatchRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Settlement Batch"
      description="Creates a draft settlement for every driver with accrued pay in the current period."
      queryKey="settlement-batch-list"
      form={form}
      notice={
        period ? (
          <div className="rounded-lg border bg-muted/30 p-3 text-sm">
            <p className="text-[11px] font-medium text-muted-foreground uppercase">
              Current Pay Period
            </p>
            <p className="mt-1 font-medium">
              {formatDate(period.periodStart)} – {formatDate(period.periodEnd)}
            </p>
            <p className="text-xs text-muted-foreground">Pays on {formatDate(period.payDate)}</p>
          </div>
        ) : undefined
      }
      formComponent={
        <FormGroup cols={1} className="pt-2">
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Batch Name"
              placeholder="Defaults to the pay period end date"
              description="A label for this payroll run; if left blank it is named after the period end date."
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="notes"
              label="Notes"
              description="Anything reviewers should know about this run, e.g. an off-cycle correction."
            />
          </FormControl>
          <p className="text-xs text-muted-foreground">
            Clean settlements can auto-approve based on your settlement control policy; anything
            with exceptions stays in review.
          </p>
        </FormGroup>
      }
      mutationFn={async (values) => {
        const batch = await generateSettlementBatch({
          name: values.name || undefined,
          notes: values.notes || undefined,
        });
        toast.success(
          `Batch generated: ${batch.settlementCount} settlement${
            batch.settlementCount === 1 ? "" : "s"
          }${batch.exceptionCount > 0 ? `, ${batch.exceptionCount} need review` : ""}`,
        );
        void queryClient.invalidateQueries({ queryKey: ["driver-settlement-list"] });
        return values;
      }}
    />
  );
}

function BatchDetail({ batchId }: { batchId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ["settlement-batch-detail", batchId],
    queryFn: async () => {
      const result = await requestGraphQL({
        document: SettlementBatchDetailDocument,
        operationName: "SettlementBatchDetail",
        variables: { id: batchId },
      });
      return result.settlementBatch;
    },
  });

  const exportMutation = useMutation({
    mutationFn: () => exportSettlementBatchCsv(batchId),
    onSuccess: (csv) => {
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `settlement-batch-${batchId}.csv`;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success("Payroll CSV downloaded");
    },
    onError: (error: Error) => toast.error(error.message || "Export failed"),
  });

  if (isLoading || !data) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto">
      <div className="flex flex-wrap items-center gap-2">
        <SettlementBatchStatusBadge status={data.status as SettlementBatchStatus} />
        <span className="text-xs text-muted-foreground">
          {formatDate(data.periodStart)} – {formatDate(data.periodEnd)} · pays{" "}
          {formatDate(data.payDate)}
        </span>
        <Button
          size="sm"
          variant="outline"
          className="ml-auto"
          disabled={exportMutation.isPending}
          onClick={() => exportMutation.mutate()}
        >
          <Download className="size-3.5" />
          Export Payroll CSV
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Settlements</p>
          <p className="mt-1 text-sm font-semibold tabular-nums">{data.settlementCount}</p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Exceptions</p>
          <p className="mt-1 flex items-center gap-1 text-sm font-semibold tabular-nums">
            {data.exceptionCount > 0 && <TriangleAlert className="size-3.5 text-amber-500" />}
            {data.exceptionCount}
          </p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Total Gross</p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay value={data.totalGrossMinor} currency={data.currencyCode} />
          </p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Total Net</p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay
              value={data.totalNetMinor}
              variant="positive"
              currency={data.currencyCode}
            />
          </p>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-xs">
          <thead className="bg-muted/50 text-left">
            <tr>
              <th className="px-3 py-2 font-medium">Settlement</th>
              <th className="px-3 py-2 font-medium">Driver</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="px-3 py-2 text-right font-medium">Gross</th>
              <th className="px-3 py-2 text-right font-medium">Net</th>
            </tr>
          </thead>
          <tbody>
            {(data.settlements ?? []).map((settlement) => (
              <tr key={settlement.id} className="border-t">
                <td className="px-3 py-2 font-mono font-medium">{settlement.settlementNumber}</td>
                <td className="px-3 py-2">
                  {settlement.worker
                    ? `${settlement.worker.firstName} ${settlement.worker.lastName}`.trim()
                    : "—"}
                </td>
                <td className="px-3 py-2">
                  <div className="flex items-center gap-1.5">
                    <DriverSettlementStatusBadge
                      status={settlement.status as DriverSettlementStatus}
                    />
                    {settlement.hasExceptions && (
                      <TriangleAlert className="size-3 text-amber-500" />
                    )}
                  </div>
                </td>
                <td className="px-3 py-2 text-right">
                  <AmountDisplay
                    value={settlement.grossEarningsMinor}
                    currency={settlement.currencyCode}
                  />
                </td>
                <td className="px-3 py-2 text-right font-medium">
                  <AmountDisplay
                    value={settlement.netPayMinor}
                    variant="positive"
                    currency={settlement.currencyCode}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
