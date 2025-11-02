import { AutoCompleteDateField } from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { getEndOfYear, getStartOfYear } from "@/lib/date";
import {
  FiscalYearSchema,
  FiscalYearStatusSchema,
} from "@/lib/schemas/fiscal-year-schema";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function FiscalYearForm({ isCreate = false }: { isCreate?: boolean }) {
  const { control, setValue } = useFormContext<FiscalYearSchema>();
  const year = useWatch({ control, name: "year" });
  const isCalendarYear = useWatch({ control, name: "isCalendarYear" });
  const status = useWatch({ control, name: "status" });

  const isDraft = status === FiscalYearStatusSchema.enum.Draft;
  const isClosed = status === FiscalYearStatusSchema.enum.Closed;
  const isLocked = status === FiscalYearStatusSchema.enum.Locked;

  useEffect(() => {
    if (isCreate && year) {
      setValue("taxYear", year);
    }
  }, [isCreate, year, setValue]);

  useEffect(() => {
    if (isCreate && isCalendarYear && year) {
      const startOfYear = getStartOfYear();
      const endOfYear = getEndOfYear();

      setValue("startDate", startOfYear);
      setValue("endDate", endOfYear);
    }
  }, [isCreate, isCalendarYear, year, setValue]);

  return (
    <div className="flex flex-col">
      {/* Basic Information */}
      <FormGroup cols={2} className="pb-2 border-b">
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            isReadOnly={isCreate}
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
            readOnly={!isCreate && (isClosed || isLocked)} // Read-only if closed/locked
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

      {/* Date Configuration */}
      <FormSection
        title="Date Configuration"
        description="Define the fiscal period and calendar year settings"
        className="py-2 border-b"
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="isCalendarYear"
              label="Calendar Year"
              description="Standard Jan 1 - Dec 31 period (automatically sets dates)"
              position="left"
              disabled={!isCreate && !isDraft}
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
              readOnly={!isCreate && !isDraft}
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
              readOnly={!isCreate && !isDraft}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      {/* Financial Planning */}
      <FormSection
        title="Financial Planning"
        description="Budget and tax reporting configuration"
        className="py-2 border-b"
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
              readOnly={!isCreate && (isClosed || isLocked)}
            />
          </FormControl>

          <FormControl>
            <NumberField
              control={control}
              name="taxYear"
              label="Tax Year"
              placeholder="2025"
              description="IRS tax reporting year (auto-synced)"
              readOnly={isCreate}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      {/* Year-End Settings */}
      <FormSection
        title="Year-End Settings"
        description="Post-close adjustment configuration"
        className={!isCreate ? "py-2 border-b" : "py-2"}
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
              readOnly={!isCreate && isLocked}
            />
          </FormControl>

          {!isCreate && (
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

      {/* System Settings (Edit Only) */}
      {!isCreate && (
        <FormSection
          title="System Settings"
          description="Active fiscal year designation"
          className="py-2"
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
    </div>
  );
}
