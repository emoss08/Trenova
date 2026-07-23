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
import { ScrollArea } from "@/components/ui/scroll-area";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { Invoice } from "@/types/invoice";
import type { InvoiceAdjustment, InvoiceAdjustmentPreview } from "@/types/invoice-adjustment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { WalletCardsIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  type AdjustmentFormValues,
  type EditableLine,
  InvoiceAdjustmentLineEditor,
  InvoiceAdjustmentPreviewPanel,
  InvoiceAdjustmentSupportingDocumentsSection,
  InvoiceAdjustmentTypeSelector,
} from "./invoice-adjustment-dialog-sections";

export function InvoiceAdjustmentPanel({ invoice }: { invoice: Invoice }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<InvoiceAdjustment | null>(null);
  const [lines, setLines] = useState<EditableLine[]>([]);
  const [preview, setPreview] = useState<InvoiceAdjustmentPreview | null>(null);
  const form = useForm<AdjustmentFormValues>({
    defaultValues: {
      kind: "CreditOnly",
      rebillStrategy: "CloneExact",
      reason: "",
      referencedDocumentIds: [],
    },
  });
  const {
    clearErrors,
    formState: { errors },
    handleSubmit,
    setValue,
  } = form;
  const kind = useWatch({ control: form.control, name: "kind" });
  const rebillStrategy = useWatch({ control: form.control, name: "rebillStrategy" });
  const supportingDocumentsRequired =
    preview?.supportingDocumentsRequired ?? draft?.supportingDocumentsRequired ?? false;
  const sourceLineAmounts = useMemo(
    () => new Map(invoice.lines.map((line) => [line.id, Math.abs(Number(line.amount ?? 0))])),
    [invoice.lines],
  );
  const previewLinesById = useMemo(
    () => new Map((preview?.lines ?? []).map((line) => [line.originalLineId, line])),
    [preview?.lines],
  );

  useEffect(() => {
    const sourceLines = draft?.lines?.length ? draft.lines : invoice.lines;
    setLines(
      sourceLines.map((line) => {
        const isDraftLine = "originalLineId" in line;
        const quantity = isDraftLine ? line.creditQuantity : line.quantity;
        const creditAmount = isDraftLine ? line.creditAmount : line.amount;
        const rebillAmount = isDraftLine ? line.rebillAmount : line.amount;

        return {
          originalLineId: isDraftLine ? line.originalLineId : line.id,
          description: line.description,
          quantity: String(quantity ?? 1),
          creditAmount: Math.abs(Number(creditAmount)).toFixed(4),
          rebillAmount:
            kind === "CreditAndRebill" && rebillStrategy !== "Rerate"
              ? Math.abs(Number(rebillAmount)).toFixed(4)
              : "0.0000",
        };
      }),
    );
  }, [draft?.lines, invoice.lines, kind, rebillStrategy]);

  useEffect(() => {
    if (!draft) {
      return;
    }

    form.reset({
      kind: draft.kind,
      rebillStrategy: draft.rebillStrategy ?? "CloneExact",
      reason: draft.reason ?? "",
      referencedDocumentIds: draft.referencedDocuments.map((reference) => reference.documentId),
    });
  }, [draft, form]);

  const createDraftMutation = useMutation({
    mutationFn: async () =>
      await apiService.invoiceAdjustmentService.createDraft({ invoiceId: invoice.id }),
    onSuccess: (result) => {
      setDraft(result);
    },
    onError: () => toast.error("Failed to start invoice adjustment"),
  });

  const ensureDraft = async () => {
    if (draft) {
      return draft;
    }

    const createdDraft = await createDraftMutation.mutateAsync();
    setDraft(createdDraft);
    return createdDraft;
  };

  const previewMutation = useApiMutation<
    InvoiceAdjustmentPreview,
    AdjustmentFormValues,
    unknown,
    AdjustmentFormValues
  >({
    setFormError: form.setError,
    resourceName: "invoice adjustment preview",
    mutationFn: async (values: AdjustmentFormValues) => {
      const currentDraft = await ensureDraft();
      const updated = await apiService.invoiceAdjustmentService.updateDraft(
        currentDraft.id,
        buildPayload(values, lines),
      );
      setDraft(updated);
      return apiService.invoiceAdjustmentService.previewDraft(updated.id);
    },
    onSuccess: (result) => {
      clearErrors();
      setPreview(result);
    },
  });

  const submitMutation = useApiMutation<
    InvoiceAdjustment,
    AdjustmentFormValues,
    unknown,
    AdjustmentFormValues
  >({
    setFormError: form.setError,
    resourceName: "invoice adjustment submit",
    mutationFn: async (values: AdjustmentFormValues) => {
      const currentDraft = await ensureDraft();
      const updated = await apiService.invoiceAdjustmentService.updateDraft(
        currentDraft.id,
        buildPayload(values, lines),
      );
      setDraft(updated);
      return apiService.invoiceAdjustmentService.submitDraft(updated.id);
    },
    onSuccess: (result) => {
      clearErrors();
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-adjustment"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      toast.success(
        result.status === "PendingApproval"
          ? "Adjustment submitted for approval"
          : "Adjustment executed",
      );
      setOpen(false);
      setDraft(null);
      setPreview(null);
      form.reset();
    },
  });

  const handlePreview = async (values: AdjustmentFormValues) => {
    clearErrors();
    await previewMutation.mutateAsync(values);
  };

  const handleAdjustmentSubmit = async (values: AdjustmentFormValues) => {
    clearErrors();
    await submitMutation.mutateAsync(values);
  };

  const handleSelectionChange = () => {
    setPreview(null);
  };

  const handleLinesChange = (nextLines: EditableLine[]) => {
    setPreview(null);
    setLines(nextLines);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (nextOpen) {
          setPreview(null);
          createDraftMutation.mutate();
        } else {
          setDraft(null);
          setPreview(null);
        }
      }}
    >
      <Button size="sm" className="cursor-pointer" variant="outline" onClick={() => setOpen(true)}>
        <WalletCardsIcon className="size-3.5" />
        Adjust Invoice
      </Button>
      <DialogContent className="gap-0 p-0 sm:max-w-4xl">
        <DialogHeader className="gap-0 p-4">
          <DialogTitle>Invoice Adjustment</DialogTitle>
          <DialogDescription>
            Preview and submit a policy-controlled credit, reversal, or credit-and-rebill flow.
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={handleSubmit(handleAdjustmentSubmit)}>
          <div className="grid gap-4 px-4 pb-4 lg:grid-cols-[0.75fr_1.25fr]">
            <div className="space-y-4">
              <InvoiceAdjustmentTypeSelector
                kind={kind}
                rebillStrategy={rebillStrategy}
                errors={errors}
                setValue={setValue}
                clearErrors={clearErrors}
                onSelectionChange={handleSelectionChange}
              />
              <InvoiceAdjustmentSupportingDocumentsSection
                control={form.control}
                supportingDocumentsRequired={supportingDocumentsRequired}
                shipmentId={invoice.shipmentId}
                draft={draft}
              />
            </div>

            <ScrollArea className="max-h-[75vh]">
              <div className="space-y-4">
                <InvoiceAdjustmentLineEditor
                  invoice={invoice}
                  lines={lines}
                  setLines={handleLinesChange}
                  kind={kind}
                  rebillStrategy={rebillStrategy}
                  sourceLineAmounts={sourceLineAmounts}
                  previewLinesById={previewLinesById}
                />
                <InvoiceAdjustmentPreviewPanel preview={preview} />
              </div>
            </ScrollArea>
          </div>
          <DialogFooter className="m-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => void handleSubmit(handlePreview)()}
              disabled={previewMutation.isPending}
            >
              Preview
            </Button>
            <Button
              type="submit"
              disabled={submitMutation.isPending || createDraftMutation.isPending || !draft}
            >
              {preview?.requiresApproval ? "Submit for Approval" : "Execute"}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function buildPayload(values: AdjustmentFormValues, lines: EditableLine[]) {
  return {
    kind: values.kind,
    rebillStrategy: values.rebillStrategy,
    reason: values.reason,
    referencedDocumentIds: values.referencedDocumentIds,
    lines: lines
      .filter((line) => Number(line.creditAmount) > 0 || Number(line.rebillAmount) > 0)
      .map((line) => ({
        originalLineId: line.originalLineId,
        creditQuantity: Number(line.quantity),
        creditAmount: Number(line.creditAmount),
        rebillQuantity: Number(line.rebillAmount) > 0 ? Number(line.quantity) : 0,
        rebillAmount: Number(line.rebillAmount),
        description: line.description,
        replacementPayload: {},
      })),
  };
}
