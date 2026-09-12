import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { getEndOfYear, getStartOfYear } from "@trenova/shared/lib/date";
import type { FiscalYear } from "@/types/fiscal-year";
import { useEffect } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { LazyLoadComponent } from "react-lazy-load-image-component";
import FiscalPeriodTable from "./fiscal-periods-table";

export function FiscalYearForm({ mode }: { mode: "create" | "edit" }) {
  const t = useT();

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
  const isPermanentlyClosed = status === "PermanentlyClosed";

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
            label={t("Status")}
            placeholder={t("Select status")}
            description={t("Current workflow status")}
            options={fiscalYearStatusChoices}
          />
        </FormControl>

        <FormControl>
          <NumberField
            control={control}
            rules={{ required: true }}
            name="year"
            label={t("Year")}
            placeholder="2025"
            description={t("Fiscal year identifier")}
            min={new Date().getFullYear() - 1}
            max={new Date().getFullYear() + 5}
          />
        </FormControl>

        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label={t("Name")}
            placeholder={t("FY 2025")}
            description={t("Display name for reports and references")}
            maxLength={100}
            readOnly={isEdit && (isClosed || isPermanentlyClosed)}
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label={t("Description")}
            placeholder={t("Optional notes about this fiscal year...")}
            description={t("Additional context or special notes")}
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title={t("Date Configuration")}
        description={t("Define the fiscal period and calendar year settings")}
        className="border-b py-2"
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="isCalendarYear"
              label={t("Calendar Year")}
              description={t("Standard Jan 1 - Dec 31 period (automatically sets dates)")}
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
              label={t("Start Date")}
              placeholder={t("Select start date")}
              description={t("First day of fiscal period")}
              readOnly={isEdit && !isDraft}
            />
          </FormControl>

          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="endDate"
              label={t("End Date")}
              placeholder={t("Select end date")}
              description={t("Last day of fiscal period")}
              readOnly={isEdit && !isDraft}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Financial Planning")}
        description={t("Budget and tax reporting configuration")}
        className="border-b py-2"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="budgetAmount"
              label={t("Budget Amount")}
              placeholder="0"
              description={t("Annual budget in dollars (optional)")}
              min={0}
              readOnly={isEdit && (isClosed || isPermanentlyClosed)}
            />
          </FormControl>

          <FormControl>
            <NumberField
              control={control}
              name="taxYear"
              label={t("Tax Year")}
              placeholder="2025"
              description={t("IRS tax reporting year (auto-synced)")}
              readOnly={!isEdit}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Year-End Settings")}
        description={t("Post-close adjustment configuration")}
        className={isEdit ? "border-b py-2" : "py-2"}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="allowAdjustingEntries"
              label={t("Allow Adjusting Entries")}
              description={t("Permit accounting adjustments after year-end close")}
              position="left"
              outlined
              readOnly={isEdit && isPermanentlyClosed}
            />
          </FormControl>

          {isEdit && (
            <FormControl cols="full">
              <AutoCompleteDateField
                control={control}
                name="adjustmentDeadline"
                label={t("Adjustment Deadline")}
                placeholder={t("Select deadline")}
                description={t("Final date for post-close adjusting entries")}
                readOnly={isPermanentlyClosed}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
      {isEdit && (
        <FormSection
          title={t("System Settings")}
          description={t("Active fiscal year designation")}
          className="border-b py-2"
        >
          <FormGroup cols={1}>
            <FormControl>
              <SwitchField
                control={control}
                name="isCurrent"
                label={t("Current Fiscal Year")}
                description={t(
                  "Active year for transaction posting (only one allowed per organization)",
                )}
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
          title={t("Fiscal Periods")}
          description={t("Manage fiscal periods")}
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
