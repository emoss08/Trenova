import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { JournalReversal, JournalReversalStatus } from "@/types/journal-reversal";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon, SendIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { JournalReversalForm } from "./journal-reversal-form";

type CreateReversalForm = {
  originalJournalEntryId: string;
  requestedAccountingDate: string;
  reasonCode: string;
  reasonText: string;
};

const CANCELLABLE_STATUSES: JournalReversalStatus[] = ["Requested", "PendingApproval", "Approved"];

function CreateReversalPanel({
  open,
  onOpenChange,
}: Pick<DataTablePanelProps<JournalReversal>, "open" | "onOpenChange">) {
  const queryClient = useQueryClient();

  const form = useForm<CreateReversalForm>({
    defaultValues: {
      originalJournalEntryId: "",
      requestedAccountingDate: "",
      reasonCode: "",
      reasonText: "",
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open) {
      reset();
    }
  }, [open, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: CreateReversalForm) => {
      return apiService.journalReversalService.create({
        originalJournalEntryId: values.originalJournalEntryId,
        requestedAccountingDate: new Date(values.requestedAccountingDate).getTime() / 1000,
        reasonCode: values.reasonCode,
        reasonText: values.reasonText,
      });
    },
    onSuccess: () => {
      toast.success("Reversal request created");
      void queryClient.invalidateQueries({ queryKey: ["journal-reversal-list"] });
      onOpenChange(false);
      reset();
    },
    setFormError: setError,
    resourceName: "Journal Reversal",
  });

  const onSubmit = handleSubmit(async (values) => {
    await mutateAsync(values);
  });

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title="New Journal Reversal"
      description="Create a new journal entry reversal request."
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="journal-reversal-create-form" disabled={isSubmitting}>
            <SendIcon className="mr-1.5 size-3.5" />
            {isSubmitting ? "Creating..." : "Create Reversal"}
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <Form id="journal-reversal-create-form" onSubmit={onSubmit}>
          <JournalReversalForm />
        </Form>
      </FormProvider>
    </DataTablePanelContainer>
  );
}

function ReversalDetailPanel({
  open,
  onOpenChange,
  row,
}: Pick<DataTablePanelProps<JournalReversal>, "open" | "onOpenChange" | "row">) {
  const queryClient = useQueryClient();
  const [rejectionReason, setRejectionReason] = useState("");
  const [cancelReason, setCancelReason] = useState("");
  const [showRejectInput, setShowRejectInput] = useState(false);
  const [showCancelInput, setShowCancelInput] = useState(false);

  const reversal = row;

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["journal-reversal-list"] });
    if (reversal?.id) {
      await queryClient.invalidateQueries({
        queryKey: queries.journalReversal.get(reversal.id).queryKey,
      });
    }
  }, [queryClient, reversal?.id]);

  const { mutateAsync: approve, isPending: isApproving } = useApiMutation({
    mutationFn: () => apiService.journalReversalService.approve(reversal!.id!),
    onSuccess: () => {
      toast.success("Reversal approved");
      void invalidate();
    },
    resourceName: "Journal Reversal",
  });

  const { mutateAsync: postReversal, isPending: isPosting } = useApiMutation({
    mutationFn: () => apiService.journalReversalService.post(reversal!.id!),
    onSuccess: () => {
      toast.success("Reversal posted");
      void invalidate();
    },
    resourceName: "Journal Reversal",
  });

  const { mutateAsync: reject, isPending: isRejecting } = useApiMutation({
    mutationFn: () =>
      apiService.journalReversalService.reject(reversal!.id!, rejectionReason),
    onSuccess: () => {
      toast.success("Reversal rejected");
      setShowRejectInput(false);
      setRejectionReason("");
      void invalidate();
    },
    resourceName: "Journal Reversal",
  });

  const { mutateAsync: cancel, isPending: isCancelling } = useApiMutation({
    mutationFn: () =>
      apiService.journalReversalService.cancel(reversal!.id!, cancelReason),
    onSuccess: () => {
      toast.success("Reversal cancelled");
      setShowCancelInput(false);
      setCancelReason("");
      void invalidate();
    },
    resourceName: "Journal Reversal",
  });

  if (!reversal) return null;

  const canCancel = CANCELLABLE_STATUSES.includes(reversal.status);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title="Journal Reversal"
      description={`Reversal for entry ${reversal.originalJournalEntryId}`}
      headerActions={<AccountingStatusBadge status={reversal.status} />}
    >
      <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
        <div>
          <dt className="text-2xs font-medium text-muted-foreground">Original Entry</dt>
          <dd className="mt-0.5 font-mono text-xs">{reversal.originalJournalEntryId}</dd>
        </div>
        <div>
          <dt className="text-2xs font-medium text-muted-foreground">Reason Code</dt>
          <dd className="mt-0.5 text-xs font-medium">{reversal.reasonCode}</dd>
        </div>
        <div className="col-span-2">
          <dt className="text-2xs font-medium text-muted-foreground">Reason</dt>
          <dd className="mt-0.5 text-xs">{reversal.reasonText}</dd>
        </div>
        {reversal.reversalJournalEntryId ? (
          <div>
            <dt className="text-2xs font-medium text-muted-foreground">Reversal Entry</dt>
            <dd className="mt-0.5 font-mono text-xs">{reversal.reversalJournalEntryId}</dd>
          </div>
        ) : null}
        {reversal.rejectionReason ? (
          <div className="col-span-2">
            <dt className="text-2xs font-medium text-muted-foreground">Rejection Reason</dt>
            <dd className="mt-0.5 text-xs text-red-600 dark:text-red-400">
              {reversal.rejectionReason}
            </dd>
          </div>
        ) : null}
        {reversal.cancelReason ? (
          <div className="col-span-2">
            <dt className="text-2xs font-medium text-muted-foreground">Cancel Reason</dt>
            <dd className="mt-0.5 text-xs">{reversal.cancelReason}</dd>
          </div>
        ) : null}
      </dl>

      {reversal.status === "PendingApproval" ||
      reversal.status === "Approved" ||
      canCancel ? (
        <>
          <Separator />
          <div className="space-y-3">
            <h4 className="text-xs font-semibold">Actions</h4>
            <div className="flex flex-wrap items-center gap-2">
              {reversal.status === "PendingApproval" ? (
                <>
                  <Button
                    size="sm"
                    onClick={() => void approve(undefined)}
                    disabled={isApproving}
                  >
                    <CheckIcon className="mr-1.5 size-3.5" />
                    {isApproving ? "Approving..." : "Approve"}
                  </Button>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => setShowRejectInput(true)}
                    disabled={showRejectInput}
                  >
                    <XIcon className="mr-1.5 size-3.5" />
                    Reject
                  </Button>
                </>
              ) : null}
              {reversal.status === "Approved" ? (
                <Button
                  size="sm"
                  onClick={() => void postReversal(undefined)}
                  disabled={isPosting}
                >
                  <SendIcon className="mr-1.5 size-3.5" />
                  {isPosting ? "Posting..." : "Post"}
                </Button>
              ) : null}
              {canCancel ? (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setShowCancelInput(true)}
                  disabled={showCancelInput}
                >
                  Cancel Reversal
                </Button>
              ) : null}
            </div>

            {showRejectInput ? (
              <div className="space-y-2">
                <textarea
                  className="w-full rounded-md border bg-background px-3 py-2 text-xs"
                  rows={2}
                  value={rejectionReason}
                  onChange={(e) => setRejectionReason(e.target.value)}
                  placeholder="Provide a reason for rejection"
                />
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => void reject(undefined)}
                    disabled={isRejecting || !rejectionReason.trim()}
                  >
                    {isRejecting ? "Rejecting..." : "Confirm Reject"}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setShowRejectInput(false);
                      setRejectionReason("");
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : null}

            {showCancelInput ? (
              <div className="space-y-2">
                <textarea
                  className="w-full rounded-md border bg-background px-3 py-2 text-xs"
                  rows={2}
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="Provide a reason for cancellation"
                />
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => void cancel(undefined)}
                    disabled={isCancelling || !cancelReason.trim()}
                  >
                    {isCancelling ? "Cancelling..." : "Confirm Cancel"}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setShowCancelInput(false);
                      setCancelReason("");
                    }}
                  >
                    Dismiss
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        </>
      ) : null}
    </DataTablePanelContainer>
  );
}

export function JournalReversalPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<JournalReversal>) {
  if (mode === "edit") {
    return <ReversalDetailPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }

  return <CreateReversalPanel open={open} onOpenChange={onOpenChange} />;
}
