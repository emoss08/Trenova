import { DocumentMultiSelectAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentUploadSection } from "@/components/document-upload-section";
import { NumberInput } from "@/components/fields/number-input";
import { TextareaField } from "@/components/fields/textarea-field";
import { Separator } from "@/components/ui/separator";
import { cn, formatCurrency } from "@/lib/utils";
import type { Invoice } from "@/types/invoice";
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
    label: "Credit Only",
    description: "Issue a credit memo without rebilling",
    icon: <CircleDollarSignIcon className="size-4" />,
  },
  {
    value: "CreditAndRebill",
    label: "Credit & Rebill",
    description: "Credit the original and issue a corrected invoice",
    icon: <RefreshCwIcon className="size-4" />,
  },
  {
    value: "FullReversal",
    label: "Full Reversal",
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
    label: "Clone Exact",
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
                  ? "border-brand bg-brand/5 ring-1 ring-brand/20"
                  : "border-border bg-background hover:border-muted-foreground/30 hover:bg-muted/40",
              )}
              onClick={() => {
                setValue("kind", type.value, { shouldDirty: true });
                clearErrors("kind");
                onSelectionChange?.();
              }}
            >
              <div
                className={cn(
                  "mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md transition-colors",
                  isSelected ? "bg-brand/10 text-brand" : "bg-muted text-muted-foreground",
                )}
              >
                {type.icon}
              </div>
              <div className="min-w-0 flex-1">
                <p
                  className={cn(
                    "text-sm font-medium",
                    isSelected ? "text-foreground" : "text-foreground",
                  )}
                >
                  {type.label}
                </p>
                <p className="text-xs text-muted-foreground">{type.description}</p>
              </div>
              <div
                className={cn(
                  "mt-1 flex size-4 shrink-0 items-center justify-center rounded-full border-2 transition-all",
                  isSelected ? "border-brand bg-brand" : "border-muted-foreground/30",
                )}
              >
                {isSelected ? <div className="size-1.5 rounded-full bg-white" /> : null}
              </div>
            </button>
          );
        })}
      </div>

      {kind === "CreditAndRebill" ? (
        <div className="space-y-1.5">
          <p className="text-xs font-medium text-muted-foreground">Rebill Strategy</p>
          <div className="flex gap-1 rounded-lg border border-border bg-muted/50 p-1">
            {rebillStrategies.map((strategy) => {
              const isSelected = rebillStrategy === strategy.value;
              return (
                <button
                  key={strategy.value}
                  type="button"
                  title={strategy.description}
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
                  {strategy.label}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      {errors.kind?.message ? (
        <p className="text-xs text-destructive">{errors.kind.message}</p>
      ) : null}
      {errors.rebillStrategy?.message ? (
        <p className="text-xs text-destructive">{errors.rebillStrategy.message}</p>
      ) : null}
    </div>
  );
}

export function InvoiceAdjustmentSupportingDocumentsSection({
  control,
  supportingDocumentsRequired,
  shipmentId,
  draft,
}: {
  control: Control<AdjustmentFormValues>;
  supportingDocumentsRequired: boolean;
  shipmentId: Invoice["shipmentId"];
  draft: InvoiceAdjustment | null;
}) {
  return (
    <div className="space-y-4">
      <Separator />
      <TextareaField
        control={control}
        name="reason"
        label="Reason"
        placeholder="Describe the commercial correction or finance rationale..."
        minRows={3}
      />

      <DocumentMultiSelectAutocompleteField
        control={control}
        name="referencedDocumentIds"
        label={
          supportingDocumentsRequired
            ? "Supporting Documents (Required)"
            : "Supporting Documents (Optional)"
        }
        placeholder="Search shipment documents..."
        description={
          supportingDocumentsRequired
            ? "Required by policy for this adjustment type."
            : "Attach supporting evidence for audit trail."
        }
        rules={{ required: supportingDocumentsRequired }}
        extraSearchParams={{
          resourceId: shipmentId,
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
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <div className="grid grid-cols-[1fr_120px_120px] items-center border-b border-border bg-muted/40 px-4 py-2">
        <span className="text-xs font-medium text-muted-foreground">Description</span>
        <span className="text-right text-xs font-medium text-muted-foreground">Credit</span>
        <span className="text-right text-xs font-medium text-muted-foreground">Rebill</span>
      </div>
      <div className="divide-y divide-border">
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
        hasError ? "bg-destructive/5" : "bg-background",
      )}
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="inline-flex size-5 shrink-0 items-center justify-center rounded bg-muted text-2xs font-medium text-muted-foreground">
            {previewLine?.lineNumber ?? index + 1}
          </span>
          <p className="truncate text-sm font-medium">{line.description}</p>
        </div>
        <div className="mt-1.5 ml-7 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-2xs text-muted-foreground">
          <span>{formatCurrency(originalAmount)} original</span>
          <span className="text-muted-foreground/40">/</span>
          <span>{formatCurrency(alreadyCreditedAmount)} credited</span>
          <span className="text-muted-foreground/40">/</span>
          <span>{formatCurrency(Math.max(remainingEligibleAmount, 0))} eligible</span>
        </div>
        {hasError ? (
          <div className="mt-1.5 ml-7 flex items-start gap-1.5">
            <BanIcon className="mt-0.5 size-3 shrink-0 text-destructive" />
            <p className="text-2xs text-destructive">
              {previewLine?.eligibilityMessage ||
                `Exceeds eligibility by ${formatCurrency(overageAmount)}`}
            </p>
          </div>
        ) : requestedCreditAmount > 0 ? (
          <p className="mt-1 ml-7 text-2xs text-muted-foreground">
            Requesting {formatCurrency(requestedCreditAmount)} credit
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
  if (!preview) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-8">
        <ReceiptIcon className="mb-2 size-5 text-muted-foreground/30" />
        <p className="text-xs text-muted-foreground/60">
          Click Preview to see the adjustment summary
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
      <div className="overflow-hidden rounded-lg border border-border">
        <div className="border-b border-border bg-muted/40 px-4 py-2">
          <p className="text-xs font-medium text-muted-foreground">Adjustment Summary</p>
        </div>
        <div className="divide-y divide-border">
          <PreviewRow
            label="Credit Total"
            value={formatCurrency(Number(preview.creditTotalAmount))}
          />
          <PreviewRow
            label="Rebill Total"
            value={formatCurrency(Number(preview.rebillTotalAmount))}
          />
          <PreviewRow
            label="Net Delta"
            value={formatCurrency(Number(preview.netDeltaAmount))}
            highlight
          />
          <PreviewRow
            label="Accounting Date"
            value={new Date(preview.accountingDate * 1000).toLocaleDateString()}
            icon={<CalendarIcon className="size-3" />}
          />
        </div>
      </div>

      {preview.requiresApproval ||
      preview.requiresReconciliationException ||
      preview.requiresReplacementInvoiceReview ||
      preview.wouldCreateUnappliedCredit ? (
        <div className="rounded-lg border border-yellow-600/20 bg-yellow-600/5 px-4 py-3">
          <div className="flex items-center gap-2">
            <ShieldAlertIcon className="size-3.5 text-yellow-600 dark:text-yellow-400" />
            <p className="text-xs font-medium text-yellow-700 dark:text-yellow-400">
              Policy Implications
            </p>
          </div>
          <div className="mt-2 space-y-1.5">
            {preview.requiresApproval ? (
              <PolicyItem text="Approval required before financial mutation" />
            ) : null}
            {preview.requiresReconciliationException ? (
              <PolicyItem text="Creates a reconciliation exception for finance follow-up" />
            ) : null}
            {preview.requiresReplacementInvoiceReview ? (
              <PolicyItem text="Replacement invoice requires billing review" />
            ) : null}
            {preview.wouldCreateUnappliedCredit ? (
              <PolicyItem text="Creates unapplied customer credit based on settlement state" />
            ) : null}
          </div>
        </div>
      ) : null}

      {preview.warnings.length > 0 ? (
        <div className="rounded-lg border border-border bg-muted/30 px-4 py-3">
          <div className="flex items-center gap-2">
            <InfoIcon className="size-3.5 text-muted-foreground" />
            <p className="text-xs font-medium text-muted-foreground">Warnings</p>
          </div>
          <div className="mt-2 space-y-1">
            {preview.warnings.map((warning) => (
              <p key={warning} className="text-xs text-muted-foreground">
                {warning}
              </p>
            ))}
          </div>
        </div>
      ) : null}

      {hasIssues ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3">
          <div className="flex items-center gap-2">
            <AlertTriangleIcon className="size-3.5 text-destructive" />
            <p className="text-xs font-medium text-destructive">Issues Found</p>
          </div>
          <div className="mt-2 space-y-2">
            {eligibilityIssues.map((line) => (
              <p key={line.originalLineId} className="text-xs text-destructive">
                {line.eligibilityMessage}
              </p>
            ))}
            {previewErrors.map(([field, messages]) => (
              <div key={field}>
                <p className="text-2xs font-medium tracking-wide text-destructive/70 uppercase">
                  {field}
                </p>
                {messages.map((message) => (
                  <p key={message} className="text-xs text-destructive">
                    {message}
                  </p>
                ))}
              </div>
            ))}
          </div>
        </div>
      ) : !hasIssues ? (
        <div className="flex items-center gap-2 rounded-lg border border-green-600/20 bg-green-600/5 px-4 py-2.5">
          <CheckCircle2Icon className="size-3.5 text-green-600 dark:text-green-400" />
          <p className="text-xs font-medium text-green-700 dark:text-green-400">
            Preview passed validation
          </p>
        </div>
      ) : null}
    </div>
  );
}

function PreviewRow({
  label,
  value,
  highlight,
  icon,
}: {
  label: string;
  value: string;
  highlight?: boolean;
  icon?: React.ReactNode;
}) {
  return (
    <div
      className={cn("flex items-center justify-between px-4 py-2.5", highlight && "bg-muted/30")}
    >
      <span
        className={cn(
          "flex items-center gap-1.5 text-xs",
          highlight ? "font-medium text-foreground" : "text-muted-foreground",
        )}
      >
        {icon}
        {label}
      </span>
      <span
        className={cn(
          "text-sm tabular-nums",
          highlight ? "font-semibold text-foreground" : "font-medium text-foreground",
        )}
      >
        {value}
      </span>
    </div>
  );
}

function PolicyItem({ text }: { text: string }) {
  return (
    <div className="flex items-start gap-2">
      <div className="mt-1 size-1 shrink-0 rounded-full bg-yellow-600 dark:bg-yellow-400" />
      <p className="text-xs text-yellow-700 dark:text-yellow-300">{text}</p>
    </div>
  );
}
