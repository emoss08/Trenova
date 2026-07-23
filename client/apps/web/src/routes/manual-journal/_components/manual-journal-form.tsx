import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { JournalLineItemsEditor } from "@/components/accounting/journal-line-items-editor";
import { JournalLineItemsTable } from "@/components/accounting/journal-line-items-table";
import {
  FiscalPeriodAutocompleteField,
  FiscalYearAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { cn } from "@trenova/shared/lib/utils";
import type { JournalEntryLine } from "@/types/journal-entry";
import type { ManualJournalLine } from "@/types/manual-journal";
import { CheckCircle2Icon, ScaleIcon } from "lucide-react";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";

function mapToJournalEntryLines(lines: ManualJournalLine[]): JournalEntryLine[] {
  return lines.map((l) => ({
    id: l.id ?? "",
    journalEntryId: "",
    glAccountId: l.glAccountId,
    lineNumber: l.lineNumber,
    description: l.description,
    debitAmount: l.debitAmount,
    creditAmount: l.creditAmount,
    netAmount: l.debitAmount - l.creditAmount,
    customerId: l.customerId ?? null,
    locationId: l.locationId ?? null,
    glAccount: l.glAccount ?? null,
  }));
}

function BalanceSummary({
  totalDebit,
  totalCredit,
}: {
  totalDebit: number;
  totalCredit: number;
}) {
  const difference = totalDebit - totalCredit;
  const hasAmounts = totalDebit + totalCredit > 0;
  const isBalanced = difference === 0 && totalDebit > 0;

  return (
    <div className="grid grid-cols-3 divide-x divide-border overflow-hidden rounded-lg border bg-muted/30">
      <div className="flex flex-col gap-1 px-4 py-3">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Total Debits
        </span>
        <AmountDisplay value={totalDebit} className="text-sm font-semibold" />
      </div>
      <div className="flex flex-col gap-1 px-4 py-3">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Total Credits
        </span>
        <AmountDisplay value={totalCredit} className="text-sm font-semibold" />
      </div>
      <div
        className={cn(
          "flex flex-col gap-1 px-4 py-3",
          hasAmounts && (isBalanced ? "bg-green-600/10" : "bg-red-600/10"),
        )}
      >
        <span
          className={cn(
            "text-2xs font-medium tracking-wide uppercase",
            !hasAmounts && "text-muted-foreground",
            hasAmounts &&
              (isBalanced
                ? "text-green-700 dark:text-green-400"
                : "text-red-700 dark:text-red-400"),
          )}
        >
          {isBalanced ? "Balanced" : "Difference"}
        </span>
        <span className="flex items-center gap-1.5">
          {isBalanced ? (
            <CheckCircle2Icon className="size-4 text-green-600 dark:text-green-400" />
          ) : hasAmounts ? (
            <ScaleIcon className="size-4 text-red-600 dark:text-red-400" />
          ) : null}
          <AmountDisplay
            value={difference}
            variant={!hasAmounts || isBalanced ? "neutral" : "negative"}
            className="text-sm font-semibold"
          />
        </span>
      </div>
    </div>
  );
}

type ManualJournalFormProps = {
  isDraft?: boolean;
};

export function ManualJournalForm({ isDraft = true }: ManualJournalFormProps) {
  const { control, setValue } = useFormContext();
  const lines: ManualJournalLine[] = useWatch({ control, name: "lines" }) ?? [];
  const fiscalYearId: string = useWatch({ control, name: "requestedFiscalYearId" }) ?? "";

  const totalDebit = lines.reduce((sum, l) => sum + (Number(l?.debitAmount) || 0), 0);
  const totalCredit = lines.reduce((sum, l) => sum + (Number(l?.creditAmount) || 0), 0);

  const periodSearchParams = useMemo(() => {
    if (!fiscalYearId) return undefined;
    return { fiscalYearId };
  }, [fiscalYearId]);

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title="Journal Details"
        description="Describe the entry and when it should hit the general ledger."
      >
        <FormGroup cols={2}>
          <FormControl>
            <TextareaField
              control={control}
              name="description"
              label="Description"
              rules={{ required: "Description is required" }}
              disabled={!isDraft}
              description="A short summary that will appear on the posted journal entry."
              placeholder="e.g. Accrue December fuel invoices"
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              disabled={!isDraft}
              description="Business justification for the adjustment, shown to approvers."
              placeholder="e.g. Vendor invoices received after period cutoff"
            />
          </FormControl>
        </FormGroup>
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="accountingDate"
              label="Accounting Date"
              rules={{ required: "Accounting date is required" }}
              disabled={!isDraft}
              description="The GL date for this entry. It must fall within an open fiscal period."
              placeholder="Select date"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="currencyCode"
              label="Currency"
              disabled={!isDraft}
              maxLength={3}
              description="ISO 4217 code for all line amounts (e.g. USD)."
              placeholder="USD"
            />
          </FormControl>
        </FormGroup>
        <FormGroup cols={2}>
          <FormControl>
            <FiscalYearAutocompleteField
              control={control}
              name="requestedFiscalYearId"
              label="Fiscal Year"
              disabled={!isDraft}
              placeholder="Select fiscal year"
              description="Requested fiscal year. The posting period is resolved from the accounting date."
              onOptionChange={() => setValue("requestedFiscalPeriodId", "")}
            />
          </FormControl>
          <FormControl>
            <FiscalPeriodAutocompleteField
              control={control}
              name="requestedFiscalPeriodId"
              label="Fiscal Period"
              disabled={!isDraft || !fiscalYearId}
              placeholder={fiscalYearId ? "Select period" : "Select a fiscal year first"}
              description="Requested period within the selected fiscal year."
              extraSearchParams={periodSearchParams}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Line Items"
        titleCount={lines.length}
        description="Each line debits or credits a GL account. Total debits must equal total credits before the journal can be submitted."
        className="border-t border-border pt-4"
      >
        {isDraft ? (
          <JournalLineItemsEditor />
        ) : (
          <JournalLineItemsTable
            lines={mapToJournalEntryLines(lines)}
            totalDebit={totalDebit}
            totalCredit={totalCredit}
          />
        )}
        <BalanceSummary totalDebit={totalDebit} totalCredit={totalCredit} />
      </FormSection>
    </div>
  );
}
