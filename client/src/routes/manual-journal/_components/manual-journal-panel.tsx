import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { ManualJournal, ManualJournalLine } from "@/types/manual-journal";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, SendIcon, StampIcon, XIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ManualJournalForm } from "./manual-journal-form";

type JournalFormValues = {
  description: string;
  reason: string;
  accountingDate: string;
  requestedFiscalYearId: string;
  requestedFiscalPeriodId: string;
  currencyCode: string;
  lines: ManualJournalLine[];
};

function makeEmptyLine(lineNumber: number): ManualJournalLine {
  return {
    glAccountId: "",
    description: "",
    debitAmount: 0,
    creditAmount: 0,
    lineNumber,
  } as ManualJournalLine;
}

function CreatePanel({
  open,
  onOpenChange,
}: Pick<DataTablePanelProps<ManualJournal>, "open" | "onOpenChange">) {
  const queryClient = useQueryClient();

  const form = useForm<JournalFormValues>({
    defaultValues: {
      description: "",
      reason: "",
      accountingDate: "",
      requestedFiscalYearId: "",
      requestedFiscalPeriodId: "",
      currencyCode: "USD",
      lines: [makeEmptyLine(1), makeEmptyLine(2)],
    },
  });

  const { mutateAsync, isPending } = useApiMutation<
    ManualJournal,
    JournalFormValues,
    unknown,
    JournalFormValues
  >({
    mutationFn: async (data: JournalFormValues) =>
      apiService.manualJournalService.createDraft(data as unknown as Partial<ManualJournal>),
    setFormError: form.setError,
    resourceName: "manual journal",
    onSuccess: () => {
      toast.success("Draft created");
      void queryClient.invalidateQueries({ queryKey: ["manual-journal-list"] });
      onOpenChange(false);
      form.reset();
    },
  });

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title="New Manual Journal"
      description="Create a new manual journal entry draft."
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="manual-journal-create-form"
            isLoading={isPending}
            disabled={isPending}
          >
            Create Draft
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <Form
          id="manual-journal-create-form"
          onSubmit={form.handleSubmit((data) => mutateAsync(data))}
        >
          <ManualJournalForm isDraft />
        </Form>
      </FormProvider>
    </DataTablePanelContainer>
  );
}

function EditPanel({
  open,
  onOpenChange,
  row,
}: Pick<DataTablePanelProps<ManualJournal>, "open" | "onOpenChange" | "row">) {
  const queryClient = useQueryClient();
  const [rejectReason, setRejectReason] = useState("");
  const [cancelReason, setCancelReason] = useState("");
  const [showRejectInput, setShowRejectInput] = useState(false);
  const [showCancelInput, setShowCancelInput] = useState(false);

  const detailQuery = useQuery({
    ...queries.manualJournal.get(row?.id ?? ""),
    enabled: Boolean(row?.id),
  });

  const journal = detailQuery.data as ManualJournal | undefined;
  const isDraft = journal?.status === "Draft";
  const status = journal?.status;

  const form = useForm<JournalFormValues>({
    values: journal
      ? {
          description: journal.description,
          reason: journal.reason,
          accountingDate: String(journal.accountingDate),
          requestedFiscalYearId: journal.requestedFiscalYearId,
          requestedFiscalPeriodId: journal.requestedFiscalPeriodId,
          currencyCode: journal.currencyCode || "USD",
          lines: journal.lines ?? [],
        }
      : undefined,
  });

  const invalidateQueries = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["manual-journal-list"] });
    void queryClient.invalidateQueries({ queryKey: ["manualJournal"] });
  }, [queryClient]);

  const saveMutation = useApiMutation<ManualJournal, JournalFormValues, unknown, JournalFormValues>(
    {
      mutationFn: async (data: JournalFormValues) =>
        apiService.manualJournalService.updateDraft(
          row!.id!,
          data as unknown as Partial<ManualJournal>,
        ),
      setFormError: form.setError,
      resourceName: "manual journal",
      onSuccess: () => {
        invalidateQueries();
        toast.success("Draft updated");
      },
    },
  );

  const submitMutation = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => apiService.manualJournalService.submit(row!.id!),
    resourceName: "manual journal",
    onSuccess: () => {
      invalidateQueries();
      toast.success("Journal submitted for approval");
    },
  });

  const approveMutation = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => apiService.manualJournalService.approve(row!.id!),
    resourceName: "manual journal",
    onSuccess: () => {
      invalidateQueries();
      toast.success("Journal approved");
    },
  });

  const postMutation = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => apiService.manualJournalService.post(row!.id!),
    resourceName: "manual journal",
    onSuccess: () => {
      invalidateQueries();
      toast.success("Journal posted");
    },
  });

  const rejectMutation = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => apiService.manualJournalService.reject(row!.id!, rejectReason),
    resourceName: "manual journal",
    onSuccess: () => {
      invalidateQueries();
      setShowRejectInput(false);
      toast.success("Journal rejected");
    },
  });

  const cancelMutation = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => apiService.manualJournalService.cancel(row!.id!, cancelReason),
    resourceName: "manual journal",
    onSuccess: () => {
      invalidateQueries();
      setShowCancelInput(false);
      toast.success("Journal cancelled");
    },
  });

  const footer = isDraft ? (
    <>
      <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
        Cancel
      </Button>
      <Button
        type="submit"
        form="manual-journal-edit-form"
        isLoading={saveMutation.isPending}
        disabled={saveMutation.isPending}
      >
        Save Draft
      </Button>
      <Button
        type="button"
        variant="outline"
        onClick={() => submitMutation.mutate(undefined)}
        disabled={submitMutation.isPending}
        isLoading={submitMutation.isPending}
      >
        <SendIcon className="mr-1.5 size-3.5" />
        Submit
      </Button>
    </>
  ) : (
    <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
      Close
    </Button>
  );

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={journal?.requestNumber ? `Journal ${journal.requestNumber}` : "Manual Journal"}
      description="View or manage this manual journal entry."
      headerActions={status ? <AccountingStatusBadge status={status} /> : undefined}
      footer={footer}
    >
      {detailQuery.isLoading ? (
        <div className="space-y-3 p-4">
          <div className="h-8 w-full animate-pulse rounded bg-muted" />
          <div className="h-40 w-full animate-pulse rounded bg-muted" />
        </div>
      ) : journal ? (
        <div className="space-y-4">
          <FormProvider {...form}>
            <Form
              id="manual-journal-edit-form"
              onSubmit={form.handleSubmit((data) => saveMutation.mutate(data))}
            >
              <ManualJournalForm isDraft={isDraft} />
            </Form>
          </FormProvider>

          {journal ? (
            <>
              <Separator />
              <div className="space-y-2">
                <h4 className="text-xs font-semibold">Summary</h4>
                <div className="flex justify-between text-xs">
                  <span className="text-muted-foreground">Total Debit</span>
                  <AmountDisplay value={journal.totalDebit} className="font-medium" />
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-muted-foreground">Total Credit</span>
                  <AmountDisplay value={journal.totalCredit} className="font-medium" />
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-muted-foreground">Difference</span>
                  <AmountDisplay
                    value={journal.totalDebit - journal.totalCredit}
                    variant={journal.totalDebit === journal.totalCredit ? "neutral" : "negative"}
                    className="font-medium"
                  />
                </div>
              </div>
            </>
          ) : null}

          {status === "PendingApproval" ? (
            <>
              <Separator />
              <div className="space-y-3">
                <h4 className="text-xs font-semibold">Actions</h4>
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    className="bg-green-600 text-white hover:bg-green-700"
                    onClick={() => approveMutation.mutate(undefined)}
                    disabled={approveMutation.isPending}
                  >
                    <CheckIcon className="mr-1.5 size-3.5" />
                    Approve
                  </Button>
                  {!showRejectInput ? (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setShowRejectInput(true)}
                    >
                      <XIcon className="mr-1.5 size-3.5" />
                      Reject
                    </Button>
                  ) : (
                    <div className="w-full space-y-2">
                      <textarea
                        value={rejectReason}
                        onChange={(e) => setRejectReason(e.target.value)}
                        placeholder="Rejection reason..."
                        className="w-full rounded-md border bg-background px-3 py-2 text-xs"
                        rows={2}
                      />
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => rejectMutation.mutate(undefined)}
                          disabled={!rejectReason.trim() || rejectMutation.isPending}
                        >
                          Confirm Reject
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setShowRejectInput(false);
                            setRejectReason("");
                          }}
                        >
                          Cancel
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : null}

          {status === "Approved" ? (
            <>
              <Separator />
              <div className="space-y-3">
                <h4 className="text-xs font-semibold">Actions</h4>
                <Button
                  size="sm"
                  onClick={() => postMutation.mutate(undefined)}
                  disabled={postMutation.isPending}
                  isLoading={postMutation.isPending}
                >
                  <StampIcon className="mr-1.5 size-3.5" />
                  Post to GL
                </Button>
              </div>
            </>
          ) : null}

          {status &&
          !showCancelInput &&
          (status === "Draft" || status === "PendingApproval" || status === "Approved") ? (
            <>
              <Separator />
              <Button size="sm" variant="outline" onClick={() => setShowCancelInput(true)}>
                Cancel Journal
              </Button>
            </>
          ) : null}

          {showCancelInput ? (
            <>
              <Separator />
              <div className="space-y-2">
                <textarea
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="Cancel reason..."
                  className="w-full rounded-md border bg-background px-3 py-2 text-xs"
                  rows={2}
                />
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => cancelMutation.mutate(undefined)}
                    disabled={!cancelReason.trim() || cancelMutation.isPending}
                  >
                    Confirm Cancel
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      setShowCancelInput(false);
                      setCancelReason("");
                    }}
                  >
                    Dismiss
                  </Button>
                </div>
              </div>
            </>
          ) : null}
        </div>
      ) : null}
    </DataTablePanelContainer>
  );
}

export function ManualJournalPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ManualJournal>) {
  if (mode === "edit") {
    return <EditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }

  return <CreatePanel open={open} onOpenChange={onOpenChange} />;
}
