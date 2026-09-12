import { useT } from "@trenova/shared/i18n/use-t";
import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { PayAdvanceStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { payAdvanceSourceChoices } from "@/lib/choices";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  issuePayAdvance,
  writeOffPayAdvance,
  type PayAdvanceRow,
} from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  issuePayAdvanceFormSchema,
  type IssuePayAdvanceFormValues,
  type PayAdvanceStatus,
} from "@trenova/shared/types/driver-pay";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

function buildDefaults(): IssuePayAdvanceFormValues {
  return {
    workerId: "",
    source: "EFSMoneyCode",
    reference: "",
    issuedDate: getTodayDate(),
    amount: 0,
    notes: "",
  };
}

export function AdvancePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<PayAdvanceRow>) {
  if (mode === "edit" && row) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={`Advance ${row.reference || ""}`.trim()}
        size="md"
      >
        <AdvanceDetail row={row} onClose={() => onOpenChange(false)} />
      </DataTablePanelContainer>
    );
  }

  return <IssueAdvancePanel open={open} onOpenChange={onOpenChange} />;
}

function IssueAdvancePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<IssuePayAdvanceFormValues>({
    resolver: zodResolver(issuePayAdvanceFormSchema) as Resolver<IssuePayAdvanceFormValues>,
    defaultValues: buildDefaults(),
  });
  const { control } = form;

  return (
    <FormCreatePanel<IssuePayAdvanceFormValues, PayAdvanceRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Pay Advance")}
      description={t(
        "Records a cash or money-code advance recovered from the driver's future settlements.",
      )}
      queryKey="pay-advance-list"
      form={form}
      formComponent={
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <WorkerAutocompleteField
              control={control}
              name="workerId"
              label={t("Driver")}
              placeholder={t("Select driver")}
              rules={{ required: true }}
              description={t(
                "The driver receiving the advance; recovery is deducted from their next settlement.",
              )}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="source"
              label={t("Source")}
              options={payAdvanceSourceChoices}
              rules={{ required: true }}
              description={t(
                "How the money was disbursed — cash, an EFS/Comdata money code, or a fuel card load.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="reference"
              label={t("Reference")}
              placeholder={t("Money code / check number")}
              description={t(
                "The money-code or check number so the advance can be matched to the card statement.",
              )}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="issuedDate"
              label={t("Issued Date")}
              rules={{ required: true }}
              description={t("The date the driver actually received the funds.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="amount"
              label={t("Amount")}
              decimalScale={2}
              fixedDecimalScale
              sideText={t("USD")}
              rules={{ required: true }}
              description={t(
                "The full amount advanced; it is recovered automatically from upcoming settlements.",
              )}
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              description={t(
                "Context for reviewers — what the advance covered, e.g. a lumper fee or breakdown repair.",
              )}
            />
          </FormControl>
        </FormGroup>
      }
      mutationFn={async (values) => {
        await issuePayAdvance({
          workerId: values.workerId,
          source: values.source,
          reference: values.reference || undefined,
          issuedDate: values.issuedDate,
          amountMinor: Math.round(values.amount * 100),
          notes: values.notes || undefined,
        });
        return values;
      }}
    />
  );
}

function AdvanceDetail({ row, onClose }: { row: PayAdvanceRow; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [writeOffOpen, setWriteOffOpen] = useState(false);
  const [reason, setReason] = useState("");

  const writeOffMutation = useMutation({
    mutationFn: () => writeOffPayAdvance({ advanceId: row.id, reason }),
    onSuccess: () => {
      toast.success(t("Advance written off"));
      void queryClient.invalidateQueries({ queryKey: ["pay-advance-list"] });
      setWriteOffOpen(false);
      onClose();
    },
    onError: (error: Error) => toast.error(error.message || "Write-off failed"),
  });

  const canWriteOff = row.status === "Outstanding" || row.status === "PartiallyRecovered";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <PayAdvanceStatusBadge status={row.status as PayAdvanceStatus} />
        <span className="text-muted-foreground text-xs">
          {row.worker ? `${row.worker.firstName} ${row.worker.lastName}`.trim() : ""}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-2">
        <div className="bg-muted/30 rounded-lg border p-3">
          <p className="text-muted-foreground text-[11px] font-medium uppercase">{t("Amount")}</p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay value={row.amountMinor} currency={row.currencyCode} />
          </p>
        </div>
        <div className="bg-muted/30 rounded-lg border p-3">
          <p className="text-muted-foreground text-[11px] font-medium uppercase">
            {t("Recovered")}
          </p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay value={row.recoveredMinor} currency={row.currencyCode} />
          </p>
        </div>
        <div className="bg-muted/30 rounded-lg border p-3">
          <p className="text-muted-foreground text-[11px] font-medium uppercase">
            {t("Outstanding")}
          </p>
          <p className="mt-1 text-sm font-semibold">
            <AmountDisplay
              value={row.outstandingMinor}
              variant={row.outstandingMinor > 0 ? "negative" : "neutral"}
              currency={row.currencyCode}
            />
          </p>
        </div>
      </div>
      {row.notes && <p className="text-muted-foreground text-xs">{row.notes}</p>}
      {row.writeOffReason && (
        <p className="text-xs text-red-600 dark:text-red-400">
          {t("Write-off reason: {0}", row.writeOffReason)}
        </p>
      )}
      {canWriteOff && (
        <div>
          <Button size="sm" variant="outline" onClick={() => setWriteOffOpen(true)}>
            {t("Write Off Remaining Balance")}
          </Button>
        </div>
      )}

      <Dialog open={writeOffOpen} onOpenChange={setWriteOffOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("Write off advance")}</DialogTitle>
            <DialogDescription>
              {t("The outstanding balance will no longer be recovered from future settlements.")}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={t("Reason (required)")}
            rows={3}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setWriteOffOpen(false)}>
              {t("Cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={!reason.trim() || writeOffMutation.isPending}
              onClick={() => writeOffMutation.mutate()}
            >
              {t("Write Off")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
