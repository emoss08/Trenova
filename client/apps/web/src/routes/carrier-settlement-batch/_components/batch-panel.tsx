import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  CarrierSettlementBatchStatusBadge,
  CarrierSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import {
  exportCarrierSettlementBatchCsv,
  fetchCarrierSettlementBatchDetail,
  fetchCurrentCarrierSettlementPeriod,
  generateCarrierSettlementBatch,
  type CarrierSettlementBatchRow,
} from "@/lib/graphql/carrier-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  generateCarrierBatchFormSchema,
  type CarrierSettlementBatchStatus,
  type CarrierSettlementStatus,
  type GenerateCarrierBatchFormValues,
} from "@trenova/shared/types/carrier-settlement";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, type Resolver } from "react-hook-form";
import { Download } from "lucide-react";
import { toast } from "sonner";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix?: number | null): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

export function CarrierBatchPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CarrierSettlementBatchRow>) {
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
  const form = useForm<GenerateCarrierBatchFormValues>({
    resolver: zodResolver(
      generateCarrierBatchFormSchema,
    ) as Resolver<GenerateCarrierBatchFormValues>,
    defaultValues: { name: "", notes: "" },
  });
  const { control } = form;
  const { data: period } = useQuery({
    queryKey: ["current-carrier-settlement-period"],
    queryFn: fetchCurrentCarrierSettlementPeriod,
    enabled: open,
  });

  return (
    <FormCreatePanel<GenerateCarrierBatchFormValues, CarrierSettlementBatchRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Carrier Settlement Batch"
      description="Creates a draft settlement for every carrier with pending cost events in the current period."
      queryKey="carrier-settlement-batch-list"
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
              description="A label for this AP run; if left blank it is named after the period end date."
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
            Settlements can auto-post on approval based on your carrier settlement control policy.
          </p>
        </FormGroup>
      }
      mutationFn={async (values) => {
        const batch = await generateCarrierSettlementBatch({
          name: values.name || undefined,
          notes: values.notes || undefined,
        });
        toast.success(
          `Batch generated: ${batch.settlementCount} settlement${
            batch.settlementCount === 1 ? "" : "s"
          }`,
        );
        void queryClient.invalidateQueries({ queryKey: ["carrier-settlement-list"] });
        return values;
      }}
    />
  );
}

function BatchDetail({ batchId }: { batchId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ["carrier-settlement-batch-detail", batchId],
    queryFn: () => fetchCarrierSettlementBatchDetail(batchId),
  });

  const exportMutation = useMutation({
    mutationFn: () => exportCarrierSettlementBatchCsv(batchId),
    onSuccess: (csv) => {
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `carrier-settlement-batch-${batchId}.csv`;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success("Remittance CSV downloaded");
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
        <CarrierSettlementBatchStatusBadge status={data.status as CarrierSettlementBatchStatus} />
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
          Export Remittance CSV
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-[11px] font-medium text-muted-foreground uppercase">Settlements</p>
          <p className="mt-1 text-sm font-semibold tabular-nums">{data.settlementCount}</p>
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
              <th className="px-3 py-2 font-medium">Carrier</th>
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
                  {settlement.carrier
                    ? settlement.carrier.scac
                      ? `${settlement.carrier.name} (${settlement.carrier.scac})`
                      : settlement.carrier.name
                    : "—"}
                </td>
                <td className="px-3 py-2">
                  <CarrierSettlementStatusBadge
                    status={settlement.status as CarrierSettlementStatus}
                  />
                </td>
                <td className="px-3 py-2 text-right">
                  <AmountDisplay
                    value={settlement.grossCostMinor}
                    currency={settlement.currencyCode}
                  />
                </td>
                <td className="px-3 py-2 text-right font-medium">
                  <AmountDisplay
                    value={settlement.netPayableMinor}
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
