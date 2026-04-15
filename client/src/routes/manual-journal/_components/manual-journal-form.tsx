import { AmountDisplay } from "@/components/accounting/amount-display";
import { JournalLineItemsEditor } from "@/components/accounting/journal-line-items-editor";
import { JournalLineItemsTable } from "@/components/accounting/journal-line-items-table";
import {
  FiscalPeriodAutocompleteField,
  FiscalYearAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import type { JournalEntryLine } from "@/types/journal-entry";
import type { ManualJournalLine } from "@/types/manual-journal";
import { AlertTriangleIcon } from "lucide-react";
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
  const isBalanced = totalDebit === totalCredit && totalDebit > 0;

  return (
    <div className="mt-3 overflow-hidden rounded-lg border bg-muted/50 p-2">
      <div className="mb-3">
        <span className="text-xs font-medium">Balance Summary</span>
        <p className="mt-0.5 text-2xs text-muted-foreground">
          Journal entries must balance — total debits must equal total credits.
        </p>
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Total Debit</span>
          <AmountDisplay value={totalDebit} className="text-sm tabular-nums" />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Total Credit</span>
          <AmountDisplay value={totalCredit} className="text-sm tabular-nums" />
        </div>
        <Separator />
        <div className="flex items-center justify-between">
          <span
            className={cn(
              "text-sm font-medium",
              isBalanced ? "text-foreground" : "text-red-600 dark:text-red-400",
            )}
          >
            Difference
          </span>
          <span className="flex items-center gap-1.5">
            {!isBalanced && totalDebit + totalCredit > 0 ? (
              <AlertTriangleIcon className="size-3.5 text-red-600 dark:text-red-400" />
            ) : null}
            <AmountDisplay
              value={difference}
              variant={isBalanced ? "neutral" : "negative"}
              className="text-base font-semibold tabular-nums"
            />
          </span>
        </div>
      </div>
    </div>
  );
}

type ManualJournalFormProps = {
  isDraft?: boolean;
};

export function ManualJournalForm({ isDraft = true }: ManualJournalFormProps) {
  const { control } = useFormContext();
  const lines: ManualJournalLine[] = useWatch({ control, name: "lines" }) ?? [];
  const fiscalYearId: string = useWatch({ control, name: "requestedFiscalYearId" }) ?? "";

  const totalDebit = lines.reduce((sum, l) => sum + (Number(l?.debitAmount) || 0), 0);
  const totalCredit = lines.reduce((sum, l) => sum + (Number(l?.creditAmount) || 0), 0);

  const periodSearchParams = useMemo(() => {
    if (!fiscalYearId) return undefined;
    return {
      fieldFilters: JSON.stringify([
        { field: "fiscalYearId", operator: "eq", value: fiscalYearId },
      ]),
    };
  }, [fiscalYearId]);

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title="Journal Details"
        description="Basic information about the manual journal entry"
      >
        <FormGroup cols={2}>
          <FormControl>
            <TextareaField
              control={control}
              name="description"
              label="Description"
              rules={{ required: true }}
              disabled={!isDraft}
              description="A brief summary of the journal entry purpose."
              placeholder="Describe the purpose of this journal entry"
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              disabled={!isDraft}
              description="Why this journal entry is being created."
              placeholder="Provide context or justification"
            />
          </FormControl>
        </FormGroup>
        <FormGroup cols={3}>
          <FormControl>
            <FiscalYearAutocompleteField
              control={control}
              name="requestedFiscalYearId"
              label="Fiscal Year"
              placeholder="Select Fiscal Year"
              description="The fiscal year this entry applies to."
            />
          </FormControl>
          <FormControl>
            <FiscalPeriodAutocompleteField
              control={control}
              name="requestedFiscalPeriodId"
              label="Fiscal Period"
              placeholder="Select Period"
              description="The fiscal period within the selected year."
              extraSearchParams={periodSearchParams}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="currencyCode"
              label="Currency"
              disabled={!isDraft}
              description="ISO currency code for all amounts."
              placeholder="USD"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Line Items"
        description="Debit and credit entries that make up this journal"
        titleCount={lines.length}
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
