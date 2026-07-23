import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { getEndOfYear, getStartOfYear } from "@/lib/date";
import type { FiscalYear } from "@/types/fiscal-year";
import { useEffect } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { LazyLoadComponent } from "react-lazy-load-image-component";
import FiscalPeriodTable from "./fiscal-periods-table";

export function FiscalYearForm({ mode }: { mode: "create" | "edit" }) {
  const { control, setValue } = useFormContext<FiscalYear>();
  const { fields: periods } = useFieldArray({
    control,
    name: "periods",
  });
  const isEdit = mode === "edit";

  const year = useWatch({ control, name: "year" });
  const isCalendarYear = useWatch({ control, name: "isCalendarYear" });
  const status = useWatch({ control, name: "status" });

  const isDraft = status === "Draft";
  const isClosed = status === "Closed";
  const isLocked = status === "Locked";

  useEffect(() => {
    if (!isEdit && year) {
      setValue("taxYear", year);
    }
  }, [isEdit, year, setValue]);

  useEffect(() => {
    if (!isEdit && isCalendarYear && year) {
      const startOfYear = getStartOfYear();
      const endOfYear = getEndOfYear();

      setValue("startDate", startOfYear);
      setValue("endDate", endOfYear);
    }
  }, [isEdit, isCalendarYear, year, setValue]);

  return (
    <div className="flex flex-col">
      <FormGroup cols={2} className="border-b pb-2">
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            isReadOnly={!isEdit}
            label="Status"
            placeholder="Select status"
            description="Current workflow status"
            options={fiscalYearStatusChoices}
          />
        </FormControl>

        <FormControl>
          <NumberField
            control={control}
            rules={{ required: true }}
            name="year"
            label="Year"
            placeholder="2025"
            description="Fiscal year identifier"
            min={new Date().getFullYear() - 1}
            max={new Date().getFullYear() + 5}
          />
        </FormControl>

        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="FY 2025"
            description="Display name for reports and references"
            maxLength={100}
            readOnly={isEdit && (isClosed || isLocked)}
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Optional notes about this fiscal year..."
            description="Additional context or special notes"
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title="Date Configuration"
        description="Define the fiscal period and calendar year settings"
        className="border-b py-2"
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="isCalendarYear"
              label="Calendar Year"
              description="Standard Jan 1 - Dec 31 period (automatically sets dates)"
              position="left"
              disabled={isEdit && !isDraft}
              outlined
            />
          </FormControl>
        </FormGroup>

        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="startDate"
              label="Start Date"
              placeholder="Select start date"
              description="First day of fiscal period"
              readOnly={isEdit && !isDraft}
            />
          </FormControl>

          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="endDate"
              label="End Date"
              placeholder="Select end date"
              description="Last day of fiscal period"
              readOnly={isEdit && !isDraft}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Financial Planning"
        description="Budget and tax reporting configuration"
        className="border-b py-2"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="budgetAmount"
              label="Budget Amount"
              placeholder="0"
              description="Annual budget in dollars (optional)"
              min={0}
              readOnly={isEdit && (isClosed || isLocked)}
            />
          </FormControl>

          <FormControl>
            <NumberField
              control={control}
              name="taxYear"
              label="Tax Year"
              placeholder="2025"
              description="IRS tax reporting year (auto-synced)"
              readOnly={!isEdit}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Year-End Settings"
        description="Post-close adjustment configuration"
        className={isEdit ? "border-b py-2" : "py-2"}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="allowAdjustingEntries"
              label="Allow Adjusting Entries"
              description="Permit accounting adjustments after year-end close"
              position="left"
              outlined
              readOnly={isEdit && isLocked}
            />
          </FormControl>

          {isEdit && (
            <FormControl cols="full">
              <AutoCompleteDateField
                control={control}
                name="adjustmentDeadline"
                label="Adjustment Deadline"
                placeholder="Select deadline"
                description="Final date for post-close adjusting entries"
                readOnly={isLocked}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
      {isEdit && (
        <FormSection
          title="System Settings"
          description="Active fiscal year designation"
          className="border-b py-2"
        >
          <FormGroup cols={1}>
            <FormControl>
              <SwitchField
                control={control}
                name="isCurrent"
                label="Current Fiscal Year"
                description="Active year for transaction posting (only one allowed per organization)"
                position="left"
                outlined
                disabled
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      )}

      {isEdit && (
        <FormSection
          title="Fiscal Periods"
          description="Manage fiscal periods"
          className="py-2"
        >
          <LazyLoadComponent>
            <FiscalPeriodTable periods={periods} />
          </LazyLoadComponent>
        </FormSection>
      )}
    </div>
  );
}
