import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { rateTableLookupTypeChoices } from "@/lib/choices";
import type { RateTable } from "@/types/rate-table";
import { Plus, Table2, Trash2 } from "lucide-react";
import { useCallback } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const EMPTY_ENTRY = {
  matchKey: "",
  rangeMin: null,
  rangeMax: null,
  value: undefined,
  sortOrder: 0,
} as unknown as RateTable["entries"][number];

export function RateTableForm({ disabled }: { disabled?: boolean }) {
  const { control } = useFormContext<RateTable>();

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="active"
            label="Active"
            description="Toggles whether this rate table is available for formula lookups."
            outlined
            position="left"
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label="Name"
            placeholder="Fuel Surcharge Bands"
            rules={{ required: true }}
            maxLength={100}
            description="Human-friendly name shown in lists and pickers."
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="key"
            label="Key"
            placeholder="fuel_surcharge"
            rules={{ required: true }}
            maxLength={64}
            description="Identifier used by lookup() in formula expressions; letters, numbers, and underscores."
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="lookupType"
            label="Lookup Type"
            placeholder="Select Lookup Type"
            rules={{ required: true }}
            description="Exact matches entries by key; Range matches numeric values against min/max bands."
            options={rateTableLookupTypeChoices}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Briefly explain what this table is used for"
            description="Short context explaining what this table rates and how entries are maintained."
            disabled={disabled}
          />
        </FormControl>
      </FormGroup>
      <RateTableEntriesEditor disabled={disabled} />
    </div>
  );
}

function RateTableEntriesEditor({ disabled }: { disabled?: boolean }) {
  const {
    control,
    formState: { errors },
  } = useFormContext<RateTable>();
  const lookupType = useWatch({ control, name: "lookupType" });
  const { fields, append, remove } = useFieldArray({ control, name: "entries" });
  const isExact = lookupType === "Exact";

  const handleAdd = useCallback(() => {
    append({ ...EMPTY_ENTRY, sortOrder: fields.length });
  }, [append, fields.length]);

  const entriesError = errors.entries?.message ?? errors.entries?.root?.message;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <Table2 className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Entries</CardTitle>
            <p className="text-xs text-muted-foreground">
              {isExact
                ? "Each entry maps a match key to a value"
                : "Each entry maps a numeric band (min inclusive, max exclusive) to a value; leave max empty for open-ended"}
            </p>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleAdd}
          disabled={disabled}
          className="gap-1.5"
        >
          <Plus className="size-3.5" />
          Add
        </Button>
      </CardHeader>
      <CardContent className="p-4">
        {entriesError && <p className="mb-3 text-xs text-red-500">{entriesError}</p>}
        {fields.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <div className="flex size-12 items-center justify-center rounded-full bg-muted">
              <Table2 className="size-5 text-muted-foreground" />
            </div>
            <p className="mt-3 text-sm font-medium">No entries</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Add at least one entry so lookups can resolve a value
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleAdd}
              disabled={disabled}
              className="mt-4 gap-1.5"
            >
              <Plus className="size-3.5" />
              Add Entry
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {fields.map((field, index) => (
              <div
                key={field.id}
                className="group relative grid grid-cols-12 items-start gap-3 rounded-lg border bg-muted/30 p-3 transition-colors hover:bg-muted/50"
              >
                {isExact ? (
                  <div className="col-span-6">
                    <InputField
                      control={control}
                      name={`entries.${index}.matchKey`}
                      label="Match Key"
                      placeholder="CA"
                      rules={{ required: true }}
                      maxLength={100}
                      disabled={disabled}
                    />
                  </div>
                ) : (
                  <>
                    <div className="col-span-3">
                      <NumberField
                        control={control}
                        name={`entries.${index}.rangeMin`}
                        label="Range Min"
                        placeholder="0"
                        rules={{ required: true }}
                        decimalScale={4}
                        thousandSeparator
                        disabled={disabled}
                      />
                    </div>
                    <div className="col-span-3">
                      <NumberField
                        control={control}
                        name={`entries.${index}.rangeMax`}
                        label="Range Max"
                        placeholder="Open-ended"
                        decimalScale={4}
                        thousandSeparator
                        disabled={disabled}
                      />
                    </div>
                  </>
                )}
                <div className="col-span-5">
                  <NumberField
                    control={control}
                    name={`entries.${index}.value`}
                    label="Value"
                    placeholder="0.00"
                    rules={{ required: true }}
                    decimalScale={4}
                    thousandSeparator
                    sideText="$"
                    disabled={disabled}
                  />
                </div>
                <div className="col-span-1 flex justify-end pt-6">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => remove(index)}
                    disabled={disabled}
                    className="size-8 p-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive"
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
