import { AmountDisplay } from "@/components/accounting/amount-display";
import { BatchSourceAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";

export type ReceiptLineValues = {
  receiptDate: string;
  amount: string;
  referenceNumber: string;
  memo: string;
};

export type ImportBatchFormValues = {
  source: string;
  reference: string;
  receipts: ReceiptLineValues[];
};

export const EMPTY_LINE: ReceiptLineValues = {
  receiptDate: "",
  amount: "",
  referenceNumber: "",
  memo: "",
};

function toCents(dollarStr: string): number {
  return Math.round(parseFloat(dollarStr) * 100);
}

export function ImportBatchForm() {
  const { control } = useFormContext<ImportBatchFormValues>();

  const { fields, append, remove } = useFieldArray({
    control,
    name: "receipts",
  });

  const watchedReceipts = useWatch({ control, name: "receipts" });

  const totalCents = (watchedReceipts ?? []).reduce(
    (sum, line) => sum + (line?.amount ? toCents(line.amount) || 0 : 0),
    0,
  );

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title="Batch Information"
        description="Identify the source bank and a reference for this import"
      >
        <FormGroup cols={2}>
          <FormControl>
            <BatchSourceAutocompleteField
              control={control}
              name="source"
              label="Source"
              placeholder="e.g. Chase, Wells Fargo"
              rules={{ required: "Source is required" }}
              description="The bank or institution this batch originates from."
              clearable
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="reference"
              label="Reference"
              rules={{ required: "Reference is required" }}
              placeholder="e.g. Statement 2026-04"
              description="A unique identifier for this batch, such as a statement number."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Receipt Lines"
        description="Individual bank receipts to import in this batch"
        titleCount={fields.length}
        className="border-t border-border pt-4"
      >
        <div className="overflow-hidden rounded-md border">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left text-muted-foreground">
              <tr>
                <th className="w-10 px-3 py-2 text-xs font-medium">#</th>
                <th className="px-3 py-2 text-xs font-medium">Date</th>
                <th className="px-3 py-2 text-xs font-medium">Amount ($)</th>
                <th className="px-3 py-2 text-xs font-medium">Reference #</th>
                <th className="px-3 py-2 text-xs font-medium">Memo</th>
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
                    <Controller
                      control={control}
                      name={`receipts.${index}.receiptDate`}
                      rules={{ required: "Required" }}
                      render={({ field: f, fieldState }) => (
                        <Input
                          {...f}
                          type="date"
                          className={`h-8 text-xs ${fieldState.error ? "border-red-500" : ""}`}
                        />
                      )}
                    />
                  </td>
                  <td className="px-1.5 py-1.5">
                    <Controller
                      control={control}
                      name={`receipts.${index}.amount`}
                      rules={{
                        required: "Required",
                        validate: (v) => {
                          const num = parseFloat(v);
                          if (Number.isNaN(num) || num <= 0) return "Must be > 0";
                          return true;
                        },
                      }}
                      render={({ field: f, fieldState }) => (
                        <Input
                          {...f}
                          type="number"
                          step="0.01"
                          min="0.01"
                          placeholder="0.00"
                          className={`h-8 text-right text-xs tabular-nums ${fieldState.error ? "border-red-500" : ""}`}
                        />
                      )}
                    />
                  </td>
                  <td className="px-1.5 py-1.5">
                    <Controller
                      control={control}
                      name={`receipts.${index}.referenceNumber`}
                      rules={{ required: "Required" }}
                      render={({ field: f, fieldState }) => (
                        <Input
                          {...f}
                          placeholder="Check #, txn ID..."
                          className={`h-8 text-xs ${fieldState.error ? "border-red-500" : ""}`}
                        />
                      )}
                    />
                  </td>
                  <td className="px-1.5 py-1.5">
                    <Controller
                      control={control}
                      name={`receipts.${index}.memo`}
                      render={({ field: f }) => (
                        <Input {...f} placeholder="Optional" className="h-8 text-xs" />
                      )}
                    />
                  </td>
                  <td className="px-1.5 py-1.5">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-xxs"
                      onClick={() => remove(index)}
                      disabled={fields.length <= 1}
                    >
                      <Trash2Icon className="size-3.5 text-muted-foreground" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t bg-muted/30">
              <tr>
                <td colSpan={2} className="px-3 py-2 text-right text-xs font-medium">
                  Total
                </td>
                <td className="px-3 py-2 text-right">
                  <AmountDisplay value={totalCents} className="text-xs font-semibold" />
                </td>
                <td colSpan={3} />
              </tr>
            </tfoot>
          </table>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => append({ ...EMPTY_LINE })}
        >
          <PlusIcon className="mr-1.5 size-3.5" />
          Add Line
        </Button>
      </FormSection>
    </div>
  );
}
