import { AmountDisplay } from "@/components/accounting/amount-display";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ManualJournalLine } from "@/types/manual-journal";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

type JournalLineItemsEditorProps = {
  className?: string;
};

export function JournalLineItemsEditor({ className }: JournalLineItemsEditorProps) {
  const { control } = useFormContext<{ lines: ManualJournalLine[] }>();
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
  const isBalanced = totalDebit === totalCredit && totalDebit > 0;

  return (
    <div className={cn("space-y-3", className)}>
      <div className="overflow-hidden rounded-md border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-muted-foreground">
            <tr>
              <th className="w-10 px-3 py-2 text-xs font-medium">#</th>
              <th className="px-3 py-2 text-xs font-medium">GL Account ID</th>
              <th className="px-3 py-2 text-xs font-medium">Description</th>
              <th className="w-40 px-3 py-2 text-right text-xs font-medium">Debit</th>
              <th className="w-40 px-3 py-2 text-right text-xs font-medium">Credit</th>
              <th className="w-10 px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            {fields.map((field, index) => (
              <tr key={field.id} className="border-t">
                <td className="px-3 py-1.5 font-mono text-2xs text-muted-foreground">
                  {index + 1}
                </td>
                <td className="px-1.5 py-1.5">
                  <InputField
                    control={control}
                    name={`lines.${index}.glAccountId`}
                    placeholder="Account ID"
                    className="h-8 text-xs"
                  />
                </td>
                <td className="px-1.5 py-1.5">
                  <InputField
                    control={control}
                    name={`lines.${index}.description`}
                    placeholder="Description"
                    className="h-8 text-xs"
                  />
                </td>
                <td className="px-1.5 py-1.5">
                  <NumberField
                    control={control}
                    name={`lines.${index}.debitAmount`}
                    placeholder="0"
                    className="h-8 text-right text-xs"
                  />
                </td>
                <td className="px-1.5 py-1.5">
                  <NumberField
                    control={control}
                    name={`lines.${index}.creditAmount`}
                    placeholder="0"
                    className="h-8 text-right text-xs"
                  />
                </td>
                <td className="px-1.5 py-1.5">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xxs"
                    onClick={() => remove(index)}
                    disabled={fields.length <= 2}
                  >
                    <Trash2Icon className="size-3.5 text-muted-foreground" />
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
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() =>
            append({
              glAccountId: "",
              description: "",
              debitAmount: 0,
              creditAmount: 0,
              lineNumber: fields.length + 1,
            } as ManualJournalLine)
          }
        >
          <PlusIcon className="mr-1.5 size-3.5" />
          Add Line
        </Button>

        {!isBalanced && totalDebit + totalCredit > 0 ? (
          <p className="text-xs font-medium text-red-600 dark:text-red-400">
            Out of balance by{" "}
            <AmountDisplay value={Math.abs(totalDebit - totalCredit)} className="font-semibold" />
          </p>
        ) : null}
      </div>
    </div>
  );
}
