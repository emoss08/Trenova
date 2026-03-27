import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Checkbox } from "@/components/animate-ui/components/base/checkbox";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { queries } from "@/lib/queries";
import type { SelectOption } from "@/types/fields";
import type { TCASubscriptionFormValues } from "@/types/table-change-alert";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { Controller, useFormContext } from "react-hook-form";
import { ConditionBuilder } from "./condition-builder";

const PRIORITY_OPTIONS: SelectOption[] = [
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

export function SubscriptionForm() {
  const { control } = useFormContext<TCASubscriptionFormValues>();

  const { data: allowlistedTables } = useQuery(
    queries.tableChangeAlert.allowlistedTables(),
  );

  const tableOptions = useMemo<SelectOption[]>(
    () =>
      (allowlistedTables ?? []).map((t) => ({
        value: t.tableName,
        label: t.displayName,
      })),
    [allowlistedTables],
  );

  return (
    <>
      <FormGroup cols={2}>
        <FormControl>
          <InputField<TCASubscriptionFormValues>
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="My shipment alerts"
            description="A friendly name to identify this subscription."
          />
        </FormControl>
        <FormControl>
          <SelectField<TCASubscriptionFormValues>
            control={control}
            rules={{ required: true }}
            name="tableName"
            label="Table"
            options={tableOptions}
            placeholder="Select a table"
            description="The database table to monitor for changes."
          />
        </FormControl>
        <FormControl>
          <InputField<TCASubscriptionFormValues>
            control={control}
            name="recordId"
            label="Record ID"
            placeholder="Leave empty to watch all records"
            description="Optional. Specify a record ID to only watch a single record."
          />
        </FormControl>
        <FormControl>
          <SelectField<TCASubscriptionFormValues>
            control={control}
            name="priority"
            label="Priority"
            options={PRIORITY_OPTIONS}
            description="Notification priority level."
          />
        </FormControl>
        <FormControl cols="full">
          <Controller
            name="eventTypes"
            control={control}
            rules={{
              validate: (v) =>
                (v && v.length > 0) || "At least one event type is required",
            }}
            render={({ field, fieldState }) => (
              <div className="space-y-2">
                <Label className={fieldState.error ? "text-destructive" : ""}>
                  Event Types *
                </Label>
                <div className="flex gap-6">
                  {(["INSERT", "UPDATE", "DELETE"] as const).map((et) => (
                    <label key={et} className="flex items-center gap-2 text-sm">
                      <Checkbox
                        checked={field.value?.includes(et)}
                        onCheckedChange={(checked) => {
                          const current = field.value ?? [];
                          field.onChange(
                            checked
                              ? [...current, et]
                              : current.filter((v: string) => v !== et),
                          );
                        }}
                      />
                      {et.charAt(0) + et.slice(1).toLowerCase()}
                    </label>
                  ))}
                </div>
                {fieldState.error && (
                  <p className="text-2xs text-destructive">
                    {fieldState.error.message}
                  </p>
                )}
              </div>
            )}
          />
        </FormControl>
      </FormGroup>

      <FormSection title="Conditions" className="border-t py-2">
        <div className="space-y-4">
          <ConditionBuilder control={control} />
          <FormGroup cols={1}>
            <FormControl>
              <Controller
                name="watchedColumns"
                control={control}
                render={({ field }) => (
                  <InputField<TCASubscriptionFormValues>
                    control={control}
                    name="watchedColumns"
                    label="Watched Columns"
                    placeholder="e.g. status, eta, assigned_driver_id"
                    description="Comma-separated column names. Only trigger on UPDATE when these columns change. Leave empty to watch all."
                    value={(field.value ?? []).join(", ")}
                    onChange={(e) => {
                      const val = (e.target as HTMLInputElement).value;
                      field.onChange(
                        val
                          ? val.split(",").map((s) => s.trim()).filter(Boolean)
                          : [],
                      );
                    }}
                  />
                )}
              />
            </FormControl>
          </FormGroup>
        </div>
      </FormSection>

      <FormSection title="Notification" className="border-t py-2">
        <FormGroup cols={2}>
          <FormControl>
            <InputField<TCASubscriptionFormValues>
              control={control}
              name="topic"
              label="Topic"
              placeholder="e.g. shipment-delays"
              description="Optional categorization tag."
              maxLength={100}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField<TCASubscriptionFormValues>
              control={control}
              name="customTitle"
              label="Custom Title"
              placeholder="e.g. {{new.pro_number}} status changed"
              description="Available: {{table}}, {{operation}}, {{record_id}}, {{new.field}}, {{old.field}}, {{changed_fields}}"
              maxLength={500}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField<TCASubscriptionFormValues>
              control={control}
              name="customMessage"
              label="Custom Message"
              placeholder="e.g. Status changed from {{old.status}} to {{new.status}}"
              description="Leave empty to use auto-generated summary."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
  );
}
