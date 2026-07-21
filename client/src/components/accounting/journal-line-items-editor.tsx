import { AmountDisplay } from "@/components/accounting/amount-display";
import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ManualJournalLine } from "@/types/manual-journal";
import { CheckCircle2Icon, PlusIcon, Trash2Icon } from "lucide-react";
import { useCallback } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

type LineFormValues = { lines: ManualJournalLine[] };

type JournalLineItemsEditorProps = {
  className?: string;
};

export function JournalLineItemsEditor({ className }: JournalLineItemsEditorProps) {
  const { control, getValues, setValue } = useFormContext<LineFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "lines" });

  const watchedLines = useWatch({ control, name: "lines" });

  const totalDebit = (watchedLines ?? []).reduce(
    (sum, line) => sum + (Number(line?.debitAmount) || 0),
    0,
  );
  const totalCredit = (watchedLines ?? []).reduce(
    (sum, line) => sum + (Number(line?.creditAmount) || 0),
    0,
  );
  const difference = totalDebit - totalCredit;
  const hasAmounts = totalDebit + totalCredit > 0;
  const isBalanced = difference === 0 && totalDebit > 0;

  const clearOpposite = useCallback(
    (index: number, opposite: "debitAmount" | "creditAmount") => (cents: number) => {
      if (cents > 0 && (getValues(`lines.${index}.${opposite}`) ?? 0) !== 0) {
        setValue(`lines.${index}.${opposite}`, 0, { shouldDirty: true });
      }
    },
    [getValues, setValue],
  );

  const handleRemove = useCallback(
    (index: number) => {
      remove(index);
      const lines = getValues("lines") ?? [];
      lines.forEach((_, lineIndex) => {
        setValue(`lines.${lineIndex}.lineNumber`, lineIndex + 1);
      });
    },
    [remove, getValues, setValue],
  );

  const handleAppend = useCallback(() => {
    append({
      glAccountId: "",
      description: "",
      debitAmount: 0,
      creditAmount: 0,
      lineNumber: fields.length + 1,
    } as ManualJournalLine);
  }, [append, fields.length]);

  return (
    <div className={cn("space-y-2", className)}>
      <div className="overflow-hidden rounded-lg border">
        <table className="w-full table-fixed text-sm">
          <thead className="bg-muted/50 text-left text-muted-foreground">
            <tr>
              <th className="w-8 px-2 py-2 text-center text-xs font-medium">#</th>
              <th className="px-2 py-2 text-xs font-medium">GL Account</th>
              <th className="px-2 py-2 text-xs font-medium">Memo</th>
              <th className="w-28 px-2 py-2 text-right text-xs font-medium">Debit</th>
              <th className="w-28 px-2 py-2 text-right text-xs font-medium">Credit</th>
              <th className="w-9 px-2 py-2" />
            </tr>
          </thead>
          <tbody>
            {fields.map((field, index) => (
              <tr key={field.id} className="group border-t align-top">
                <td className="px-2 py-1.5 pt-3 text-center font-mono text-2xs text-muted-foreground">
                  {index + 1}
                </td>
                <td className="px-1 py-1.5">
                  <GLAccountAutocompleteField
                    control={control}
                    name={`lines.${index}.glAccountId`}
                    placeholder="Select account"
                    triggerClassName="sm:h-8 text-xs"
                    clearable={false}
                  />
                </td>
                <td className="px-1 py-1.5">
                  <InputField
                    control={control}
                    name={`lines.${index}.description`}
                    placeholder="What is this line for?"
                    inputClassProps="text-xs h-8"
                  />
                </td>
                <td className="px-1 py-1.5">
                  <MoneyField
                    control={control}
                    name={`lines.${index}.debitAmount`}
                    aria-label={`Line ${index + 1} debit amount`}
                    className="h-8 text-xs"
                    onValueCommit={clearOpposite(index, "creditAmount")}
                  />
                </td>
                <td className="px-1 py-1.5">
                  <MoneyField
                    control={control}
                    name={`lines.${index}.creditAmount`}
                    aria-label={`Line ${index + 1} credit amount`}
                    className="h-8 text-xs"
                    onValueCommit={clearOpposite(index, "debitAmount")}
                  />
                </td>
                <td className="px-1 py-1.5 pt-4 justify-center flex items-center text-center">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xxs"
                    aria-label={`Remove line ${index + 1}`}
                    onClick={() => handleRemove(index)}
                    disabled={fields.length <= 2}
                    className="opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100 disabled:opacity-0"
                  >
                    <Trash2Icon className="size-3.5 text-muted-foreground hover:text-red-600" />
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot className="border-t bg-muted/30">
            <tr>
              <td colSpan={3} className="px-3 py-2 text-right text-xs font-medium">
                Totals
              </td>
              <td className="px-3 py-2 text-right">
                <AmountDisplay value={totalDebit} className="text-xs font-semibold" />
              </td>
              <td className="px-3 py-2 text-right">
                <AmountDisplay value={totalCredit} className="text-xs font-semibold" />
              </td>
              <td />
            </tr>
          </tfoot>
        </table>
      </div>

      <div className="flex items-center justify-between">
        <Button type="button" variant="outline" size="sm" onClick={handleAppend}>
          <PlusIcon className="mr-1.5 size-3.5" />
          Add Line
        </Button>

        {hasAmounts &&
          (isBalanced ? (
            <p className="flex items-center gap-1.5 text-xs font-medium text-green-600 dark:text-green-400">
              <CheckCircle2Icon className="size-3.5" />
              Balanced
            </p>
          ) : (
            <p className="text-xs font-medium text-red-600 dark:text-red-400">
              Out of balance by{" "}
              <AmountDisplay value={Math.abs(difference)} className="font-semibold" />
            </p>
          ))}
      </div>
    </div>
  );
}
