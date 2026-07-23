import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { BatchDetailResponse, CreateBatchRequest } from "@/types/bank-receipt-batch";
import { useQueryClient } from "@tanstack/react-query";
import { UploadIcon } from "lucide-react";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { EMPTY_LINE, ImportBatchForm, type ImportBatchFormValues } from "./import-batch-form";

function toUnixTimestamp(dateStr: string): number {
  const [year, month, day] = dateStr.split("-").map(Number);
  return Math.floor(new Date(year, month - 1, day).getTime() / 1000);
}

function toCents(dollarStr: string): number {
  return Math.round(parseFloat(dollarStr) * 100);
}

type ImportBatchDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function ImportBatchDialog({ open, onOpenChange }: ImportBatchDialogProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const form = useForm<ImportBatchFormValues>({
    defaultValues: {
      source: "",
      reference: "",
      receipts: [{ ...EMPTY_LINE }],
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

  const { mutateAsync } = useApiMutation<
    BatchDetailResponse,
    CreateBatchRequest,
    unknown,
    ImportBatchFormValues
  >({
    mutationFn: async (data: CreateBatchRequest) =>
      apiService.bankReceiptBatchService.create(data),
    setFormError: setError,
    resourceName: "Import Batch",
    onSuccess: (result) => {
      toast.success("Import batch created");
      void queryClient.invalidateQueries({ queryKey: ["bankReceiptBatch"] });
      onOpenChange(false);
      reset();
      if (result?.batch?.id) {
        void navigate(`/accounting/reconciliation/import-batches/${result.batch.id}`);
      }
    },
  });

  const onSubmit = useCallback(
    async (values: ImportBatchFormValues) => {
      const payload: CreateBatchRequest = {
        source: values.source.trim(),
        reference: values.reference.trim(),
        receipts: values.receipts.map((line) => ({
          receiptDate: toUnixTimestamp(line.receiptDate),
          amountMinor: toCents(line.amount),
          referenceNumber: line.referenceNumber.trim(),
          memo: line.memo.trim() || undefined,
        })),
      };
      await mutateAsync(payload);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Import Bank Receipts</DialogTitle>
          <DialogDescription>
            Create a batch of bank receipts to import for reconciliation.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <ImportBatchForm />
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting} isLoading={isSubmitting}>
                <UploadIcon className="mr-1.5 size-3.5" />
                Import Batch
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
