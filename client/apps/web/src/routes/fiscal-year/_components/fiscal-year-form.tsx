import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { FISCAL_CALENDAR_TIMEZONE } from "@/lib/fiscal-calendar";
import type { FiscalPeriod } from "@/types/fiscal-period";
import type { FiscalYear } from "@/types/fiscal-year";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getUTCYearBounds } from "@trenova/shared/lib/date";
import { useCallback, useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { LazyLoadComponent } from "react-lazy-load-image-component";
import { FiscalPeriodTable } from "./fiscal-periods-table";

const NO_PERIODS: FiscalPeriod[] = [];

export function FiscalYearForm({ mode }: { mode: "create" | "edit" }) {
  const t = useT();

  const { control, setValue, getValues, getFieldState, reset, formState } =
    useFormContext<FiscalYear>();
  const isEdit = mode === "edit";

  const year = useWatch({ control, name: "year" });
  const isCalendarYear = useWatch({ control, name: "isCalendarYear" });
  const status = useWatch({ control, name: "status" });
  const periods = useWatch({ control, name: "periods" }) ?? NO_PERIODS;

  const isClosed = status === "Closed";
  const isPermanentlyClosed = status === "PermanentlyClosed";

  useEffect(() => {
    if (isEdit || !year) return;

    if (!getFieldState("name").isDirty) {
      setValue("name", `FY ${year}`);
    }

    if (isCalendarYear) {
      const { startDate, endDate } = getUTCYearBounds(year);
      setValue("startDate", startDate, { shouldValidate: true });
      setValue("endDate", endDate, { shouldValidate: true });
    }
  }, [isEdit, isCalendarYear, year, setValue, getFieldState]);

  const handlePeriodUpdated = useCallback(
    (updated: FiscalPeriod) => {
      const nextPeriods = (getValues("periods") ?? []).map((existing) =>
        existing.id === updated.id ? { ...existing, ...updated } : existing,
      );
      reset(
        { ...(formState.defaultValues as FiscalYear), periods: nextPeriods },
        { keepDirtyValues: true, keepErrors: true, keepTouched: true, keepSubmitCount: true },
      );
    },
    [getValues, reset, formState],
  );

  return (
    <div className="flex flex-col gap-6">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            isReadOnly
            label={t("Status")}
            placeholder={t("Select status")}
            description={t("Changes through the fiscal year actions")}
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
            readOnly={isEdit}
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
            readOnly={isEdit && isPermanentlyClosed}
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title={t("Date configuration")}
        description={
          isEdit
            ? t("The calendar is fixed once the fiscal year is created")
            : t("Define the fiscal period and calendar year settings")
        }
      >
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="isCalendarYear"
              label={t("Calendar year")}
              description={t("Standard Jan 1 - Dec 31 period (automatically sets dates)")}
              position="left"
              readOnly={isEdit}
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
              label={t("Start date")}
              placeholder={t("Select start date")}
              description={t("First day of fiscal period (UTC)")}
              timezone={FISCAL_CALENDAR_TIMEZONE}
              // readOnly={isEdit || isCalendarYear}
            />
          </FormControl>

          <FormControl>
            <AutoCompleteDateField
              rules={{ required: true }}
              control={control}
              name="endDate"
              label={t("End date")}
              placeholder={t("Select end date")}
              description={t("Last day of fiscal period (UTC)")}
              timezone={FISCAL_CALENDAR_TIMEZONE}
              // readOnly={isEdit || isCalendarYear}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Year-end settings")}
        description={t("Post-close adjustment configuration")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="allowAdjustingEntries"
              label={t("Allow adjusting entries")}
              description={t("Permit accounting adjustments after year-end close")}
              position="left"
              outlined
              readOnly={isEdit && isPermanentlyClosed}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      {isEdit && (
        <FormSection title={t("System settings")} description={t("Active fiscal year designation")}>
          <FormGroup cols={1}>
            <FormControl>
              <SwitchField
                control={control}
                name="isCurrent"
                label={t("Current fiscal year")}
                description={t(
                  "Active year for transaction posting (only one allowed per organization)",
                )}
                position="left"
                outlined
                readOnly
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      )}

      {isEdit && (
        <FormSection title={t("Fiscal periods")} description={t("Manage fiscal periods")}>
          <LazyLoadComponent>
            <FiscalPeriodTable
              periods={periods}
              fiscalYearStatus={status}
              onPeriodUpdated={handlePeriodUpdated}
            />
          </LazyLoadComponent>
        </FormSection>
      )}
    </div>
  );
}
