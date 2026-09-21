import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import type { FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { fiscalYearSchema, type FiscalYear } from "@/types/fiscal-year";
import { zodResolver } from "@hookform/resolvers/zod";
import { useT } from "@trenova/shared/i18n/use-t";
import { getUTCYearBounds } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useCallback } from "react";
import { useForm, useWatch } from "react-hook-form";
import { FiscalYearForm } from "./fiscal-year-form";
import {
  FiscalYearLifecycleActions,
  FiscalYearStatusSummary,
} from "./fiscal-year-lifecycle-actions";

export function FiscalYearPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FiscalYearRow>) {
  const t = useT();

  const currentYear = new Date().getFullYear();
  const { startDate, endDate } = getUTCYearBounds(currentYear);

  const form = useForm({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: {
      status: "Draft" as const,
      year: currentYear,
      name: `FY ${currentYear}`,
      description: "",
      startDate,
      endDate,
      isCalendarYear: true,
      allowAdjustingEntries: false,
      isCurrent: false,
    },
    mode: "onChange",
  });

  const [id, year, yearEndDate, status, isCurrent] = useWatch({
    control: form.control,
    name: ["id", "year", "endDate", "status", "isCurrent"],
  });

  const { reset, formState } = form;
  const handleFiscalYearUpdated = useCallback(
    (updated: FiscalYear) => {
      reset(
        {
          ...(formState.defaultValues as FiscalYear),
          status: updated.status,
          isCurrent: updated.isCurrent,
          closedAt: updated.closedAt,
          closedById: updated.closedById,
          lockedAt: updated.lockedAt,
          lockedById: updated.lockedById,
          version: updated.version,
          updatedAt: updated.updatedAt,
        },
        { keepDirtyValues: true, keepErrors: true, keepTouched: true, keepSubmitCount: true },
      );
    },
    [reset, formState],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/fiscal-years/"
        queryKey="fiscal-year-list"
        title={t("Fiscal year")}
        fieldKey="name"
        formComponent={<FiscalYearForm mode="edit" />}
        size="lg"
        descriptionExtra={<FiscalYearStatusSummary status={status} isCurrent={isCurrent} />}
        headerActions={
          <FiscalYearLifecycleActions
            fiscalYear={{ id, year, endDate: yearEndDate, status, isCurrent }}
            onCompleted={handleFiscalYearUpdated}
          />
        }
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/fiscal-years/"
      queryKey="fiscal-year-list"
      title={t("Fiscal year")}
      formComponent={<FiscalYearForm mode="create" />}
    />
  );
}
