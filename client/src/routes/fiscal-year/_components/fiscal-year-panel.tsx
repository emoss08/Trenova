import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { fiscalYearSchema, type FiscalYear } from "@/types/fiscal-year";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FiscalYearForm } from "./fiscal-year-form";

export function FiscalYearPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FiscalYear>) {
  const currentYear = new Date().getFullYear();

  const form = useForm({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: {
      status: "Draft" as const,
      year: currentYear,
      name: `FY ${currentYear}`,
      description: "",
      startDate: undefined as unknown as number,
      endDate: undefined as unknown as number,
      isCalendarYear: true,
      budgetAmount: null,
      taxYear: null,
      allowAdjustingEntries: false,
      adjustmentDeadline: null,
      isCurrent: false,
    },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/fiscal-years/"
        queryKey="fiscal-year-list"
        title="Fiscal Year"
        fieldKey="name"
        formComponent={<FiscalYearForm mode="edit" />}
        size="lg"
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
      title="Fiscal Year"
      formComponent={<FiscalYearForm mode="create" />}
    />
  );
}
