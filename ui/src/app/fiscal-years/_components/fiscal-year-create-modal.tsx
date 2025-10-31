import { FormCreateModal } from "@/components/ui/form-create-modal";
import { getCurrentYear, getEndOfYear, getStartOfYear } from "@/lib/date";
import {
  fiscalYearSchema,
  FiscalYearStatusSchema,
} from "@/lib/schemas/fiscal-year-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FiscalYearForm } from "./fiscal-year-form";

export function CreateFiscalYearModal({ open, onOpenChange }: TableSheetProps) {
  const currentYear = getCurrentYear();
  const startOfYear = getStartOfYear();
  const endOfYear = getEndOfYear();

  const form = useForm({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: {
      year: new Date().getFullYear(),
      status: FiscalYearStatusSchema.enum.Draft,
      adjustmentDeadline: undefined,
      budgetAmount: undefined,
      isCurrent: false,
      isCalendarYear: false,
      allowAdjustingEntries: false,
      name: `FY ${currentYear}`,
      description: "",
      startDate: startOfYear,
      endDate: endOfYear,
      taxYear: new Date().getFullYear(),
      closedAt: undefined,
      lockedAt: undefined,
      closedById: undefined,
      lockedById: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Fiscal Year"
      formComponent={<FiscalYearForm isCreate />}
      description="Add a new fiscal year for historical data or future planning."
      form={form}
      url="/fiscal-years/"
      queryKey="fiscal-year-list"
    />
  );
}
