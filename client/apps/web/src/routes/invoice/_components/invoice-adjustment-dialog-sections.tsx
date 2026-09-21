import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentMultiSelectAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentUploadSection } from "@/components/document-upload-section";
import { NumberInput } from "@/components/fields/number-input";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Separator } from "@trenova/shared/components/ui/separator";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import type {
  InvoiceAdjustment,
  InvoiceAdjustmentKind,
  InvoiceAdjustmentPreview,
  InvoiceAdjustmentPreviewLine,
  RebillStrategy,
} from "@/types/invoice-adjustment";
import {
  AlertTriangleIcon,
  BanIcon,
  CalendarIcon,
  CheckCircle2Icon,
  CircleDollarSignIcon,
  InfoIcon,
  ReceiptIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  ShieldAlertIcon,
} from "lucide-react";
import type { Control, FieldErrors, UseFormClearErrors, UseFormSetValue } from "react-hook-form";
import { formatUnixDate } from "@trenova/shared/lib/date";

export type EditableLine = {
  originalLineId: string;
  description: string;
  quantity: string;
  creditAmount: string;
  rebillAmount: string;
};

export type AdjustmentFormValues = {
  kind: InvoiceAdjustmentKind;
  rebillStrategy: RebillStrategy;
  reason: string;
  referencedDocumentIds: string[];
};

const adjustmentTypes: {
  value: InvoiceAdjustmentKind;
  label: string;
  description: string;
  icon: React.ReactNode;
}[] = [
  {
    value: "CreditOnly",
    label: "Credit only",
    description: "Issue a credit memo without rebilling",
    icon: <CircleDollarSignIcon className="size-4" />,
  },
  {
    value: "CreditAndRebill",
    label: "Credit & rebill",
    description: "Credit the original and issue a corrected invoice",
    icon: <RefreshCwIcon className="size-4" />,
  },
  {
    value: "FullReversal",
    label: "Full reversal",
    description: "Reverse all charges on this invoice",
    icon: <RotateCcwIcon className="size-4" />,
  },
];

const rebillStrategies: {
  value: RebillStrategy;
  label: string;
  description: string;
}[] = [
  {
    value: "CloneExact",
    label: "Clone exact",
    description: "Copy original line amounts",
  },
  {
    value: "Rerate",
    label: "Rerate",
    description: "Recalculate from current rates",
  },
  {
    value: "Manual",
    label: "Manual",
    description: "Set rebill amounts manually",
  },
];

export function InvoiceAdjustmentTypeSelector({
  kind,
  rebillStrategy,
  errors,
  setValue,
  clearErrors,
  onSelectionChange,
}: {
  kind: InvoiceAdjustmentKind;
  rebillStrategy: RebillStrategy;
  errors: FieldErrors<AdjustmentFormValues>;
  setValue: UseFormSetValue<AdjustmentFormValues>;
  clearErrors: UseFormClearErrors<AdjustmentFormValues>;
  onSelectionChange?: () => void;
}) {
  const t = useT();

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {adjustmentTypes.map((type) => {
          const isSelected = kind === type.value;
          return (
            <button
              key={type.value}
              type="button"
              className={cn(
                "flex w-full items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-all duration-150",
                isSelected
                  ? "border-brand bg-surface-selected"
                  : "border-border bg-background hover:border-muted-foreground/30 hover:bg-muted/40",
              )}
              onClick={() => {
                setValue("kind", type.value, { shouldDirty: true });
                clearErrors("kind");
                onSelectionChange?.();
              }}
            >
              <span
                className={cn(
                  "mt-0.5 shrink-0 transition-colors",
                  isSelected ? "text-brand" : "text-muted-foreground",
                )}
              >
                {type.icon}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-foreground text-sm font-medium">{t(type.label)}</p>
                <p className="text-muted-foreground text-xs">{t(type.description)}</p>
              </div>
              <div
                className={cn(
                  "mt-1 flex size-4 shrink-0 items-center justify-center rounded-full border-2 transition-all",
                  isSelected ? "border-brand bg-brand" : "border-muted-foreground/30",
                )}
              >
                {isSelected ? (
                  <div className="size-1.5 rounded-full bg-foreground-on-solid" />
                ) : null}
              </div>
            </button>
          );
        })}
      </div>

      {kind === "CreditAndRebill" ? (
        <div className="space-y-1.5">
          <p className="text-muted-foreground text-xs font-medium">{t("Rebill strategy")}</p>
          <div className="border-border bg-muted/50 flex gap-1 rounded-lg border p-1">
            {rebillStrategies.map((strategy) => {
              const isSelected = rebillStrategy === strategy.value;
              return (
                <button
                  key={strategy.value}
                  type="button"
                  title={t(strategy.description)}
                  className={cn(
                    "flex-1 rounded-md border border-transparent px-2.5 py-1.5 text-xs font-medium transition-all duration-150",
                    isSelected
                      ? "border-border bg-background text-foreground"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                  onClick={() => {
                    setValue("rebillStrategy", strategy.value, { shouldDirty: true });
                    clearErrors("rebillStrategy");
                    onSelectionChange?.();
                  }}
                >
                  {t(strategy.label)}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      {errors.kind?.message ? (
        <p className="text-destructive text-xs">{errors.kind.message}</p>
      ) : null}
      {errors.rebillStrategy?.message ? (
        <p className="text-destructive text-xs">{errors.rebillStrategy.message}</p>
      ) : null}
    </div>
  );
}

export function InvoiceAdjustmentSupportingDocumentsSection({
  control,
  shipmentId,
  draft,
}: {
  control: Control<AdjustmentFormValues>;
  shipmentId: Invoice["shipmentId"];
  draft: InvoiceAdjustment | null;
}) {
  const t = useT();

  return (
    <div className="space-y-4">
      <Separator />
      <TextareaField
        control={control}
        name="reason"
        label={t("Reason")}
        placeholder={t("Describe the commercial correction or finance rationale...")}
        minRows={3}
      />

      <DocumentMultiSelectAutocompleteField
        control={control}
        name="referencedDocumentIds"
        label={t("Supporting documents (optional)")}
        placeholder={t("Search shipment documents...")}
        description={t("Attach supporting evidence for audit trail.")}
        extraSearchParams={{
          resourceId: shipmentId ?? "",
          resourceType: "shipment",
        }}
      />

      {draft ? (
        <DocumentUploadSection
          resourceId={draft.id}
          resourceType="invoice_adjustment"
          disabled={draft.status !== "Draft"}
        />
      ) : null}
    </div>
  );
}

export function InvoiceAdjustmentLineEditor({
  invoice,
  lines,
  setLines,
  kind,
  rebillStrategy,
  sourceLineAmounts,
  previewLinesById,
}: {
  invoice: Invoice;
  lines: EditableLine[];
  setLines: (updater: EditableLine[]) => void;
  kind: InvoiceAdjustmentKind;
  rebillStrategy: RebillStrategy;
  sourceLineAmounts: Map<string, number>;
  previewLinesById: Map<string, InvoiceAdjustmentPreviewLine>;
}) {
  const t = useT();

  return (
    <div className="border-border overflow-hidden rounded-lg border">
      <div className="border-border bg-muted/40 grid grid-cols-[1fr_120px_120px] items-center border-b px-4 py-2">
        <span className="text-muted-foreground text-xs font-medium">{t("Description")}</span>
        <span className="text-muted-foreground text-right text-xs font-medium">{t("Credit")}</span>
        <span className="text-muted-foreground text-right text-xs font-medium">{t("Rebill")}</span>
      </div>
      <div className="divide-border divide-y">
        {lines.map((line, index) => (
          <InvoiceAdjustmentLineEditorRow
            key={line.originalLineId}
            index={index}
            invoice={invoice}
            line={line}
            lines={lines}
            setLines={setLines}
            kind={kind}
            rebillStrategy={rebillStrategy}
            sourceLineAmounts={sourceLineAmounts}
            previewLine={previewLinesById.get(line.originalLineId)}
          />
        ))}
      </div>
    </div>
  );
}

function InvoiceAdjustmentLineEditorRow({
  index,
  invoice,
  line,
  lines,
  setLines,
  kind,
  rebillStrategy,
  sourceLineAmounts,
  previewLine,
}: {
  index: number;
  invoice: Invoice;
  line: EditableLine;
  lines: EditableLine[];
  setLines: (updater: EditableLine[]) => void;
  kind: InvoiceAdjustmentKind;
  rebillStrategy: RebillStrategy;
  sourceLineAmounts: Map<string, number>;
  previewLine?: InvoiceAdjustmentPreviewLine;
}) {
  const t = useT();

  const originalAmount =
    sourceLineAmounts.get(line.originalLineId) ??
    Math.abs(Number(invoice.lines[index]?.amount ?? 0));
  const alreadyCreditedAmount = Number(previewLine?.alreadyCreditedAmount ?? 0);
  const remainingEligibleAmount = Number(previewLine?.remainingEligibleAmount ?? originalAmount);
  const requestedCreditAmount = Number(
    previewLine?.requestedCreditAmount ?? line.creditAmount ?? 0,
  );
  const overageAmount = Number(previewLine?.eligibilityOverageAmount ?? 0);
  const hasError = previewLine?.hasEligibilityError;

  return (
    <div
      className={cn(
        "grid grid-cols-[1fr_120px_120px] items-start gap-3 px-4 py-3 transition-colors",
        hasError ? "bg-danger-subtle" : "bg-background",
      )}
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="bg-muted text-2xs text-muted-foreground inline-flex size-5 shrink-0 items-center justify-center rounded-md font-medium">
            {previewLine?.lineNumber ?? index + 1}
          </span>
          <p className="truncate text-sm font-medium">{t(line.description)}</p>
        </div>
        <div className="text-2xs text-muted-foreground mt-1.5 ml-7 flex flex-wrap items-center gap-x-3 gap-y-0.5">
          <span>{t("{0} original", formatCurrency(originalAmount))}</span>
          <span className="text-muted-foreground/40">/</span>
          <span>{t("{0} credited", formatCurrency(alreadyCreditedAmount))}</span>
          <span className="text-muted-foreground/40">/</span>
          <span>{t("{0} eligible", formatCurrency(Math.max(remainingEligibleAmount, 0)))}</span>
        </div>
        {hasError ? (
          <div className="mt-1.5 ml-7 flex items-start gap-1.5">
            <BanIcon className="text-destructive mt-0.5 size-3 shrink-0" />
            <p className="text-2xs text-destructive">
              {previewLine?.eligibilityMessage ||
                t("Exceeds eligibility by {0}", formatCurrency(overageAmount))}
            </p>
          </div>
        ) : requestedCreditAmount > 0 ? (
          <p className="text-2xs text-muted-foreground mt-1 ml-7">
            {t("Requesting {0} credit", formatCurrency(requestedCreditAmount))}
          </p>
        ) : null}
      </div>
      <div className="pt-0.5">
        <NumberInput
          value={line.creditAmount}
          onValueChange={(value) => {
            const next = [...lines];
            next[index] = { ...line, creditAmount: value };
            setLines(next);
          }}
          decimalScale={4}
          fixedDecimalScale
          allowNegative={false}
          placeholder="0.0000"
          aria-label={`Credit amount for ${line.description}`}
        />
      </div>
      <div className="pt-0.5">
        <NumberInput
          value={line.rebillAmount}
          disabled={kind !== "CreditAndRebill" || rebillStrategy === "Rerate"}
          onValueChange={(value) => {
            const next = [...lines];
            next[index] = { ...line, rebillAmount: value };
            setLines(next);
          }}
          decimalScale={4}
          fixedDecimalScale
          allowNegative={false}
          placeholder="0.0000"
          aria-label={`Rebill amount for ${line.description}`}
        />
      </div>
    </div>
  );
}

export function InvoiceAdjustmentPreviewPanel({
  preview,
}: {
  preview: InvoiceAdjustmentPreview | null;
}) {
  const t = useT();

  if (!preview) {
    return (
      <div className="border-border flex flex-col items-center justify-center rounded-lg border border-dashed py-8">
        <ReceiptIcon className="text-muted-foreground/30 mb-2 size-5" />
        <p className="text-muted-foreground/60 text-xs">
          {t("Click Preview to see the adjustment summary")}
        </p>
      </div>
    );
  }

  const eligibilityIssues = preview.lines.filter((line) => line.hasEligibilityError);
  const previewErrors = Object.entries(preview.errors).filter(
    ([field]) => field !== "lines" || eligibilityIssues.length === 0,
  );
  const hasIssues = eligibilityIssues.length > 0 || previewErrors.length > 0;

  return (
    <div className="space-y-3">
      <div className="border-border overflow-hidden rounded-lg border">
        <div className="border-border bg-muted/40 border-b px-4 py-2">
          <p className="text-muted-foreground text-xs font-medium">{t("Adjustment summary")}</p>
        </div>
        <DescriptionList layout="split" className="px-4">
          <DescriptionItem label={t("Credit total")} numeric>
            {formatCurrency(Number(preview.creditTotalAmount))}
          </DescriptionItem>
          <DescriptionItem label={t("Rebill total")} numeric>
            {formatCurrency(Number(preview.rebillTotalAmount))}
          </DescriptionItem>
          <DescriptionItem label={t("Net delta")} numeric valueClassName="font-semibold">
            {formatCurrency(Number(preview.netDeltaAmount))}
          </DescriptionItem>
          <DescriptionItem
            label={
              <span className="inline-flex items-center gap-1.5">
                <CalendarIcon className="size-3" />
                {t("Accounting date")}
              </span>
            }
            numeric
          >
            {formatUnixDate(preview.accountingDate)}
          </DescriptionItem>
        </DescriptionList>
      </div>

      {preview.requiresApproval ||
      preview.requiresReconciliationException ||
      preview.requiresReplacementInvoiceReview ||
      preview.wouldCreateUnappliedCredit ? (
        <Alert variant="warning">
          <ShieldAlertIcon />
          <AlertTitle>{t("Policy implications")}</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4">
              {preview.requiresApproval ? (
                <li>{t("Approval required before financial mutation")}</li>
              ) : null}
              {preview.requiresReconciliationException ? (
                <li>{t("Creates a reconciliation exception for finance follow-up")}</li>
              ) : null}
              {preview.requiresReplacementInvoiceReview ? (
                <li>{t("Replacement invoice requires billing review")}</li>
              ) : null}
              {preview.wouldCreateUnappliedCredit ? (
                <li>{t("Creates unapplied customer credit based on settlement state")}</li>
              ) : null}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}

      {preview.warnings.length > 0 ? (
        <div className="border-border bg-muted/30 rounded-lg border px-4 py-3">
          <div className="flex items-center gap-2">
            <InfoIcon className="text-muted-foreground size-3.5" />
            <p className="text-muted-foreground text-xs font-medium">{t("Warnings")}</p>
          </div>
          <div className="mt-2 space-y-1">
            {preview.warnings.map((warning) => (
              <p key={warning} className="text-muted-foreground text-xs">
                {warning}
              </p>
            ))}
          </div>
        </div>
      ) : null}

      {hasIssues ? (
        <Alert variant="destructive">
          <AlertTriangleIcon />
          <AlertTitle>{t("Issues found")}</AlertTitle>
          <AlertDescription>
            {eligibilityIssues.map((line) => (
              <p key={line.originalLineId}>{line.eligibilityMessage}</p>
            ))}
            {previewErrors.map(([field, messages]) => (
              <div key={field}>
                <p className="font-medium">{field}</p>
                {messages.map((message) => (
                  <p key={message}>{message}</p>
                ))}
              </div>
            ))}
          </AlertDescription>
        </Alert>
      ) : (
        <Alert variant="success" size="sm">
          <CheckCircle2Icon />
          <AlertTitle>{t("Preview passed validation")}</AlertTitle>
        </Alert>
      )}
    </div>
  );
}
